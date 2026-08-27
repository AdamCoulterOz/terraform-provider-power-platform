// Extracts the provider's HTTP call sites from Go source, without type checking.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

var fset = token.NewFileSet()

func src(n ast.Node) string {
	if n == nil {
		return ""
	}
	var b bytes.Buffer
	printer.Fprint(&b, fset, n)
	return b.String()
}

var consts = map[string]string{
	"http.MethodGet":     "GET",
	"http.MethodPost":    "POST",
	"http.MethodPut":     "PUT",
	"http.MethodPatch":   "PATCH",
	"http.MethodDelete":  "DELETE",
	"http.MethodHead":    "HEAD",
	"http.MethodOptions": "OPTIONS",
}

type parsedFile struct {
	rel string
	pkg string
	f   *ast.File
}

var files []parsedFile

func loadAll(root string) {
	filepath.Walk(filepath.Join(root, "internal"), func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if strings.Contains(rel, "/mocks/") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse:", rel, err)
			return nil
		}
		files = append(files, parsedFile{rel: rel, pkg: f.Name.Name, f: f})
		return nil
	})
	for _, pf := range files {
		for _, d := range pf.f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, s := range gd.Specs {
				vs, ok := s.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					bl, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || bl.Kind != token.STRING {
						continue
					}
					v, err := strconv.Unquote(bl.Value)
					if err != nil {
						continue
					}
					consts[pf.pkg+"."+name.Name] = v
					if vs.Type != nil {
						tk := pf.pkg + "." + src(vs.Type)
						typedConsts[tk] = append(typedConsts[tk], v)
					}
					if _, dup := consts[name.Name]; !dup {
						consts[name.Name] = v
					}
				}
			}
		}
	}
}

func resolve(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			if s, err := strconv.Unquote(v.Value); err == nil {
				return s
			}
		}
		return v.Value
	case *ast.SelectorExpr:
		if s, ok := consts[src(v)]; ok {
			return s
		}
		return "${" + src(v) + "}"
	case *ast.Ident:
		if s, ok := consts[v.Name]; ok {
			return s
		}
		return "${" + v.Name + "}"
	case *ast.BinaryExpr:
		if v.Op == token.ADD {
			return resolve(v.X) + resolve(v.Y)
		}
	case *ast.CallExpr:
		if s, ok := renderSprintf(v); ok {
			return s
		}
		// a string(x) conversion carries no information of its own
		if id, ok := v.Fun.(*ast.Ident); ok && id.Name == "string" && len(v.Args) == 1 {
			return resolve(v.Args[0])
		}
	}
	return "${" + src(e) + "}"
}

var verbRe = regexp.MustCompile(`%[#+\-0-9.]*[a-zA-Z]`)

func renderSprintf(c *ast.CallExpr) (string, bool) {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != "fmt" || sel.Sel.Name != "Sprintf" || len(c.Args) == 0 {
		return "", false
	}
	format := resolve(c.Args[0])
	args := c.Args[1:]
	i := 0
	return verbRe.ReplaceAllStringFunc(format, func(verb string) string {
		if verb == "%%" || i >= len(args) {
			return verb
		}
		a := args[i]
		i++
		if v, ok := constValue(a); ok {
			return v
		}
		return "{" + src(a) + "}"
	}), true
}

// constValue reports the compile-time string value of an expression, if known.
func constValue(a ast.Expr) (string, bool) {
	if bl, ok := a.(*ast.BasicLit); ok && bl.Kind == token.STRING {
		if v, err := strconv.Unquote(bl.Value); err == nil {
			return v, true
		}
	}
	v, ok := consts[src(a)]
	return v, ok
}

// ---- URL builder functions -------------------------------------------------

type builder struct {
	queryParam string
	params     []string
	host       string
	path       string
	apiVer     string
	query      []string
	pending    *ast.CallExpr // unresolved delegation to another builder
	pkg        string
}

// typedConsts groups declared string constants by their named type, so a path
// segment typed as an enum expands to the values that type can hold.
var typedConsts = map[string][]string{}

// paramTypes maps a function's parameter names to their declared type names.
func paramTypes(ft *ast.FuncType) map[string]string {
	out := map[string]string{}
	if ft.Params == nil {
		return out
	}
	for _, fl := range ft.Params.List {
		t := src(fl.Type)
		for _, n := range fl.Names {
			out[n.Name] = t
		}
	}
	return out
}

