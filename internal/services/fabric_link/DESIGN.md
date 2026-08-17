# `powerplatform_fabric_link` — design & status

Manages a Dataverse **Link to Microsoft Fabric** as Terraform. The provisioning API was
reverse-engineered from a maker-portal HAR capture (2026-06-20).

## How Link to Fabric is actually provisioned

It is **not** a direct `synapselinkprofile` Web API create. The portal drives an orchestration
service (internally "athena", the old Export-to-Data-Lake service); the `synapselinkprofile` /
`datalakefolder` rows are created server-side.

This resource owns only the **provision** call. The **connection** and the **workspace** are composed
from existing resources (see Composition below) and passed in as `connection_id` + `fabric_workspace_id`;
athena binds them (it receives both ConnectionId and WorkspaceId).

**Provision** — `POST https://athenawebservice.e{cluster.uriSuffix}.powerapps.com/environment/{bapEnvId}/lakehouseArtifacts?dxt=false`
   ```jsonc
   { "OrganizationId": "<dataverse org id>", "OrganizationUrl": "https://<org>.crm<N>.dynamics.com/",
     "EnvironmentFriendlyName": "...", "EnvironmentUniqueName": "unq...",
     "Entities": ["account", ...], "EntityDescriptions": [{"Type":"account","EntitySource":"Dataverse"}, ...],
     "IsManagedLake": true, "WorkspaceId": "<fabric workspace id>", "ConnectionId": "<from step 1>" }
   ```
   Response is **base64-encoded JSON**: `{ "WorkspaceId", "LakehouseId", "ConnectionId" }` — these are
   the stable mirror ids (the lakehouse name is the unstable `dataverse_<env>_cds2_workspace_<unq>` form).

**Unlink** — `DELETE …/environment/{bapEnvId}/lakehouseArtifacts/{datalakefolderId}?dxt=false`. The id is
the Dataverse **`datalakefolderid`** (NOT the lakehouse id), resolved via
`GET {org}/api/data/v9.1/datalakefolders?$select=datalakefolderid,datalakefolder_uniquename` (the observed
unlink used the `cds3_workspace` folder; `cds2_workspace` is the fallback). Captured at create time into
`datalake_folder_id` and used by `Delete`.

### Host / field derivation (from the BAP env GET, `api-version=2020-10-01-alpha`)

- athena host = `athenawebservice.e` + `properties.cluster.uriSuffix` + `.powerapps.com`
  (e.g. `au-il301.gateway.prod.island` → `athenawebservice.eau-il301.gateway.prod.island.powerapps.com`).
- `OrganizationId` = `properties.linkedEnvironmentMetadata.resourceId`;
  `OrganizationUrl` = `…instanceUrl`; `EnvironmentUniqueName` = `…uniqueName`;
  `EnvironmentFriendlyName` = `…friendlyName`.
- path `{bapEnvId}` = the resource's `environment_id`.

## Implemented (`go build` + `go vet` clean)

