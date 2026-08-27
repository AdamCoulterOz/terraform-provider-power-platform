// A module of its own, so the provider's dependency graph does not grow a
// static-analysis library it never links.
module github.com/microsoft/terraform-provider-power-platform/tools/apiusage

go 1.25.0

require golang.org/x/tools v0.48.0

require (
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
)
