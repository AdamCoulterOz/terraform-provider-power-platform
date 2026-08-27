// Type-checked call graph from provider resources/datasources to api.Client HTTP calls.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

var execMethods = map[string]bool{"Execute": true, "ExecuteWithoutRetry": true, "ExecuteForGivenScopes": true}

type edge struct {
	callee *types.Func
	site   string // non-empty when this is an HTTP call site
	cond   bool   // call sits inside a conditional/loop body, so it is not on every execution
}

type fnInfo struct {
	fn    *types.Func
	decl  *ast.FuncDecl
	pkg   *packages.Package
	calls []edge
	iface []string // interface method names called, unresolved
}

type Op struct {
	Entrypoint  string   `json:"entrypoint"`
	Site        string   `json:"site"`
	Path        []string `json:"call_path"`
	Conditional bool     `json:"conditional"`
	ViaIface    bool     `json:"via_interface,omitempty"`
}

type Artifact struct {
	Address string `json:"address"`
	Kind    string `json:"kind"`
	Package string `json:"package"`
	GoType  string `json:"go_type"`
	Source  struct {
		Path string `json:"path"`
		Line int    `json:"line"`
	} `json:"source"`
	Ops []Op `json:"operations"`
}

var frameworkMethods = map[string]bool{
	"Create": true, "Read": true, "Update": true, "Delete": true,
	"ImportState": true, "ModifyPlan": true, "ValidateConfig": true,
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
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Dir:   root,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, "./internal/...")
	if err != nil {
		panic(err)
	}
	nerr := 0
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		nerr += len(p.Errors)
	})
	fmt.Fprintf(os.Stderr, "packages: %d, errors: %d\n", len(pkgs), nerr)

	fns := map[*types.Func]*fnInfo{}
	fset := pkgs[0].Fset

	// index every function/method declared in the corpus
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}
				obj := p.TypesInfo.Defs[fd.Name]
				fn, ok := obj.(*types.Func)
				if !ok {
					continue
				}
				fns[fn] = &fnInfo{fn: fn, decl: fd, pkg: p}
			}
		}
	}

	// methods by name, for interface dispatch fallback
	byName := map[string][]*types.Func{}
	for fn := range fns {
		byName[fn.Name()] = append(byName[fn.Name()], fn)
	}

	for _, fi := range fns {
		condRanges := conditionalRanges(fi.decl.Body)
		ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			isCond := inConditional(condRanges, call.Pos())
			callee := typeutil.Callee(fi.pkg.TypesInfo, call)
			cf, ok := callee.(*types.Func)
			if !ok {
				return true
			}
			if execMethods[cf.Name()] && strings.HasSuffix(cf.Pkg().Path(), "/internal/api") {
				pos := fset.Position(call.Pos())
				rel, _ := filepath.Rel(root, pos.Filename)
				fi.calls = append(fi.calls, edge{site: rel + ":" + itoa(pos.Line), cond: isCond})
				return true
			}
			if _, known := fns[cf]; known {
				fi.calls = append(fi.calls, edge{callee: cf, cond: isCond})
				return true
			}
			// interface method: dispatch by name to same-named concrete methods
			if cf.Type() != nil {
				if sig, ok := cf.Type().(*types.Signature); ok && sig.Recv() != nil {
					if _, isIface := sig.Recv().Type().Underlying().(*types.Interface); isIface {
						fi.iface = append(fi.iface, cf.Name())
					}
				}
			}
			return true
		})
	}

	// Artifacts come from the provider's own registration lists, which are authoritative.
	methodsOf := map[*types.TypeName]map[string]*fnInfo{}
	for fn, fi := range fns {
		sig, ok := fn.Type().(*types.Signature)
		if !ok || sig.Recv() == nil {
			continue
		}
		t := sig.Recv().Type()
		if pt, ok := t.(*types.Pointer); ok {
			t = pt.Elem()
		}
		nt, ok := t.(*types.Named)
		if !ok {
			continue
		}
		if methodsOf[nt.Obj()] == nil {
			methodsOf[nt.Obj()] = map[string]*fnInfo{}
		}
		methodsOf[nt.Obj()][fn.Name()] = fi
	}

	var arts []Artifact
	for _, p := range pkgs {
		if p.Name != "provider" {
			continue
		}
		for _, f := range p.Syntax {
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}
				var kind string
				switch fd.Name.Name {
				case "Resources":
					kind = "resource"
				case "DataSources":
					kind = "datasource"
				default:
					continue
				}
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					cf, ok := typeutil.Callee(p.TypesInfo, call).(*types.Func)
					if !ok {
						return true
					}
					ctor := fns[cf]
					if ctor == nil {
						return true
					}
					nt, tn := ctorTarget(ctor)
					if nt == nil || tn == "" {
						fmt.Fprintf(os.Stderr, "unresolved constructor: %s\n", cf.FullName())
						return true
					}
					a := Artifact{Address: "powerplatform_" + tn, Kind: kind,
						Package: nt.Pkg().Name(), GoType: nt.Name()}
					// point at the registered constructor rather than the type
					// declaration: it is what the provider registers and it
					// always sits in the component's own file.
					pos := fset.Position(ctor.decl.Pos())
					rel, _ := filepath.Rel(root, pos.Filename)
					a.Source.Path = rel
					a.Source.Line = pos.Line
					for name, fi := range methodsOf[nt] {
						if !frameworkMethods[name] {
							continue
						}
						for _, op := range reach(fi, fns, byName) {
							op.Entrypoint = name
							a.Ops = append(a.Ops, op)
						}
					}
					sort.Slice(a.Ops, func(i, j int) bool {
						if a.Ops[i].Entrypoint != a.Ops[j].Entrypoint {
							return a.Ops[i].Entrypoint < a.Ops[j].Entrypoint
						}
						return a.Ops[i].Site < a.Ops[j].Site
					})
					arts = append(arts, a)
					return true
				})
			}
		}
	}
	sort.Slice(arts, func(i, j int) bool {
		if arts[i].Address != arts[j].Address {
			return arts[i].Address < arts[j].Address
		}
		return arts[i].Kind < arts[j].Kind
	})
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(arts)
	fmt.Fprintf(os.Stderr, "artifacts: %d funcs: %d\n", len(arts), len(fns))
}