var builders = map[string]*builder{}

// callArgs records, per package and callee name, the argument lists used at
// every in-package call site, so a path passed in as a parameter can be
// resolved back to the literals its callers supply.
type callRecord struct {
	args   []ast.Expr
	locals map[string][]string
}

var callArgs = map[string][]callRecord{}

func indexCallArgs() {
	for _, pf := range files {
		for _, d := range pf.f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			locals := stringLocals(fd.Body)
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				var nm string
				switch f := c.Fun.(type) {
				case *ast.Ident:
					nm = f.Name
				case *ast.SelectorExpr:
					nm = f.Sel.Name
				}
				if nm == "" {
					return true
				}
				k := pf.pkg + "." + nm
				callArgs[k] = append(callArgs[k], callRecord{args: c.Args, locals: locals})
				return true
			})
		}
	}
}

// expandParams resolves ${param} placeholders using the values in-package
// callers pass for that parameter. Returns every distinct result.
func expandParams(pkg string, fd *ast.FuncDecl, s string) []string {
	params := flatParams(fd.Type)
	out := []string{s}
	for i, p := range params {
		var next []string
		seen := map[string]bool{}
		for _, cand := range out {
			if !strings.Contains(cand, "${"+p+"}") {
				if !seen[cand] {
					seen[cand] = true
					next = append(next, cand)
				}
				continue
			}
			for _, rec := range callArgs[pkg+"."+fd.Name.Name] {
				if i >= len(rec.args) {
					continue
				}
				v, _ := expandLocals(resolve(rec.args[i]), rec.locals)
				if strings.Contains(v, "${") {
					continue
				}
				r := strings.ReplaceAll(cand, "${"+p+"}", v)
				if !seen[r] {
					seen[r] = true
					next = append(next, r)
				}
			}
			if len(next) == 0 {
				for _, v := range typedConsts[pkg+"."+paramTypes(fd.Type)[p]] {
					r := strings.ReplaceAll(cand, "${"+p+"}", v)
					if !seen[r] {
						seen[r] = true
						next = append(next, r)
					}
				}
			}
			if len(next) == 0 && !seen[cand] {
				seen[cand] = true
				next = append(next, cand)
			}
		}
		out = next
	}
	sort.Strings(out)
	return out
}

func flatParams(ft *ast.FuncType) []string {
	var out []string
	if ft.Params == nil {
		return out
	}
	for _, fl := range ft.Params.List {
		if len(fl.Names) == 0 {
			out = append(out, "_")
			continue
		}
		for _, n := range fl.Names {
			out = append(out, n.Name)
		}
	}
	return out
}

func returnsString(ft *ast.FuncType) bool {
	if ft.Results == nil || len(ft.Results.List) == 0 {
		return false
	}
	return src(ft.Results.List[0].Type) == "string"
}

func analyseFunc(pkg string, fd *ast.FuncDecl) *builder {
	b := &builder{params: flatParams(fd.Type), pkg: pkg}
	var lastURL string
	urls := map[string]*builder{}
	found := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			for i, rhs := range s.Rhs {
				if i >= len(s.Lhs) {
					break
				}
				e := rhs
				if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
					e = u.X
				}
				if cl, ok := e.(*ast.CompositeLit); ok && src(cl.Type) == "url.URL" {
					ub := &builder{}
					for _, el := range cl.Elts {
						kv, ok := el.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						switch src(kv.Key) {
						case "Host":
							ub.host = resolve(kv.Value)
						case "Path":
							ub.path = resolve(kv.Value)
						}
					}
					name := src(s.Lhs[i])
					urls[name] = ub
					lastURL = name
					found = true
				}
			}
			if len(s.Lhs) == 1 && len(s.Rhs) == 1 {
				if sel, ok := s.Lhs[0].(*ast.SelectorExpr); ok {
					if ub, ok := urls[src(sel.X)]; ok && sel.Sel.Name == "Path" {
						ub.path = resolve(s.Rhs[0])
					}
				}
			}
		case *ast.CallExpr:
			sel, ok := s.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Add" || len(s.Args) != 2 {
				return true
			}
			k := resolve(s.Args[0])
			v := resolve(s.Args[1])
			ub := urls[lastURL]
			if ub == nil {
				ub = b
			}
			if k == "api-version" {
				ub.apiVer = v
			} else {
				ub.query = append(ub.query, k+"="+v)
			}
		}
		return true
	})
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		sel, ok := as.Lhs[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "RawQuery" {
			return true
		}
		if v := encodeSource(as.Rhs[0]); v != "" {
			for _, p := range b.params {
				if p == v {
					b.queryParam = v
				}
			}
		}
		return true
	})
	if found && len(urls) == 1 {
		for _, ub := range urls {
			b.host, b.path, b.query = ub.host, ub.path, ub.query
			if ub.apiVer != "" {
				b.apiVer = ub.apiVer
			}
		}
		return b
	}
	// delegation: return SomeBuilder(...)
	var ret *ast.CallExpr
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if rs, ok := n.(*ast.ReturnStmt); ok && len(rs.Results) > 0 {
			if c, ok := rs.Results[0].(*ast.CallExpr); ok && ret == nil {
				ret = c
			}
		}
		return true
	})
	if ret != nil {
		b.pending = ret
		return b
	}
	return nil
}

