# Interface

## Purpose

Terraform provider for declaratively managing Microsoft Power Platform and Dataverse resources.

## Responsibilities

- Expose typed Terraform resources and data sources for supported Power Platform lifecycle operations.
- Translate Terraform desired state into authenticated Power Platform, BAPI, and Dataverse API calls.
- Preserve Terraform state and fail when the target cannot satisfy the declared lifecycle.
- Do not own business-solution semantics or deployment graph composition.

## Public Interfaces

- Terraform provider `microsoft/power-platform` and its documented resources/data sources.
- `powerplatform_managed_solution` accepts an environment id, exact managed solution identity/version, local or remote package source, and connection-reference bindings.

## Invariants

- Managed solution package unique name, version, and managed layer must match configuration.
- Initial `powerplatform_managed_solution` creation performs a managed install.
- An existing resource version update uses Dataverse stage-and-upgrade so components absent from the incoming package are removed.
- Required custom solution dependencies and connection references must be satisfied before import.
- Provider errors are surfaced; lifecycle failures are not converted into successful state.

## Side Effects

- Reads and mutates resources through Power Platform administration APIs, BAPI, Microsoft Graph where declared, and Dataverse Web API.
- Managed-solution updates can delete managed components omitted from the new package.

## Dependency Boundaries

- Terraform configuration owns desired resource cardinality and ordering.
- Package producers own solution contents and versions.
- The provider owns API execution and Terraform state mapping, not cross-package dependency inference.

## Lifecycle / Execution Model

- Terraform create/read/update/delete methods map resource state to remote operations.
- Managed solution create stages and imports the package; update stages and upgrades it asynchronously, polls completion, validates the import result, and refreshes installed identity.

## Anti-Goals

- Authoring or exporting business solution content.
- Treating ordinary merge imports as authoritative managed updates.
- Hiding missing dependencies, invalid package identity, or incomplete connection bindings.

## Agent Guidance

- Preserve initial-install versus upgrade semantics when refactoring managed solutions.
- Add focused HTTP proof for action/endpoint changes and run the owning service tests.
- Update generated resource documentation when schema descriptions change.