- `dto.go` — BAP env, `lakehouseArtifacts` request/response (base64). (No connection DTOs — the
  connection is `powerplatform_connection`'s job.)
- `api_fabric_link.go` — `getBapEnvironment`, `athenaHost`, `CreateFabricLink` (env → POST → base64-decode)
  with the explicit athena token scope.
- `resource_fabric_link.go` — schema (`environment_id`, `name`, `fabric_workspace_id`, `connection_id`,
  `table_names`; computed `id`/`mirror_lakehouse_id`/`mirror_workspace_id`/`datalake_folder_id`/`profile_state`).
  **`Create` and `Delete` are wired**: Create provisions via athena and reads back the mirror ids +
  datalakefolder id; Delete resolves/uses the datalakefolder id to unlink. `Read` preserves state;
  `ImportState` by id.
- Registered in `internal/provider/provider.go`.

## Auth (from the fuller capture)

Two distinct token resources, both delegated (`scp=user_impersonation`) via the maker client
`appid=a8f7a65c-f5ba-4859-b2d6-df772c264e9d` in the capture:

- **athena** (`lakehouseArtifacts`, `/entities`, …): `aud = 7f15f9d9-cad0-44f1-bbba-d36650e07765`
  (the Synapse Link service app). Scope `7f15f9d9-…/.default` — set in the client.
- **connection upsert**: `aud = https://powerquery.microsoft.com`. Scope `…/.default` — set in the client.

Two *separate* notions of "service principal":
1. **Connection credentials** — what the link uses to read Dataverse on an ongoing basis. This is a
   standard CommonDataService connection created by `powerplatform_connection` with service-principal
   connection parameters — NOT this resource's concern.
2. **Caller identity** — who *calls* athena. An app-only token for `7f15f9d9-…/.default` returned `200`
   from the discovery-only `/entities` endpoint, but that did not prove the provisioning path. The
   create/registration path used by the SCH deployment requires a delegated token. `SCH.Platform.Crm`
   therefore maps this resource to its contained `powerplatform.svc_elmo` username/password alias;
   ordinary Power Platform resources continue to use the default app-only/OIDC pipeline identity.

## Composition (corrected — the connection is Fabric→Dataverse)

The "connection" is NOT Dataverse→Fabric; it is a **Fabric→Dataverse** cloud connection that lets Fabric
read Dataverse, authenticated by the **workspace identity** (`authKind: 8` in the capture). The mirror
lakehouse + SQL endpoint are created in the target Fabric workspace. Full composition:

```hcl
# 1. Workspace with a system-assigned workspace identity.
resource "fabric_workspace" "mirror" {
  display_name = "MetaForm Dataverse Mirror"
  capacity_id  = var.capacity_id
  identity      = { type = "SystemAssigned" }
}

# 2. Add the workspace identity (an SP) as a Dataverse application user and grant it the
#    "Synapse Link Service Access" role, so Fabric can read Dataverse. Uses the
#    powerplatform_application_user + powerplatform_role_assignment resources (feature/application_user
#    branch of this fork). The workspace identity exposes application_id + service_principal_id.
resource "powerplatform_application_user" "ws_identity" {
  environment_id = var.environment_id
  application_id = fabric_workspace.mirror.identity.application_id
}
resource "powerplatform_role_assignment" "synapse_link" {
  environment_id = var.environment_id
  principal_id   = powerplatform_application_user.ws_identity.system_user_id
  security_role  = "Synapse Link Service Access" # assigned by name
}

# 3. Fabric -> Dataverse connection, authenticated by the workspace identity.
#    fabric_connection already supports WorkspaceIdentity credential on main (create/update/read +
#    validator); no fork change needed.
resource "fabric_connection" "dataverse" {
  display_name      = "MetaForm Dataverse"
  connectivity_type = "ShareableCloud"
  # connection_details: CommonDataService, path = <org>.crm<N>.dynamics.com
  credential_details = { credential_type = "WorkspaceIdentity" }
}

# 4. The link (this resource) — athena orchestration. connection_id = the connection DatasourceId.
resource "powerplatform_fabric_link" "this" {
  environment_id      = var.environment_id
  name                = "MetaForm"
  fabric_workspace_id = fabric_workspace.mirror.id
  connection_id       = fabric_connection.dataverse.datasource_id # athena uses the DatasourceId
  table_names         = ["mf_formassignment", /* ... */]
}
```

Fork pieces this implies. Each stays on its OWN feature branch (so it can be PR'd to the upstream
provider independently); the consolidated build comes from merging each forward into the fork's
`codex/preview-integration` branch — feature branches are never combined with each other.
- (a) this `powerplatform_fabric_link` resource — done, `feature/fabric-link-resource` (power-platform fork).
- (b) `powerplatform_application_user` + `powerplatform_role_assignment` for the workspace-identity role
  grant — exist on `feature/application_user` (power-platform fork).
- (c) `fabric_connection` WorkspaceIdentity credential — ALREADY supported on the fabric fork's `main`
  (create/update/read + validator; `fabcore.WorkspaceIdentityCredentials`). No change needed; only a
  dedicated acceptance test is missing.
All pieces exist; each is merged forward into its fork's `codex/preview-integration` for the build.

`SCH.Platform.Crm` composes this (environment singleton) and surfaces
`mirror_workspace_id` / `mirror_lakehouse_id` as CRM facet outputs. The mirror lakehouse lands in the
target workspace, so point the link at a dedicated mirror workspace, not the Git-managed report workspace.

## Remaining

1. **Drift read** — optionally GET `synapselinkprofiles` / `datalakefolders` to detect external unlink in
   `Read` (currently state is preserved as-is).
2. **End-to-end apply** — run the full composition once (workspace+identity → app user + role →
   fabric_connection → fabric_link) to confirm the provision POST and subsequent refresh behavior under
   the contained delegated alias.

## Consumed by

`SCH.Platform.Crm` (the CRM facet, environment owner) creates this resource against a **dedicated mirror
workspace** and surfaces `mirror_workspace_id` / `mirror_lakehouse_id` as CRM facet outputs (environment
singleton) for `SCH.Platform.Data` to shortcut into the report lakehouse.