func reach(start *fnInfo, fns map[*types.Func]*fnInfo, byName map[string][]*types.Func) []Op {
	type qi struct {
		fi    *fnInfo
		path  []string
		iface bool
		cond  bool
	}
	var out []Op
	best := map[string]int{} // site -> index in out
	// visited keyed by (func, cond) so an unconditional route is still explored
	// after a conditional one reached the same function.
	type vk struct {
		f *types.Func
		c bool
	}
	visited := map[vk]bool{{start.fn, false}: true}
	q := []qi{{fi: start, path: []string{label(start.fn)}}}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, e := range cur.fi.calls {
			cond := cur.cond || e.cond
			if e.site != "" {
				if i, ok := best[e.site]; ok {
					if out[i].Conditional && !cond {
						out[i] = Op{Site: e.site, Path: cur.path, ViaIface: cur.iface, Conditional: cond}
					}
					continue
				}
				best[e.site] = len(out)
				out = append(out, Op{Site: e.site, Path: cur.path, ViaIface: cur.iface, Conditional: cond})
				continue
			}
			ci := fns[e.callee]
			if ci == nil || visited[vk{e.callee, cond}] {
				continue
			}
			visited[vk{e.callee, cond}] = true
			q = append(q, qi{fi: ci, path: append(append([]string{}, cur.path...), label(e.callee)), iface: cur.iface, cond: cond})
		}
		for _, nm := range cur.fi.iface {
			for _, cand := range byName[nm] {
				if visited[vk{cand, cur.cond}] {
					continue
				}
				ci := fns[cand]
				if ci == nil {
					continue
				}
				visited[vk{cand, cur.cond}] = true
				q = append(q, qi{fi: ci, path: append(append([]string{}, cur.path...), label(cand)+" (via interface)"), iface: true, cond: cur.cond})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Site < out[j].Site })
	return out
}

func label(fn *types.Func) string {
	pkg := ""
	if fn.Pkg() != nil {
		pkg = fn.Pkg().Name()
	}
	recv := ""
	if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
		recv = types.TypeString(sig.Recv().Type(), func(p *types.Package) string { return "" })
		recv = strings.TrimPrefix(strings.TrimPrefix(recv, "*"), ".")
		if i := strings.LastIndex(recv, "."); i >= 0 {
			recv = recv[i+1:]
		}
		recv = "(" + recv + ")."
	}
	return pkg + "." + recv + fn.Name()
}

// ctorTarget finds the concrete type a New*Resource/New*DataSource constructor
// returns, and the TypeName it stamps into helpers.TypeInfo.
func ctorTarget(ctor *fnInfo) (*types.TypeName, string) {
	var nt *types.TypeName
	tn := ""
	ast.Inspect(ctor.decl.Body, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		t := ctor.pkg.TypesInfo.TypeOf(cl)
		if named, ok := t.(*types.Named); ok && nt == nil {
			if _, isStruct := named.Underlying().(*types.Struct); isStruct && named.Obj().Name() != "TypeInfo" {
				nt = named.Obj()
			}
		}
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			id, ok := kv.Key.(*ast.Ident)
			if !ok || id.Name != "TypeName" {
				continue
			}
			if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING && tn == "" {
				tn = strings.Trim(bl.Value, `"`)
			}
		}
		return true
	})
	return nt, tn
}

type posRange struct{ lo, hi token.Pos }

// conditionalRanges returns the source spans of bodies that do not execute
// unconditionally: if/else arms, loop bodies, switch and select clauses.
func conditionalRanges(body *ast.BlockStmt) []posRange {
	var out []posRange
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.IfStmt:
			out = append(out, posRange{s.Body.Pos(), s.Body.End()})
			if s.Else != nil {
				out = append(out, posRange{s.Else.Pos(), s.Else.End()})
			}
		case *ast.ForStmt:
			out = append(out, posRange{s.Body.Pos(), s.Body.End()})
		case *ast.RangeStmt:
			out = append(out, posRange{s.Body.Pos(), s.Body.End()})
		case *ast.CaseClause:
			out = append(out, posRange{s.Pos(), s.End()})
		case *ast.CommClause:
			out = append(out, posRange{s.Pos(), s.End()})
		}
		return true
	})
	return out
}

func inConditional(rs []posRange, p token.Pos) bool {
	for _, r := range rs {
		if p >= r.lo && p < r.hi {
			return true
		}
	}
	return false
}

func recvName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	switch t := fd.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func typeNameOf(f *ast.File) string {
	out := ""
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok || out != "" {
			return true
		}
		id, ok := kv.Key.(*ast.Ident)
		if !ok || id.Name != "TypeName" {
			return true
		}
		if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			out = strings.Trim(bl.Value, `"`)
		}
		return true
	})
	return out
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
