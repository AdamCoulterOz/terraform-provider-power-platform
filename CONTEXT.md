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
  - carries in-flight work around git integration resources, unmanaged solutions, and publishers
- `codex/fix-environment-security-group-update`
  - clean bugfix branch cut from `upstream/main`
  - fixes environment update behavior so non-Developer environment updates preserve the planned `dataverse.security_group_id`
- `codex/user-refresh-missing-environment`
  - clean bugfix branch cut from `upstream/main`
  - fixes `powerplatform_user` refresh and delete behavior when the parent environment no longer exists
  - treats `EnvironmentNotFound` as resource disappearance instead of surfacing a provider error

## Release path used by shared Azure Pipelines

- Forked preview binaries are published from:
  - `.github/workflows/fork_provider_binaries.yml`
- Preview assets are released as GitHub prereleases with tags like:
  - `fork-v4.1.1-adam-preview.6`
- Shared Azure DevOps pipeline templates download those prerelease zip assets directly.

## Recent provider work carried on preview branches

- `powerplatform_publisher` resource and data source
- Dataverse CRUD against `/api/data/v9.2/publishers`
- unmanaged solution resource/data source work
- publisher mapping fixes for placeholder address values and explicit empty-string handling
- derived default `customization_option_value_prefix`
- unmanaged solution description handling that preserves explicit empty strings

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

- Two separate bug reports are needed before opening PRs:
  - environment security group update
  - user refresh when parent environment is missing
- Each bugfix should stay isolated to its own branch, issue, and PR.