func builderKeys(pkg string, fd *ast.FuncDecl) []string {
	if fd.Recv != nil {
		return []string{"M:" + pkg + "." + fd.Name.Name}
	}
	return []string{"F:" + pkg + "." + fd.Name.Name, "F:*." + fd.Name.Name}
}

func lookupBuilder(pkg string, fn ast.Expr) *builder {
	var name, qual string
	switch v := fn.(type) {
	case *ast.Ident:
		name = v.Name
	case *ast.SelectorExpr:
		name = v.Sel.Name
		if id, ok := v.X.(*ast.Ident); ok {
			qual = id.Name
		}
	default:
		return nil
	}
	for _, k := range []string{"M:" + pkg + "." + name, "F:" + qual + "." + name, "F:" + pkg + "." + name, "F:*." + name} {
		if b, ok := builders[k]; ok {
			return b
		}
	}
	return nil
}

// builderQueryArg returns the name of the url.Values variable the caller passes
// for a builder's query parameter.
func builderQueryArg(b *builder, args []ast.Expr) string {
	if b.queryParam == "" {
		return ""
	}
	for i, p := range b.params {
		if p != b.queryParam || i >= len(args) {
			continue
		}
		if id, ok := args[i].(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func substitute(b *builder, args []ast.Expr) (host, path, apiVer string, query []string) {
	host, path, apiVer, query = b.host, b.path, b.apiVer, append([]string{}, b.query...)
	for i, p := range b.params {
		if i >= len(args) || p == "_" {
			continue
		}
		val := resolve(args[i])
		host = strings.ReplaceAll(host, "${"+p+"}", val)
		path = strings.ReplaceAll(path, "${"+p+"}", val)
	}
	return
}

func resolveBuilders() {
	for range 5 {
		changed := false
		for _, b := range builders {
			if b.pending == nil {
				continue
			}
			target := lookupBuilder(b.pkg, b.pending.Fun)
			if target == nil || target.pending != nil {
				continue
			}
			h, p, av, q := substitute(target, b.pending.Args)
			b.host, b.path, b.query = h, p, q
			if b.queryParam == "" && target.queryParam != "" {
				for i, tp := range target.params {
					if tp != target.queryParam || i >= len(b.pending.Args) {
						continue
					}
					if id, ok := b.pending.Args[i].(*ast.Ident); ok {
						b.queryParam = id.Name
					}
				}
			}
			if av != "" {
				b.apiVer = av
			}
			b.pending = nil
			changed = true
		}
		if !changed {
			break
		}
	}
	for k, b := range builders {
		if b.pending != nil || (b.host == "" && b.path == "") {
			delete(builders, k)
		}
	}
}

// ---- call sites ------------------------------------------------------------

type local struct {
	host, path, apiVer string
	query              []string
	queryBound         bool      // query already taken from an explicit RawQuery binding
	pos                token.Pos // where this binding was assigned
}

// localSet holds every binding of a variable in source order, so a call site
// resolves against the assignment that precedes it rather than the last one.
type localSet map[string][]*local

func (ls localSet) bind(name string, l *local, pos token.Pos) {
	l.pos = pos
	ls[name] = append(ls[name], l)
}

// at returns the binding of name in effect at pos.
func (ls localSet) at(name string, pos token.Pos) *local {
	var best *local
	for _, l := range ls[name] {
		if l.pos <= pos && (best == nil || l.pos > best.pos) {
			best = l
		}
	}
	return best
}

// latest returns the most recent binding of name, for query attachment while
// the enclosing statement is still being walked.
func (ls localSet) latest(name string) *local {
	var best *local
	for _, l := range ls[name] {
		if best == nil || l.pos > best.pos {
			best = l
		}
	}
	return best
}

func (ls localSet) only() *local {
	var out *local
	n := 0
	for _, v := range ls {
		n += len(v)
		if len(v) > 0 {
			out = v[len(v)-1]
		}
	}
	if n == 1 {
		return out
	}
	return nil
}

type Site struct {
	Package      string   `json:"package"`
	File         string   `json:"file"`
	Line         int      `json:"line"`
	Recv         string   `json:"recv,omitempty"`
	Func         string   `json:"func"`
	Method       string   `json:"http_method"`
	Boundary     string   `json:"boundary"`
	Host         string   `json:"host"`
	Path         string   `json:"path"`
	PathVariants []string `json:"path_variants,omitempty"`
	APIVersion   string   `json:"api_version,omitempty"`
	Query        []string `json:"query,omitempty"`
	Statuses     []string `json:"statuses,omitempty"`
	Retry        bool     `json:"retry"`
	Scopes       string   `json:"scopes,omitempty"`
	URLExpr      string   `json:"url_expr,omitempty"`
	Dynamic      string   `json:"dynamic,omitempty"`
	RuntimePath  bool     `json:"runtime_path,omitempty"`
	Approximate  bool     `json:"approximate,omitempty"`
}

func classify(host string) string {
	switch {
	case strings.Contains(host, "Urls.BapiUrl"):
		return "bapi"
	case strings.Contains(host, "BuildEnvironmentHostUri"):
		return "ppapi-environment"
	case strings.Contains(host, "BuildTenantHostUri"):
		return "ppapi-tenant"
	case strings.Contains(host, "locationHeader"):
		return "async-poll"
	case strings.Contains(host, "analyticsUrl"):
		return "analytics"
	case strings.Contains(host, "rulesBaseUrl"), strings.Contains(host, "advisorURL"), strings.Contains(host, "powerAppsAdvisorUrl"):
		return "advisor"
	case strings.Contains(host, ".tenant.${client.Api.GetConfig().Urls.PowerPlatformUrl}"), strings.Contains(host, ".tenant.${client.Api.Config.Urls.PowerPlatformUrl}"):
		return "ppapi-tenant"
	case strings.Contains(host, ".environment.${client.Api.GetConfig().Urls.PowerPlatformUrl}"):
		return "ppapi-environment"
	case strings.Contains(host, "Urls.PowerPlatformUrl"):
		return "ppapi"
	case strings.Contains(host, "Urls.AdminPowerPlatformUrl"):
		return "admin"
	case strings.Contains(host, "Urls.PowerAppsUrl"):
		return "powerapps"
	case strings.Contains(host, "environmentHost"), strings.Contains(host, "EnvironmentHost"):
		return "dataverse"
	case strings.Contains(host, "advisor"), strings.Contains(host, "Advisor"):
		return "advisor"
	case strings.Contains(host, "copilot"), strings.Contains(host, "Copilot"):
		return "copilot"
	case strings.Contains(host, "analytics"), strings.Contains(host, "Analytics"):
		return "analytics"
	case strings.Contains(host, "licensing"), strings.Contains(host, "Licensing"):
		return "licensing"
	case host == "":
		return "unresolved"
	}
	return "other"
}

// parseAbs splits a fully rendered absolute URL template into host, path and query.
func parseAbs(s string) *local {
	if !strings.HasPrefix(s, "https://") {
		return nil
	}
	rest := strings.TrimPrefix(s, "https://")
	host, path := rest, ""
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		host, path = rest[:i], rest[i:]
	}
	l := &local{host: host}
	if i := strings.IndexByte(path, '?'); i >= 0 {
		q := path[i+1:]
		path = path[:i]
		for _, kv := range strings.Split(q, "&") {
			if strings.HasPrefix(kv, "api-version=") {
				l.apiVer = strings.TrimPrefix(kv, "api-version=")
			} else if kv != "" {
				l.query = append(l.query, kv)
			}
		}
	}
	l.path = path
	return l
}

// queryVars collects the key/value pairs added to each url.Values variable in a
// function body, independent of where the variable is later consumed.
func queryVars(body *ast.BlockStmt) map[string][][2]string {
	out := map[string][][2]string{}
	ast.Inspect(body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Add" || len(c.Args) != 2 {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		out[recv.Name] = append(out[recv.Name], [2]string{resolve(c.Args[0]), resolve(c.Args[1])})
		return true
	})
	return out
}

// literalValues reads an inline url.Values{...}.Encode() expression.
func literalValues(e ast.Expr) ([][2]string, bool) {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Encode" {
		return nil, false
	}
	cl, ok := sel.X.(*ast.CompositeLit)
	if !ok || src(cl.Type) != "url.Values" {
		return nil, false
	}
	var out [][2]string
	for _, el := range cl.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		k := resolve(kv.Key)
		v := ""
		if vcl, ok := kv.Value.(*ast.CompositeLit); ok && len(vcl.Elts) > 0 {
			v = resolve(vcl.Elts[0])
		} else {
			v = resolve(kv.Value)
		}
		out = append(out, [2]string{k, v})
	}
	return out, true
}

