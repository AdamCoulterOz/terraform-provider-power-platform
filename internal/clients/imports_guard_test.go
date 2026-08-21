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
