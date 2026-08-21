// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment_group_rule_set

// CONNECTOR_API_PREFIX stays in the service because the Terraform schema
// documentation references it; the other rule set wire-contract constants
// moved to internal/clients/ppapi with the rule set DTOs.
const (
	CONNECTOR_API_PREFIX = "/providers/Microsoft.PowerApps/apis/"
)
