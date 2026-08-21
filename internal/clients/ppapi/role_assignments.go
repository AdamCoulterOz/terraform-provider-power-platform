// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

const roleAssignmentsApiVersion = "2024-10-01"

// The RBAC API rejects a freshly created environment or environment group until its id propagates.
var scopePropagationErrorCodes = []string{"EnvironmentIdInvalid", "EnvironmentGroupIdInvalid"}

// CreateRoleAssignment creates a role assignment at tenant level. The request's
// Scope must already be fully qualified (e.g. "/tenants/{tenantId}"); resolving
// it requires the BAPI tenant lookup and stays with the caller.
func (client *Client) CreateRoleAssignment(ctx context.Context, request RoleAssignmentRequestDto) (*RoleAssignmentDto, error) {
	return client.createRoleAssignmentAtPath(ctx, "/authorization/roleAssignments", request)
}

// CreateEnvironmentGroupRoleAssignment creates a role assignment for an
// environment group. The request's Scope must already be fully qualified.
func (client *Client) CreateEnvironmentGroupRoleAssignment(ctx context.Context, environmentGroupId string, request RoleAssignmentRequestDto) (*RoleAssignmentDto, error) {
	path := fmt.Sprintf("/authorization/environmentGroups/%s/roleAssignments", environmentGroupId)
	return client.createRoleAssignmentAtPath(ctx, path, request)
}

// CreateEnvironmentRoleAssignment creates a role assignment for an environment.
// The request's Scope must already be fully qualified.
func (client *Client) CreateEnvironmentRoleAssignment(ctx context.Context, environmentId string, request RoleAssignmentRequestDto) (*RoleAssignmentDto, error) {
	path := fmt.Sprintf("/authorization/environments/%s/roleAssignments", environmentId)
	return client.createRoleAssignmentAtPath(ctx, path, request)
}

func (client *Client) createRoleAssignmentAtPath(ctx context.Context, path string, request RoleAssignmentRequestDto) (*RoleAssignmentDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   path,
	}
	values := url.Values{}
	values.Add("api-version", roleAssignmentsApiVersion)
	apiUrl.RawQuery = values.Encode()

	response := RoleAssignmentDto{}
	for {
		_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, request, []int{http.StatusOK, http.StatusCreated}, &response)
		if err == nil {
			return &response, nil
		}
		if !isScopeNotYetPropagated(err) {
			return nil, err
		}

		waitFor := api.DefaultRetryAfter()
		tflog.Debug(ctx, fmt.Sprintf("Scope %s is not yet available for role assignments, retrying after %s", request.Scope, waitFor))
		if sleepErr := client.Api.SleepWithContext(ctx, waitFor); sleepErr != nil {
			return nil, err
		}
	}
}

func isScopeNotYetPropagated(err error) bool {
	var httpErr customerrors.UnexpectedHttpStatusCodeError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		return false
	}
	for _, code := range scopePropagationErrorCodes {
		if strings.Contains(string(httpErr.Body), code) {
			return true
		}
	}
	return false
}

// ListRoleAssignments lists role assignments at tenant level.
func (client *Client) ListRoleAssignments(ctx context.Context) ([]RoleAssignmentDto, error) {
	return client.listRoleAssignmentsAtPath(ctx, "/authorization/roleAssignments")
}

// ListEnvironmentGroupRoleAssignments lists role assignments for an environment group.
func (client *Client) ListEnvironmentGroupRoleAssignments(ctx context.Context, environmentGroupId string) ([]RoleAssignmentDto, error) {
	path := fmt.Sprintf("/authorization/environmentGroups/%s/roleAssignments", environmentGroupId)
	return client.listRoleAssignmentsAtPath(ctx, path)
}

// ListEnvironmentRoleAssignments lists role assignments for an environment.
func (client *Client) ListEnvironmentRoleAssignments(ctx context.Context, environmentId string) ([]RoleAssignmentDto, error) {
	path := fmt.Sprintf("/authorization/environments/%s/roleAssignments", environmentId)
	return client.listRoleAssignmentsAtPath(ctx, path)
}

func (client *Client) listRoleAssignmentsAtPath(ctx context.Context, path string) ([]RoleAssignmentDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   path,
	}
	values := url.Values{}
	values.Add("api-version", roleAssignmentsApiVersion)
	apiUrl.RawQuery = values.Encode()

	response := RoleAssignmentsListDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &response)
	if err != nil {
		return nil, err
	}
	return response.Value, nil
}

// DeleteRoleAssignment deletes a role assignment at tenant level.
func (client *Client) DeleteRoleAssignment(ctx context.Context, roleAssignmentId string) error {
	path := fmt.Sprintf("/authorization/roleAssignments/%s", roleAssignmentId)
	return client.deleteRoleAssignmentAtPath(ctx, path)
}

// DeleteEnvironmentGroupRoleAssignment deletes a role assignment for an environment group.
func (client *Client) DeleteEnvironmentGroupRoleAssignment(ctx context.Context, environmentGroupId, roleAssignmentId string) error {
	path := fmt.Sprintf("/authorization/environmentGroups/%s/roleAssignments/%s", environmentGroupId, roleAssignmentId)
	return client.deleteRoleAssignmentAtPath(ctx, path)
}

// DeleteEnvironmentRoleAssignment deletes a role assignment for an environment.
func (client *Client) DeleteEnvironmentRoleAssignment(ctx context.Context, environmentId, roleAssignmentId string) error {
	path := fmt.Sprintf("/authorization/environments/%s/roleAssignments/%s", environmentId, roleAssignmentId)
	return client.deleteRoleAssignmentAtPath(ctx, path)
}

func (client *Client) deleteRoleAssignmentAtPath(ctx context.Context, path string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   path,
	}
	values := url.Values{}
	values.Add("api-version", roleAssignmentsApiVersion)
	apiUrl.RawQuery = values.Encode()

	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNoContent}, nil)
	return err
}

// ListRoleDefinitions lists available role definitions.
func (client *Client) ListRoleDefinitions(ctx context.Context) ([]RoleDefinitionDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/authorization/roleDefinitions",
	}
	values := url.Values{}
	values.Add("api-version", roleAssignmentsApiVersion)
	apiUrl.RawQuery = values.Encode()

	response := RoleDefinitionsListDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &response)
	if err != nil {
		return nil, err
	}
	return response.Value, nil
}
