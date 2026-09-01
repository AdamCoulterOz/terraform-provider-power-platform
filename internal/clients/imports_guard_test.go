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

// TestUnitClientPackagesDoNotImportProviderPlane enforces the dependency rule
// from internal/clients/README.md: client libraries must stay independent of
// the Terraform plane so they can be extracted as standalone modules.
func TestUnitClientPackagesDoNotImportProviderPlane(t *testing.T) {
	forbidden := []string{
		"github.com/hashicorp/terraform-plugin-framework",
		"github.com/hashicorp/terraform-plugin-testing",
		"github.com/microsoft/terraform-provider-power-platform/internal/services",
		"github.com/microsoft/terraform-provider-power-platform/internal/provider",
		"github.com/microsoft/terraform-provider-power-platform/internal/modifiers",
		"github.com/microsoft/terraform-provider-power-platform/internal/validators",
	}

	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, banned := range forbidden {
				if importPath == banned || strings.HasPrefix(importPath, banned+"/") {
					t.Errorf("%s imports %q, which is not allowed in client libraries", path, importPath)
				}
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
// removed. That is not hypothetical -- renaming the admin centre client from
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