// encodeSource returns the url.Values variable behind an expression of the form
// values.Encode().
func encodeSource(e ast.Expr) string {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Encode" {
		return ""
	}
	if id, ok := sel.X.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func applyQuery(l *local, pairs [][2]string) {
	for _, kv := range pairs {
		if kv[0] == "api-version" {
			l.apiVer = kv[1]
		} else {
			l.query = append(l.query, kv[0]+"="+kv[1])
		}
	}
}

// stringLocals resolves simple string-valued local variables in a function
// body. A variable reassigned on a branch yields several values, all of which
// are real, so they are all kept.
func stringLocals(body *ast.BlockStmt) map[string][]string {
	out := map[string][]string{}
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range as.Rhs {
			if i >= len(as.Lhs) {
				break
			}
			switch rhs.(type) {
			case *ast.BasicLit, *ast.BinaryExpr, *ast.CallExpr:
				name := src(as.Lhs[i])
				v := resolve(rhs)
				if v == "" {
					continue
				}
				// x = f(x) refers to the previous binding; expand it against
				// what is already known rather than dropping it.
				var vals []string
				if strings.Contains(v, "${"+name+"}") {
					for _, prev := range out[name] {
						vals = append(vals, strings.ReplaceAll(v, "${"+name+"}", prev))
					}
				} else {
					vals = []string{v}
				}
				for _, nv := range vals {
					if !slices.Contains(out[name], nv) {
						out[name] = append(out[name], nv)
					}
				}
			}
		}
		return true
	})
	return out
}

