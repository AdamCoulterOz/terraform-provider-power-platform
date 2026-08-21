// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment_wave

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResourceModel struct {
	Id            types.String   `tfsdk:"id"`
	EnvironmentId types.String   `tfsdk:"environment_id"`
	FeatureName   types.String   `tfsdk:"feature_name"`
	State         types.String   `tfsdk:"state"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}
