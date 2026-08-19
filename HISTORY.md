# History

## 2026-07-12 - Equivalent managed-solution version renderings stabilized

Dataverse renders omitted trailing solution-version segments as zeroes, so a package and declaration at `0.1.39` is reported remotely as `0.1.39.0`. Returning that remote rendering after import violated Terraform's planned-value contract and caused a successful import to end with an inconsistent-result error. Managed-solution lifecycle handling now compares the normalized four-part identities, preserves the declared representation in state when the remote identity is equivalent, and still reports real installed-version drift.

## 2026-07-12 - Fabric unlink made organization-scoped, bounded, and fail-loud

An acdev MetaForm retirement exposed that `powerplatform_fabric_link` Delete omitted athena's mandatory `x-ms-organization-id` header. The generic HTTP retry loop then retried the unresolved request for more than ten minutes while the live Dataverse profile remained Active.

Delete now resolves the BAP environment metadata, supplies the Dataverse organization id on the athena request, limits retryable unlink failures to a two-minute child context, preserves the final real HTTP status/body when that deadline ends the loop, and no longer treats an ambiguous `404` as proof that the live link is absent. Terraform keeps the resource in state unless unlink returns a confirmed `200` or `204`.

## 2026-07-02 - Fabric-link preview added

Added `powerplatform_fabric_link` around the maker-portal athena orchestration flow. Create records the mirror identifiers and Dataverse datalake-folder identifier required for unlink; workspace and Fabric connection ownership remain outside the resource.

## 2026-04-03 - Preview provider work consolidated

Consolidated the fork's application-user, environment-variable, managed-solution, publisher, Git-integration, connection, environment-settings, and other preview branches into `codex/preview-integration` so the SCH provider binary is built from one coherent feature set.