// expandLocals substitutes a local string variable into a path template, in
// either the form an unresolved reference takes (${name}) or the form a format
// argument takes ({name}).
//
// Only the first binding is used. A variable reassigned on a branch cannot be
// resolved by this analysis without tracking which branch ran, so the result is
// reported as approximate rather than guessed at.
func expandLocals(s string, strLocals map[string][]string) (string, bool) {
	approx := false
	for range 3 {
		before := s
		for name, vs := range strLocals {
			if len(vs) == 0 {
				continue
			}
			for _, form := range []string{"${" + name + "}", "{" + name + "}"} {
				if !strings.Contains(s, form) || vs[0] == s {
					continue
				}
				if len(vs) > 1 {
					approx = true
				}
				s = strings.ReplaceAll(s, form, vs[0])
			}
		}
		if s == before {
			break
		}
	}
	return s, approx
}

var odataKeyRe = regexp.MustCompile(`\([^)]*\)`)
var versionSegRe = regexp.MustCompile(`^v[0-9]+(\.[0-9]+)?$`)

// runtimePath reports whether a path template names a resource whose *name* -
// an entity set or a navigation property - is only known at run time, as
// opposed to one whose identifier is a parameter.
//
// The two API styles in this codebase spell that differently. OData addresses
// keys inside parentheses, so a bare placeholder segment is a name. The REST
// APIs alternate literal collection and identifier, so a bare placeholder
// segment is an id. A path with a runtime name cannot correspond to any single
// catalogue operation; a path with a runtime id can.
func runtimePath(p string) bool {
	if p == "" {
		return false
	}
	if !strings.Contains(p, "/api/data/") {
		return false
	}
	for _, seg := range strings.Split(strings.TrimPrefix(p, "/"), "/") {
		bare := odataKeyRe.ReplaceAllString(seg, "")
		if bare == "" || versionSegRe.MatchString(bare) {
			continue
		}
		if strings.HasPrefix(bare, "{") && strings.HasSuffix(bare, "}") {
			return true
		}
	}
	return false
}

