# History

## 2026-07-13 - Managed solution version state became representation-stable

Preserved the declared Terraform version when Dataverse reports the same solution version using its canonical four-part representation. This prevents post-apply inconsistency for inputs such as `0.1.39` while continuing to expose genuine remote version drift.

## 2026-07-13 - Managed solution updates became authoritative upgrades

Changed `powerplatform_managed_solution` update behavior from ordinary managed import to Dataverse `StageAndUpgradeAsync`. Initial resource creation still performs a normal managed install. Version updates now remove components omitted from the incoming managed package, matching Terraform desired-state semantics rather than merge semantics.
