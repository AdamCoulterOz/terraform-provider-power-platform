# `fabric_link` — pick-up notes

Branch-local note for `feature/fabric-link-ropc`. Delete it when the branch merges;
it exists because the athena spec moved after this code was written and the code has
not caught up yet.

## Where things stand

Three defects were fixed in `eb90b78d` and are done: the athena host prefix is derived
from `properties.azureRegion` instead of a hardcoded `e`, `getDatalakeFolderId` no
longer falls back to `value[0]` of the Dataverse `datalakefolders` list, and the BAP
environment read uses `constants.BAP_API_VERSION` rather than `2020-10-01-alpha`.
Unit tests pin the host construction and the folder selection.

Nothing on the athena surface has ever been exercised live from this provider. Every
operation on it either provisions or destroys, so there is no safe read to test with.

## What the 18-capture spec supersedes

[`AdamCoulterOz/powerplatform-apis/athena`](https://github.com/AdamCoulterOz/powerplatform-apis/tree/main/athena)
was rewritten from 18 HAR captures (291 recorded calls) *after* this code was written.
It now covers 15 operations against the 3 this branch drives, and it contradicts four
things here. The spec is the authority in each case — it is recorded traffic, whereas
the comments in this package are inferences from a single earlier capture. Do not
"fix" the spec to match this code.

1. **Resolve the folder id from `GET .../lakedetails`, not from Dataverse.** It returns
   the id as `Id`, and the recorded unlinks pass exactly that value to
   `DELETE .../lakehouseArtifacts/{datalakeFolderId}`. It is an empty array when there
   is no link, and empties ~30s after a successful unlink. This is strictly better than
   the `synapselinkprofiles` cross-check in `getDatalakeFolderId`: it is the service's
   own answer, it distinguishes a linked environment from an unlinked one, and it can
   tell a Fabric link from a Synapse one — which the Dataverse read cannot do except by
   the `cds3_workspace` name. It also removes the need to persist `datalake_folder_id`
   in state at all, since `Delete` could resolve it live. Keep the cross-check only as
   a fallback, if at all.

2. **The create response is plain JSON.** `decodeArtifactsResponse` tries base64 first.
   That was never real: the base64 came from the HAR file's own
   `response.content.encoding` field — the capture format, not the API. Recorded
   responses carry `Content-Type: application/json` and a plain object. The plain-JSON
   fallback means this is a wrong comment and a dead path rather than a live bug.

3. **The bodiless `403` is on the licence check, not on create.** It lands on
   `GET .../lakehouseArtifacts/hasPowerBIPremiumLicense`, and both portals recover by
   calling `updateorganizationdetails` within 200ms and repeating the licence check.
   The create-403 self-heal in `CreateFabricLink` was inferred and the traffic
   contradicts it. A client that treats that 403 as fatal reports a licensing problem
   to a user who has none.

4. **`updateorganizationdetails` takes no body.** Query params only, every recorded
   request `Content-Length: 0`. The `{"headers": {...}}` body this package sends was a
   portal bug — it had serialised its own HTTP header bag — copied verbatim from the
   capture of that bug. The service ignored it.

## Twelve operations this branch does not know about

The gap is the whole read side plus in-place amendment, and it is why `Read` preserves
state and `Update` is a scaffold error:

- `GET .../lakeprofile/{id}` — per-table sync state, the only place link progress is
  observable. `CurrentState` walks `InitialSyncNotStarted` → `InitialSyncInProgress` →
  `InitialSyncPostProcessing` → `InProgress`, where `InProgress` means *done, on delta
  sync*. This is what a real `Read` and a data source would be built on.
- `PUT .../lakeprofile/{id}` + `POST .../lakeprofile/{id}/activate` — add and remove
  tables on an existing link. Without these, every `table_names` change is a
  destroy-and-recreate. Re-posting lakehouse artifacts does not change the table list.
- `GET /entities` — the table catalogue, *organization*-scoped (takes `organizationUrl`,
  no path id). Flags each table `IsDisabled` with a `ReasonIfDisabled`; across 13,690
  recorded rows the only reason was `Change Tracking is Disabled`, which is the most
  common thing a caller hits and is fixable only in Dataverse.
- `GET /relationships` — many-to-many only. Linking both ends of an N:N does not bring
  the join table across; this is how you find the intersect table's name.
- `GET .../fabric/workspaces` — athena proxies Fabric, so no Fabric token is needed to
  enumerate workspaces, and it supplies the `capacityId` to filter on (a workspace with
  no capacity cannot host a mirror lakehouse).

## Two things still unverified in what shipped

- **The letter for `central` is extrapolated.** DNS confirmed `e`, `n`, `w`, `se`, and
  the captures independently confirm `e` and `se` on the same island. `c`/`nc`/`sc`/`wc`
  follow the one-letter-per-compass-word rule but have not been observed. A wrong prefix
  is `NXDOMAIN`, not a diagnosable error.
- **The `synapselinkprofile` → `datalakefolder` lookup column name was never confirmed.**
  `getLinkedDatalakeFolderIds` deliberately omits `$select` and matches
  `_datalakefolderid_value` / `datalakefolderid` off the row, so a wrong guess degrades
  instead of 400-ing the query. If the real column is named something else, folder
  resolution silently returns nothing: create logs a warning and destroy cannot unlink.
  Switching to `lakedetails` (1 above) makes this moot.

## Also

The admin centre client (`api.admin.powerplatform.microsoft.com`, `PPAC_SCOPE`) also
carried the internal code name Athena, for a different service. It was renamed
`internal/clients/athena` → `internal/clients/admin` in `20e855ee`, so on
`refactor/clients-integration` the name is free. **The rename did not propagate**: as
of writing, twenty per-boundary branches still carry `internal/clients/athena`,
including `refactor/clients-boundaries-base`, which owns the file (`ad3fb32f`). Until
the rename lands there and is merged forward, every branch PR'd upstream on its own
reintroduces it. Either way, prefer `synapselink` for this boundary on clarity
grounds: the service is Synapse Link, and Link to Fabric is one product surface on
top of it. Three different things in this ecosystem have carried the Athena name.
