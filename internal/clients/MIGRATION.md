# Client extraction recipe

This is the harness contract every boundary package follows and the recipe for
migrating a service's API code into it. Do not deviate from the contract —
consistency across packages is the point.

## Harness contract (per boundary package)

Every boundary package (`bapi`, `ppapi`, `dataverse`, `powerapps`, `athena`,
`advisor`, `copilot`) has exactly this shape:

- `doc.go` — package doc: which API, host, auth scope, spec status.
- `client.go` — exactly:
  - `type Client struct { Api *api.Client }` (field exported).
  - `func NewClient(apiClient *api.Client) *Client`.
  - `func (client *Client) scopes() []string` returning the boundary's OAuth
    scope from config. For per-environment-host boundaries (`dataverse`,
    `advisor`, `copilot`) it is `scopes(environmentHost string) []string`
    returning `{"https://" + environmentHost + "/.default"}` (or the boundary's
    fixed scope constant where one exists).
  - `func (client *Client) host() string` returning the boundary host from
    config. Omitted for per-environment-host boundaries; those take the host as
    a method parameter.
- `<domain>.go` — operations for one resource domain (e.g. `environments.go`,
  `tenant.go`). Receiver is always `client *Client`.
- `<domain>_dtos.go` — the wire DTOs for that domain, all exported.

Rules for operations:

- Move method bodies **verbatim** from the service `api_*.go`: identical URL
  path, api-version, query values, headers, body, and acceptable status codes.
  The only permitted changes: receiver type, `client.Api.GetConfig().Urls.XxxUrl`
  → `client.host()`, and `client.Api.Execute(ctx, nil, ...)` →
  `client.Api.Execute(ctx, client.scopes(), ...)`.
- After migration, `Execute(ctx, nil` must not appear anywhere under
  `internal/clients/<boundary>/` — grep for it.
- Single-boundary workflows (retry-on-409 loops, lifecycle waits, read-after-
  write polls) move with the client. Anything that calls a second boundary or a
  service package stays in the service.
- Client packages never import `terraform-plugin-framework` or
  `internal/services/...` (enforced by `imports_guard_test.go`).
- Keep `customerrors` wrapping exactly as-is; error text is part of observed
  behavior.

## DTO rules

- Exported names, `Dto` suffix preserved. When a service-local lowercase type
  moves, capitalize it (`environmentIdDto` → `EnvironmentIdDto`).
- Before adding a DTO, grep the boundary package for an existing type of the
  same wire shape. Reuse the canonical type when every referenced field exists
  on it with identical json tags **and** the type is only unmarshaled
  (responses). Request DTOs keep their exact original shape — never widen a
  type that gets marshaled into a request body.
- If a same-named but differently-shaped type must move, prefix it with its
  domain (`RoleAssignmentEnvironmentDto`), never overwrite the canonical one.

## Service rewiring rules

- The service keeps its own client struct but **embeds** the boundary client
  so existing call sites promote unchanged:

  ```go
  type client struct {
      *bapi.Client
      // cross-boundary orchestration deps stay here
  }
  func newWidgetClient(apiClient *api.Client) client {
      return client{Client: bapi.NewClient(apiClient)}
  }
  ```

  `client.Api` keeps working through promotion. If a service client collapses
  to nothing but the embed, replace it with the boundary client directly.
- References to moved DTOs get qualified (`widgetDto` → `bapi.WidgetDto`) in
  all service files including `_test.go`.
- Services that duplicated an operation (environment lookup, env-URL
  resolution, tenant lookup) switch to the canonical boundary method and their
  local copy is deleted.
- Do not change any Terraform schema, model, or conversion logic.

## Definition of done (per service)

1. `go build ./...` clean.
2. `go vet ./internal/...` clean.
3. `go test ./internal/services/<svc>/... -run '^TestUnit' -count=1` passes.
4. `go test ./internal/clients/... -run '^TestUnit' -count=1` passes.
5. No `Execute(ctx, nil` under the boundary package; no forbidden imports.
6. `git diff` for the service shows only: deleted/shrunk `api_*.go` and
   `dto.go`, qualified type references, and client construction changes.