// normalisePlaceholders renders every unresolved expression as {expr}, so a
// path template has one placeholder spelling regardless of how it was built.
func normalisePlaceholders(s string) string {
	return strings.NewReplacer("${", "{").Replace(s)
}

func recvType(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	return strings.TrimPrefix(src(fd.Recv.List[0].Type), "*")
}

func main() {
	root := "../.."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	// paths reported by the parser are absolute, so the base has to be too
	root, absErr := filepath.Abs(root)
	if absErr != nil {
		panic(absErr)
	}
	loadAll(root)

	for _, pf := range files {
		for _, d := range pf.f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || !returnsString(fd.Type) {
				continue
			}
			if b := analyseFunc(pf.pkg, fd); b != nil {
				for _, k := range builderKeys(pf.pkg, fd) {
					if _, dup := builders[k]; !dup {
						builders[k] = b
					}
				}
			}
		}
	}
	resolveBuilders()
	indexCallArgs()

	var sites []Site
	for _, pf := range files {
		for _, d := range pf.f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			locals := localSet{}
			strLocals := stringLocals(fd.Body)
			qvars := queryVars(fd.Body)
			var lastURL string

			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch s := n.(type) {
				case *ast.AssignStmt:
					for i, rhs := range s.Rhs {
						if i >= len(s.Lhs) {
							break
						}
						name := src(s.Lhs[i])
						e := rhs
						if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
							e = u.X
						}
						if cl, ok := e.(*ast.CompositeLit); ok && src(cl.Type) == "url.URL" {
							l := &local{}
							for _, el := range cl.Elts {
								kv, ok := el.(*ast.KeyValueExpr)
								if !ok {
									continue
								}
								switch src(kv.Key) {
								case "Host":
									l.host = resolve(kv.Value)
								case "Path":
									l.path = resolve(kv.Value)
								case "RawQuery":
									if v := encodeSource(kv.Value); v != "" {
										applyQuery(l, qvars[v])
										l.queryBound = true
									} else if pairs, ok := literalValues(kv.Value); ok {
										applyQuery(l, pairs)
										l.queryBound = true
									}
								}
							}
							locals.bind(name, l, s.Pos())
							lastURL = name
							continue
						}
						if l := parseAbs(resolve(e)); l != nil {
							locals.bind(name, l, s.Pos())
							lastURL = name
							continue
						}
						if c, ok := e.(*ast.CallExpr); ok {
							if b := lookupBuilder(pf.pkg, c.Fun); b != nil {
								h, p, av, q := substitute(b, c.Args)
								l := &local{host: h, path: p, apiVer: av, query: q}
								if qn := builderQueryArg(b, c.Args); qn != "" {
									applyQuery(l, qvars[qn])
									l.queryBound = true
								}
								locals.bind(name, l, s.Pos())
								lastURL = name
								continue
							}
							if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Parse" && src(sel.X) == "url" && len(c.Args) == 1 {
								if prev := locals.latest(src(c.Args[0])); prev != nil {
									cp := *prev
									locals.bind(name, &cp, s.Pos())
								} else if pl := parseAbs(resolve(c.Args[0])); pl != nil {
									locals.bind(name, pl, s.Pos())
								} else {
									locals.bind(name, &local{host: resolve(c.Args[0])}, s.Pos())
								}
								lastURL = name
								continue
							}
						}
					}
					if len(s.Lhs) == 1 && len(s.Rhs) == 1 {
						if sel, ok := s.Lhs[0].(*ast.SelectorExpr); ok {
							l := locals.latest(src(sel.X))
							ok := l != nil
							if ok && sel.Sel.Name == "Path" {
								l.path = resolve(s.Rhs[0])
							}
							if ok && sel.Sel.Name == "RawQuery" {
								if v := encodeSource(s.Rhs[0]); v != "" {
									applyQuery(l, qvars[v])
									l.queryBound = true
								}
							}
						}
					}
				case *ast.CallExpr:
					sel, ok := s.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "Add" || len(s.Args) != 2 {
						return true
					}
					k := resolve(s.Args[0])
					v := resolve(s.Args[1])
					l := locals.latest(lastURL)
					if l == nil || l.queryBound {
						return true
					}
					if k == "api-version" {
						l.apiVer = v
					} else {
						l.query = append(l.query, k+"="+v)
					}
				}
				return true
			})

			ast.Inspect(fd.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := c.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				name := sel.Sel.Name
				if name != "Execute" && name != "ExecuteWithoutRetry" && name != "ExecuteForGivenScopes" {
					return true
				}
				if len(c.Args) < 4 {
					return true
				}
				st := Site{
					Package: pf.pkg, File: pf.rel, Line: fset.Position(c.Pos()).Line,
					Recv: recvType(fd), Func: fd.Name.Name,
					Method: resolve(c.Args[2]), Retry: name == "Execute",
					URLExpr: src(c.Args[3]),
				}
				if s := src(c.Args[1]); s != "nil" {
					st.Scopes = s
				}
				if len(c.Args) >= 7 {
					if cl, ok := c.Args[6].(*ast.CompositeLit); ok {
						for _, el := range cl.Elts {
							st.Statuses = append(st.Statuses, src(el))
						}
					}
				}
				ue := c.Args[3]
				base := src(ue)
				if cc, ok := ue.(*ast.CallExpr); ok {
					if s2, ok := cc.Fun.(*ast.SelectorExpr); ok && s2.Sel.Name == "String" {
						base = src(s2.X)
					} else if b := lookupBuilder(pf.pkg, cc.Fun); b != nil {
						h, p, av, q := substitute(b, cc.Args)
						st.Host, st.Path, st.APIVersion, st.Query = h, p, av, q
						if qn := builderQueryArg(b, cc.Args); qn != "" {
							l := &local{apiVer: av, query: q}
							applyQuery(l, qvars[qn])
							st.APIVersion, st.Query = l.apiVer, l.query
						}
					}
				}
				if st.Host == "" && st.Path == "" {
					if l := locals.at(base, c.Pos()); l != nil {
						st.Host, st.Path, st.APIVersion, st.Query = l.host, l.path, l.apiVer, l.query
					} else if l := locals.only(); l != nil {
						st.Host, st.Path, st.APIVersion, st.Query = l.host, l.path, l.apiVer, l.query
					}
				}
				if st.Host == "" && st.Path == "" {
					lower := strings.ToLower(base)
					switch {
					case strings.Contains(lower, "location"):
						st.Dynamic = "follows Location/Operation-Location header from a prior response"
					default:
						st.Dynamic = "url supplied at runtime: " + base
					}
				}
				var approx bool
				st.Host, _ = expandLocals(st.Host, strLocals)
				st.Method, _ = expandLocals(st.Method, strLocals)
				if vs, ok := strLocals[strings.Trim(st.Method, "${}")]; ok && len(vs) > 1 {
					approx = true
				}
				var variants, unresolved []string
				seenV := map[string]bool{}
				{
					raw, a := expandLocals(st.Path, strLocals)
					if a {
						approx = true
					}
					for _, v := range expandParams(pf.pkg, fd, raw) {
						// a surviving ${x} is a reference this analysis could
						// not follow, not a path parameter; prefer any variant
						// that resolved fully.
						dangling := strings.Contains(v, "${")
						v = normalisePlaceholders(v)
						if seenV[v] {
							continue
						}
						seenV[v] = true
						if dangling {
							unresolved = append(unresolved, v)
						} else {
							variants = append(variants, v)
						}
					}
				}
				if len(variants) == 0 {
					variants = unresolved
				}
				st.Approximate = approx
				sort.Strings(variants)
				st.Path = variants[0]
				if len(variants) > 1 {
					st.PathVariants = variants
				}
				st.RuntimePath = runtimePath(st.Path)
				st.Boundary = classify(st.Host)
				sites = append(sites, st)
				return true
			})
		}
	}

	sort.Slice(sites, func(i, j int) bool {
		if sites[i].File != sites[j].File {
			return sites[i].File < sites[j].File
		}
		return sites[i].Line < sites[j].Line
	})
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(sites)
	fmt.Fprintf(os.Stderr, "sites: %d builders: %d\n", len(sites), len(builders))
}
