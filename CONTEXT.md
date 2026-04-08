# Context

## Repository purpose

- Terraform provider for Microsoft Power Platform and Dataverse.
- Implemented with the Terraform Plugin Framework.
- Manages environments, solutions, publishers, users, connections, and related Power Platform resources through Power Platform admin APIs, BAPI, and Dataverse Web API.

## High-level structure

- `internal/provider`
  - provider schema, authentication wiring, and registration of resources and data sources
- `internal/services/<service>`
  - feature-specific Terraform resources, data sources, DTOs, API calls, and tests
- `internal/api`
  - shared HTTP execution, retry handling, auth, and response helpers
- `internal/helpers`
  - common utility code, request context helpers, and config glue
- `examples`, `docs`, `templates`
  - examples and generated/provider documentation inputs
- `.changes`
  - changelog material managed with Changie

## Working conventions

- New functionality usually lives under `internal/services/<name>`.
- Provider registration changes go through `internal/provider/provider.go`.
- Unit tests use `httpmock` plus `internal/mocks`.
- Docs are generated from schema descriptions and examples.
- Primary branch naming is `main`.
- Merge strategy should remain fast-forward or merge commit only. Never rebase.

## Current branch state

- `codex/preview-integration`
  - working integration branch used to build forked preview binaries for Azure DevOps pipeline consumption
  - now contains the previously separate local branch work from:
    - `main`
    - `codex/fix-environment-security-group-update`
    - `codex/git-integration-design`
    - `codex/powerplatform-publisher-resource-datasource`
    - `codex/user-refresh-missing-environment`
    - `unmanaged_solution`
    - `bug/connection_parameters_unknown`
    - `bug/solution_settings_inconsistent`
  - local merge consolidation completed on 2026-04-03
  - `git branch --no-merged codex/preview-integration` returned no remaining local branches after consolidation

## Release path used by shared Azure Pipelines

- Forked preview binaries are published from:
  - `.github/workflows/fork_provider_binaries.yml`
- Preview assets are released as GitHub prereleases with tags like:
  - `fork-v4.1.1-adam-preview.9`
- Shared Azure DevOps pipeline templates download those prerelease zip assets directly.
- The current workflow-dispatch default version in the fork binary workflow is:
  - `4.1.1-adam-preview.9`
- The fork binary workflow now forces JavaScript actions onto Node 24 and uses the same Node 24-capable pinned action SHAs as the rest of the repository.

## Recent provider work carried on preview branches

- `powerplatform_publisher` resource and data source
- Dataverse CRUD against `/api/data/v9.2/publishers`
- unmanaged solution resource/data source work
- git integration review cleanup and extra internal validation coverage
- publisher mapping fixes for placeholder address values and explicit empty-string handling
- derived default `customization_option_value_prefix`
- publisher friendly-name case-only drift handling
- publisher numeric prefix preservation after create
- unmanaged solution description handling that preserves explicit empty strings
- environment security group update fix for non-Developer environments
- `powerplatform_user` handling when the parent environment no longer exists
- connection parameter state handling when parameters remain unknown after apply
- solution settings checksum consistency across plan/apply

## Security group update bugfix

- Existing environment update logic in:
  - `internal/services/environment/resource_environment.go`
  - function `updateExistingDataverse(...)`
- The pre-fix behavior nullified `LinkedEnvironmentMetadata.SecurityGroupId` for non-Developer environments during update.
- Result:
  - Terraform planned a security-group change
  - provider update request dropped that value
  - post-apply read still returned the old group id
  - Terraform failed with `Provider produced inconsistent result after apply`
- Fix:
  - preserve the planned `dataverse.security_group_id` during non-Developer updates
- Verified locally with:
  - `go test ./internal/services/environment/...`

## User refresh missing environment bugfix

- Existing `powerplatform_user` read and delete logic in:
  - `internal/services/authorization/resource_user.go`
- Pre-fix behavior:
  - `EnvironmentHasDataverse(...)` returned `customerrors.ErrObjectNotFound` when the parent environment had been deleted
  - the resource surfaced that as a provider error during refresh
  - Terraform could not converge by removing the missing `powerplatform_user` from state
- Fix:
  - treat missing parent environments as resource disappearance
  - `Read` removes the resource from state
  - `Delete` exits cleanly when the parent environment is already gone
- Verified locally with:
  - unit coverage in `internal/services/authorization/resource_user_test.go`

## Current bugfix coordination

- The isolated bugfix branches still exist locally for clean PR slicing if needed.
- Their commits are also present on `codex/preview-integration` after the 2026-04-03 local merge consolidation.
