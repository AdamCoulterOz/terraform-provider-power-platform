# API client libraries

This tree contains the provider's API client libraries, separated by formal API
boundary. Each package owns the transport wire shapes (DTOs) and request logic
for exactly one Power Platform service surface, so it can evolve — and
eventually be published — as a standalone Go client module.

| Package | API boundary | Host (public cloud) |
|---|---|---|
| `api` | Shared transport: auth (azidentity), retries, correlation headers, lifecycle (async operation) polling | — |
| `bapi` | Business Application Platform API | `api.bap.microsoft.com` |
| `ppapi` | Power Platform API | `api.powerplatform.com` |
| `dataverse` | Dataverse Web API (per-environment OData v4) | `{org}.crm.dynamics.com` |
| `powerapps` | PowerApps API | `api.powerapps.com` |
| `athena` | Power Platform admin center API ("Athena") | `api.admin.powerplatform.microsoft.com` |
| `analytics` | Power Platform admin analytics (CS Analytics) API | regional `*.csanalytics.powerplatform.microsoft.com` |
| `advisor` | PowerApps Advisor API (solution checker) | per-environment runtime endpoint |
| `copilot` | Copilot Studio API | per-environment runtime endpoint |

Rules:

- Client packages MUST NOT import `terraform-plugin-framework` or any
  `internal/services` package. They may depend on `internal/clients/api`,
  `internal/constants`, `internal/config`, `internal/customerrors`, and
  `internal/helpers`.
- A client method performs calls against a single API boundary. Workflows that
  span boundaries (e.g. create environment via BAPI, then add to a billing
  policy via PPAPI) belong to the service layer in `internal/services`.
- Wire DTOs live next to the operations that produce/consume them and are
  exported. Terraform models and DTO↔model conversion stay in the services.
