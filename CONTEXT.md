# Project Context

## Repository purpose

This repository contains the Microsoft Terraform provider for Power Platform. It is implemented with the Terraform Plugin Framework and manages Power Platform and Dataverse resources through a mix of Power Platform admin APIs, BAPI endpoints, and Dataverse Web API calls.

## High-level structure

- `internal/provider`: provider schema, authentication wiring, and registration of all resources and data sources.
- `internal/services/<service>`: one package per feature area, usually containing API client code, Terraform schema/model code, and tests.
- `internal/api`: shared HTTP execution, retry handling, authentication, and response helpers.
- `internal/helpers`: URL builders, request context helpers, config fallbacks, and common utility code.
- `examples`: example Terraform configurations used by docs generation.
- `docs` and `templates`: generated/provider docs and tfplugindocs templates.
- `.changes`: changelog material managed with Changie.

## Working conventions

- New functionality is typically introduced as a new package under `internal/services/<name>`.
- Provider registration must be updated in `internal/provider/provider.go`.
- Unit tests use `httpmock` together with `internal/mocks`.
- Documentation is generated from schema descriptions and examples via `go generate` / `make userdocs`.
- The repo currently uses `main` as the primary branch, which matches local project instructions.

## Current branch state

Branch: `codex/git-integration-binary-pipeline`

Current work on this branch combines the forked provider binary pipeline with several in-flight product features:

- temporary GitHub Actions workflow for building forked provider binaries for Azure DevOps pipeline consumption
- git integration resources under `internal/services/git_integration`
- unmanaged solution resource and data source under `internal/services/solution`
- typed Dataverse publisher resource and data source under `internal/services/publisher`

The fork-binary workflow currently stamps preview builds with an `adam-preview` version suffix so generated archives and prerelease tags are distinct from the earlier `gitintegration` builds.

Recent publisher work on this branch adds:

- `powerplatform_publisher` resource
- `powerplatform_publisher` data source
- Dataverse CRUD against `/api/data/v9.2/publishers`
- Provider registration, examples, tests, and docs generation inputs
- The publisher mapper now ignores Dataverse placeholder address slots that only contain internal/default values unless that slot was already tracked in Terraform state, which avoids `Provider produced inconsistent result after apply` when configuration omits `address`.
- The publisher mapper also preserves explicit empty-string optional fields and explicit empty `address` configuration, avoiding additional empty-vs-null drift after create/update.
- `customization_option_value_prefix` is now intended to be optional on the resource, with the provider deriving the default value using the same hash algorithm used by the Power Apps publisher UI when the field is omitted.
- The unmanaged solution resource now preserves an explicitly configured empty-string `description` instead of normalizing it to `null`, which avoids `Provider produced inconsistent result after apply` during create/update.

## Publisher design notes

- Resource id format is planned as `<environment_id>_<publisher_id>` so imports can carry both the Dataverse environment and raw publisher GUID.
- The schema uses explicit top-level publisher fields for core publisher metadata.
- Addresses are modeled as a repeated child `address` structure with up to two entries, mapped to Dataverse `address1_*` and `address2_*` fields.
- The data source supports lookup by either `publisher_id` or `uniquename`.

## Open assumptions

- The current publisher schema covers the core publisher metadata plus the full mutable address surface. It does not yet expose every non-address property on the Dataverse publisher entity.
- `uniquename` is treated as replacement-only because it behaves like a stable identity field.
- Optional fields are cleared on update by sending `null` values to Dataverse for omitted attributes.
