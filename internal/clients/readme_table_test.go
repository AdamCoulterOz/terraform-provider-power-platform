// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package clients_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// tableRow matches a boundary-table row in README.md, whose first cell is the
// package name in backticks: | `bapi` | Business Application Platform API | ...
var tableRow = regexp.MustCompile("^\\|\\s*`([a-z][a-z0-9]*)`\\s*\\|")

// TestUnitEveryClientPackageIsDocumented asserts that every package directory
// in internal/clients has a row in the README boundary table.
//
// A client package is a published API surface, so an undocumented one is a
// reviewer having to read the code to learn which service it talks to. This
// has already happened: internal/clients/analytics shipped while the table
// listed every package except that one.
//
// Only this direction is asserted. The reverse -- every row naming a directory
// that exists -- is deliberately not checked, because the table also documents
// boundaries that are reserved rather than implemented (synapselink), and that
// is the table doing its job.
func TestUnitEveryClientPackageIsDocumented(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}

	documented := map[string]bool{}
	for _, line := range strings.Split(string(readme), "\n") {
		if m := tableRow.FindStringSubmatch(line); m != nil {
			documented[m[1]] = true
		}
	}
	if len(documented) == 0 {
		t.Fatal("no boundary-table rows found in README.md; the table format changed and this test no longer checks anything")
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading internal/clients: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !documented[e.Name()] {
			t.Errorf("internal/clients/%s has no row in README.md's boundary table; every client package documents the API boundary it owns and the host it calls", e.Name())
		}
	}
}
