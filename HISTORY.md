# History

## 2026-07-13 - Managed solution updates became authoritative upgrades

Changed `powerplatform_managed_solution` update behavior from ordinary managed import to Dataverse `StageAndUpgradeAsync`. Initial resource creation still performs a normal managed install. Version updates now remove components omitted from the incoming managed package, matching Terraform desired-state semantics rather than merge semantics.
