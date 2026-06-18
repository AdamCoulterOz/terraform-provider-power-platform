// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package managed_environment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/microsoft/terraform-provider-power-platform/internal/services/environment"
)

func TestPopulateStateFromEnvironment_WhenSettingsMissing_DoesNotLeaveUnknowns(t *testing.T) {
	plan := &ManagedEnvironmentResourceModel{
		Id:                                     types.StringUnknown(),
		EnvironmentId:                          types.StringValue("00000000-0000-0000-0000-000000000001"),
		ProtectionLevel:                        types.StringUnknown(),
		IsUsageInsightsDisabled:                types.BoolValue(true),
		IsGroupSharingDisabled:                 types.BoolValue(false),
		MaxLimitUserSharing:                    types.Int64Value(-1),
		LimitSharingMode:                       types.StringValue("ExcludeSharingToSecurityGroups"),
		SolutionCheckerMode:                    types.StringValue("Warn"),
		SuppressValidationEmails:               types.BoolValue(false),
		SolutionCheckerRuleOverrides:           types.SetUnknown(types.StringType),
		PowerAutomateIsSharingDisabled:         types.BoolUnknown(),
		CopilotAllowGrantPermissionsWhenShared: types.BoolUnknown(),
		CopilotLimitSharingMode:                types.StringUnknown(),
		CopilotMaxLimitUserSharing:             types.Int64Unknown(),
	}

	env := &environment.EnvironmentDto{
		Properties: &environment.EnviromentPropertiesDto{
			GovernanceConfiguration: &environment.GovernanceConfigurationDto{
				ProtectionLevel: "Standard",
				Settings:        nil,
			},
		},
	}

	var diagnostics diag.Diagnostics
	resource := &ManagedEnvironmentResource{}
	resource.populateStateFromEnvironment(context.Background(), plan, env, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("populateStateFromEnvironment returned unexpected diagnostics: %v", diagnostics)
	}

	assertStringKnown(t, "id", plan.Id)
	assertStringKnown(t, "environment_id", plan.EnvironmentId)
	assertStringKnown(t, "protection_level", plan.ProtectionLevel)
	assertBoolKnown(t, "is_usage_insights_disabled", plan.IsUsageInsightsDisabled)
	assertBoolKnown(t, "is_group_sharing_disabled", plan.IsGroupSharingDisabled)
	assertInt64Known(t, "max_limit_user_sharing", plan.MaxLimitUserSharing)
	assertStringKnown(t, "limit_sharing_mode", plan.LimitSharingMode)
	assertStringKnown(t, "solution_checker_mode", plan.SolutionCheckerMode)
	assertBoolKnown(t, "suppress_validation_emails", plan.SuppressValidationEmails)

	if plan.SolutionCheckerRuleOverrides.IsUnknown() {
		t.Fatal("solution_checker_rule_overrides remained unknown")
	}
	assertBoolKnown(t, "power_automate_is_sharing_disabled", plan.PowerAutomateIsSharingDisabled)
	assertBoolKnown(t, "copilot_allow_grant_editor_permissions_when_shared", plan.CopilotAllowGrantPermissionsWhenShared)
	assertStringKnown(t, "copilot_limit_sharing_mode", plan.CopilotLimitSharingMode)
	assertInt64Known(t, "copilot_max_limit_user_sharing", plan.CopilotMaxLimitUserSharing)

	if !plan.PowerAutomateIsSharingDisabled.IsNull() ||
		!plan.CopilotAllowGrantPermissionsWhenShared.IsNull() ||
		!plan.CopilotLimitSharingMode.IsNull() ||
		!plan.CopilotMaxLimitUserSharing.IsNull() {
		t.Fatal("omitted optional computed settings should normalize to null when the API does not return settings")
	}
}

func assertStringKnown(t *testing.T, name string, value types.String) {
	t.Helper()
	if value.IsUnknown() {
		t.Fatalf("%s remained unknown", name)
	}
}

func assertBoolKnown(t *testing.T, name string, value types.Bool) {
	t.Helper()
	if value.IsUnknown() {
		t.Fatalf("%s remained unknown", name)
	}
}

func assertInt64Known(t *testing.T, name string, value types.Int64) {
	t.Helper()
	if value.IsUnknown() {
		t.Fatalf("%s remained unknown", name)
	}
}
