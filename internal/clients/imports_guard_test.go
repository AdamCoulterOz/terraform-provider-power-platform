// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package clients_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/microsoft/terraform-provider-power-platform"

// allowedModuleImports is the allowlist from internal/clients/README.md: the
// only module-internal packages a client library may depend on. Everything
// else inside the module - services, provider, modifiers, validators, other
// boundary client packages - is forbidden.
var allowedModuleImports = map[string]bool{
	modulePath + "/common":                true,
	modulePath + "/internal/clients/api":  true,
	modulePath + "/internal/config":       true,
	modulePath + "/internal/constants":    true,
	modulePath + "/internal/customerrors": true,
	modulePath + "/internal/helpers":      true,
}

// forbiddenExternalPrefixes are Terraform-plane dependencies that must never
// leak into client libraries.
var forbiddenExternalPrefixes = []string{
	"github.com/hashicorp/terraform-plugin-framework",
	"github.com/hashicorp/terraform-plugin-testing",
	"github.com/hashicorp/terraform-plugin-mux",
	"github.com/hashicorp/terraform-plugin-go",
}

// TestUnitClientPackagesDoNotImportProviderPlane enforces the dependency rules
// from internal/clients/README.md so the client libraries stay independently
// extractable: no Terraform-plane imports, no provider/service imports, and no
// cross-boundary client imports (boundaries may only share internal/clients/api).
func TestUnitClientPackagesDoNotImportProviderPlane(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		ownPackageDir := filepath.Dir(path) // e.g. "bapi", "api", "."

		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			for _, banned := range forbiddenExternalPrefixes {
				if importPath == banned || strings.HasPrefix(importPath, banned+"/") {
					t.Errorf("%s imports %q, which is not allowed in client libraries", path, importPath)
				}
			}

			if !strings.HasPrefix(importPath, modulePath+"/") {
				continue // stdlib and third-party (tflog, azure sdk, uuid) are fine
			}
			if importPath == modulePath+"/internal/clients/"+ownPackageDir {
				continue // own package path never appears, but be safe
			}
			if importPath == modulePath+"/internal/clients/api" && ownPackageDir == "api" {
				t.Errorf("%s: the shared api package must not import itself", path)
				continue
			}
			if !allowedModuleImports[importPath] {
				t.Errorf("%s imports %q, which is outside the client-library allowlist (cross-boundary client imports are forbidden; boundaries share only internal/clients/api)", path, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking internal/clients: %v", err)
	}
}

// TestUnitClientPackageImportsResolve checks that every internal/clients/<pkg>
// import anywhere in the module names a package that exists.
//
// The allowlist test above walks only this directory and polices *direction*:
// what a client package may import. It is blind to the opposite mistake, where
// something outside this tree imports a client package that has been renamed or
// removed. That is not hypothetical -- renaming the admin center client from
// athena to admin left internal/services/environment_wave importing a package
// that no longer existed, on six branches, and nothing here failed. The rename
// was complete inside the directory it touched and stopped at its boundary.
func TestUnitClientPackageImportsResolve(t *testing.T) {
	root := filepath.Join("..", "..")
	const prefix = "github.com/microsoft/terraform-provider-power-platform/internal/clients/"

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		parsed, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if perr != nil {
			return nil // unparseable files are not this test's business
		}
		for _, spec := range parsed.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if !strings.HasPrefix(importPath, prefix) {
				continue
			}
			pkg := strings.TrimPrefix(importPath, prefix)
			dir := filepath.Join(root, "internal", "clients", filepath.FromSlash(pkg))
			if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
				t.Errorf("%s imports %q, but internal/clients/%s does not exist; a client package was renamed or removed without updating its consumers", path, importPath, pkg)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}
}
