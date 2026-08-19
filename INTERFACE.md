# Interface Contract

## 1. Purpose

This repository builds the Terraform provider for Microsoft Power Platform. The `codex/preview-integration` fork combines the preview resources and fixes consumed by SCH Platform, including the Dataverse Link to Microsoft Fabric lifecycle adapter.

## 2. Responsibilities

The provider translates Terraform resource/data-source lifecycles into Power Platform, BAP, Dataverse, and related Microsoft service operations. It owns provider authentication, schema/state mapping, remote CRUD mechanics, retry behavior, and fail-loud diagnostics.

It may expand with provider-level Power Platform capabilities. It must not own solution-graph planning, business solution semantics, consumer pipeline orchestration, Fabric workspace ownership, or connection credential ownership.

## 3. Domain Model

Primary entities are Power Platform environments and their provider-managed children, including solutions, publishers, users, roles, connections, settings, variables, and preview Fabric links. A Fabric link associates one Dataverse environment with a target Fabric workspace and Fabric-to-Dataverse connection; athena creates the mirror artifacts and Dataverse folder/profile records.

## 4. Public Interfaces

The public artifact is the Terraform provider binary registered under the Microsoft Power Platform provider address. Its resource and data-source schemas are the external contract.

`powerplatform_managed_solution.version` accepts one to four numeric segments. Omitted trailing segments are semantically zero, while Terraform state retains the consumer's declared rendering whenever Dataverse reports the same normalized identity.

`powerplatform_fabric_link` accepts `environment_id`, `name`, optional `unique_name`, `fabric_workspace_id`, `connection_id`, `table_names`, and standard timeouts. It exposes `id`, `mirror_lakehouse_id`, `mirror_workspace_id`, `profile_state`, and `datalake_folder_id`.

## 5. Invariants

- Environment-scoped athena calls include `x-ms-organization-id`.
- Managed-solution imports must report an installed version semantically equal to the declared version; equivalent three-part/four-part rendering differences must not cause state churn.
- Fabric unlink targets `datalake_folder_id`; mirror ids are not interchangeable with it.
- Unlink succeeds only on confirmed `200` or `204` responses.
- Retryable unlink failures stop within the two-minute provider ceiling or an earlier caller deadline.
- Failed or ambiguous unlink keeps the resource in Terraform state and reports the last real HTTP failure.
- Provider schemas and registered resource/data-source lists remain consistent with the built binary.

## 6. Side Effects

Provider operations obtain Azure tokens, call Microsoft service APIs, create/update/delete remote resources, and persist remote identifiers in Terraform state. Unit tests replace HTTP transport and must not call live services.

## 7. Dependency Boundaries

Upstream dependencies include Terraform Plugin Framework, Azure identity libraries, and Microsoft service APIs. Downstream consumers include SCH Platform modules and ordinary Terraform configurations. Undocumented service behavior is implementation evidence, not a stable semantic contract.

## 8. Lifecycle / Execution Model

Terraform configures a shared provider API client and invokes resource lifecycle methods synchronously. Retryable HTTP responses are retried within the owning operation context. Fabric-link Create resolves BAP organization metadata, provisions through athena, and records returned ids; Read currently preserves state; Delete resolves the same metadata and unlinks through an organization-scoped, bounded athena call.

## 9. Anti-Goals

- Directly managing server-owned Fabric-link Dataverse rows as independent semantic resources.
- Treating display names as stable remote identity.
- Silently accepting ambiguous remote responses or masking failures.
- Coupling consumer solution graphs or Fabric workspace lifecycles into the provider.

## 10. Agent Guidance

Preserve Terraform state when deletion is uncertain. Add HTTP-level regression coverage for provider request-contract changes, including headers, identifiers, accepted statuses, cancellation, and diagnostics. Run focused tests, the full Go suite, and a provider build before handoff. Update this interface when public schemas, lifecycle behavior, invariants, side effects, or dependency boundaries change.
