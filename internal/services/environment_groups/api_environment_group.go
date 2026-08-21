// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment_groups

import (
	"context"
	"fmt"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/ppapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/services/environment_group_rule_set"
)

func newEnvironmentGroupClient(apiClient *api.Client, tenantClient *bapi.Client, ruleSetClient environment_group_rule_set.Client) client {
	return client{
		Client:           tenantClient,
		PowerPlatformApi: ppapi.NewClient(apiClient),
		RuleSetApi:       ruleSetClient,
	}
}

// client embeds the BAPI client; the environment group CRUD operations promote
// from it unchanged. Workflows that span another boundary (Power Platform API,
// rule sets) go through the named clients below.
type client struct {
	*bapi.Client
	PowerPlatformApi *ppapi.Client
	RuleSetApi       environment_group_rule_set.Client
}

// RemoveEnvironmentFromEnvironmentGroup resolves the tenant (BAPI) and removes
// the environment from the group (Power Platform API).
func (client *client) RemoveEnvironmentFromEnvironmentGroup(ctx context.Context, environmentGroupId, environmentId string) error {
	tenantDto, err := client.GetTenant(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}
	return client.PowerPlatformApi.RemoveEnvironmentFromEnvironmentGroup(ctx, tenantDto.TenantId, environmentGroupId, environmentId)
}
