// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment_group_rule_set

import (
	"context"
	"fmt"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/ppapi"
)

func NewEnvironmentGroupRuleSetClient(apiClient *api.Client, tenantClient *bapi.Client) Client {
	return Client{
		Client:    ppapi.NewClient(apiClient),
		TenantApi: tenantClient,
	}
}

// Client embeds the Power Platform API client; the rule-based policy
// operations promote from it unchanged. The legacy rule sets endpoints live on
// the tenant-scoped host, so their wrappers below first resolve the tenant
// through BAPI.
type Client struct {
	*ppapi.Client
	TenantApi *bapi.Client
}

func (client *Client) GetEnvironmentGroupRuleSet(ctx context.Context, environmentGroupId string) (*ppapi.EnvironmentGroupRuleSetValueSetDto, error) {
	tenantDto, err := client.TenantApi.GetTenant(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return client.Client.GetEnvironmentGroupRuleSet(ctx, tenantDto.TenantId, environmentGroupId)
}

func (client *Client) CreateEnvironmentGroupRuleSet(ctx context.Context, environmentGroupId string, newEnvironmentGroupRuleSet ppapi.EnvironmentGroupRuleSetValueSetDto) (*ppapi.EnvironmentGroupRuleSetValueSetDto, error) {
	tenantDto, err := client.TenantApi.GetTenant(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return client.Client.CreateEnvironmentGroupRuleSet(ctx, tenantDto.TenantId, environmentGroupId, newEnvironmentGroupRuleSet)
}

func (client *Client) UpdateEnvironmentGroupRuleSet(ctx context.Context, environmentGroupId string, newEnvironmentGroupRuleSet ppapi.EnvironmentGroupRuleSetValueSetDto) (*ppapi.EnvironmentGroupRuleSetValueSetDto, error) {
	tenantDto, err := client.TenantApi.GetTenant(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return client.Client.UpdateEnvironmentGroupRuleSet(ctx, tenantDto.TenantId, environmentGroupId, newEnvironmentGroupRuleSet)
}

func (client *Client) DeleteEnvironmentGroupRuleSet(ctx context.Context, ruleSetId string) error {
	tenantDto, err := client.TenantApi.GetTenant(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}
	return client.Client.DeleteEnvironmentGroupRuleSet(ctx, tenantDto.TenantId, ruleSetId)
}
