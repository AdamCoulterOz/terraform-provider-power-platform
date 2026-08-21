// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package role_based_access

import (
	"context"
	"fmt"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/ppapi"
)

func newRoleBasedAccessClient(apiClient *api.Client) client {
	return client{
		Client:    ppapi.NewClient(apiClient),
		TenantApi: bapi.NewClient(apiClient),
	}
}

// client embeds the Power Platform API boundary client that owns the RBAC
// operations and holds the BAPI client needed to resolve the tenant scope.
type client struct {
	*ppapi.Client
	TenantApi *bapi.Client
}

// tenantScope returns the fully qualified tenant scope string required by the RBAC API.
func (client *client) tenantScope(ctx context.Context) (string, error) {
	tenantDto, err := client.TenantApi.GetTenant(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/tenants/%s", tenantDto.TenantId), nil
}

// CreateRoleAssignment creates a role assignment at tenant level. Resolving the
// tenant scope spans the BAPI boundary, so it happens here before delegating to
// the Power Platform API client.
func (client *client) CreateRoleAssignment(ctx context.Context, request ppapi.RoleAssignmentRequestDto) (*ppapi.RoleAssignmentDto, error) {
	scope, err := client.tenantScope(ctx)
	if err != nil {
		return nil, err
	}
	request.Scope = scope
	return client.Client.CreateRoleAssignment(ctx, request)
}

// CreateEnvironmentGroupRoleAssignment creates a role assignment for an environment group.
func (client *client) CreateEnvironmentGroupRoleAssignment(ctx context.Context, environmentGroupId string, request ppapi.RoleAssignmentRequestDto) (*ppapi.RoleAssignmentDto, error) {
	scope, err := client.tenantScope(ctx)
	if err != nil {
		return nil, err
	}
	request.Scope = fmt.Sprintf("%s/environmentGroups/%s", scope, environmentGroupId)
	return client.Client.CreateEnvironmentGroupRoleAssignment(ctx, environmentGroupId, request)
}

// CreateEnvironmentRoleAssignment creates a role assignment for an environment.
func (client *client) CreateEnvironmentRoleAssignment(ctx context.Context, environmentId string, request ppapi.RoleAssignmentRequestDto) (*ppapi.RoleAssignmentDto, error) {
	scope, err := client.tenantScope(ctx)
	if err != nil {
		return nil, err
	}
	request.Scope = fmt.Sprintf("%s/environments/%s", scope, environmentId)
	return client.Client.CreateEnvironmentRoleAssignment(ctx, environmentId, request)
}
