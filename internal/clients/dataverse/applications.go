// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

// GetRootBusinessUnitId reads the root business unit of an environment. The
// environmentId is used only to identify the environment in error messages.
func (client *Client) GetRootBusinessUnitId(ctx context.Context, environmentHost, environmentId string) (string, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/businessunits", constants.DATAVERSE_API_VERSION),
	}
	values := url.Values{}
	values.Add("$select", "businessunitid,name")
	values.Add("$filter", "parentbusinessunitid eq null")
	apiUrl.RawQuery = values.Encode()

	var response ApplicationBusinessUnitArrayDto
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return "", err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return "", err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return "", err
	}
	if len(response.Value) == 0 {
		return "", customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("root business unit not found in environment '%s'", environmentId))
	}
	if len(response.Value) > 1 {
		return "", fmt.Errorf("expected exactly one root business unit in environment '%s', got %d", environmentId, len(response.Value))
	}

	return response.Value[0].BusinessUnitId, nil
}

// GetApplicationUser returns the single active application user for an
// application id, or an ErrObjectNotFound / ambiguity error. The environmentId
// is used only to identify the environment in error messages.
func (client *Client) GetApplicationUser(ctx context.Context, environmentHost, environmentId, applicationId string) (*ApplicationUserDto, error) {
	users, err := client.GetApplicationUsersByApplicationId(ctx, environmentHost, environmentId, applicationId)
	if err != nil {
		return nil, err
	}

	active := make([]ApplicationUserDto, 0, len(users))
	for _, user := range users {
		if user.DeletedState == 0 {
			active = append(active, user)
		}
	}

	switch len(active) {
	case 0:
		return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("application user '%s' not found in environment '%s'", applicationId, environmentId))
	case 1:
		sort.Slice(active[0].SecurityRoles, func(i, j int) bool {
			return active[0].SecurityRoles[i].RoleId < active[0].SecurityRoles[j].RoleId
		})
		return &active[0], nil
	default:
		ids := make([]string, 0, len(active))
		for _, user := range active {
			ids = append(ids, user.SystemUserId)
		}
		sort.Strings(ids)
		return nil, fmt.Errorf(
			"multiple active application users found for application '%s' in environment '%s': %s",
			applicationId,
			environmentId,
			strings.Join(ids, ", "))
	}
}

// GetApplicationUsersByApplicationId lists the systemuser rows (active and
// deleted) for an application id. The environmentId is used only to identify
// the environment in error messages.
func (client *Client) GetApplicationUsersByApplicationId(ctx context.Context, environmentHost, environmentId, applicationId string) ([]ApplicationUserDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/systemusers", constants.DATAVERSE_API_VERSION),
	}
	values := url.Values{}
	values.Add("$select", "applicationid,systemuserid,fullname,isdisabled,deletedstate,_businessunitid_value")
	values.Add("$expand", "systemuserroles_association($select=roleid,name,_businessunitid_value)")
	values.Add("$filter", fmt.Sprintf("applicationid eq %s", applicationId))
	apiUrl.RawQuery = values.Encode()

	var response ApplicationUsersResponseDto
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if resp.HttpResponse.StatusCode == http.StatusNotFound || len(response.Value) == 0 {
		return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("application user '%s' not found in environment '%s'", applicationId, environmentId))
	}

	return response.Value, nil
}

// CreateApplicationUser creates a non-interactive application user
// (accessmode 4) in a business unit and returns the created systemuser id
// extracted from the OData-EntityId response header.
func (client *Client) CreateApplicationUser(ctx context.Context, environmentHost, applicationId, businessUnitId string) (string, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/systemusers", constants.DATAVERSE_API_VERSION),
	}

	requestBody := map[string]any{
		"applicationid":             applicationId,
		"accessmode":                4,
		"isdisabled":                false,
		"businessunitid@odata.bind": fmt.Sprintf("/businessunits(%s)", businessUnitId),
	}

	response, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", apiUrl.String(), nil, requestBody, []int{http.StatusNoContent, http.StatusCreated}, nil)
	if err != nil {
		return "", err
	}
	if err := client.Api.HandleForbiddenResponse(response); err != nil {
		return "", err
	}

	entityID := response.GetHeader(constants.HEADER_ODATA_ENTITY_ID)
	if entityID == "" {
		return "", errors.New("no entity record id returned from the odata-entityid header")
	}

	re := regexp.MustCompile("[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	match := re.FindAllStringSubmatch(entityID, -1)
	if len(match) == 0 {
		return "", errors.New("no entity record id returned from the odata-entityid header")
	}

	return match[len(match)-1][0], nil
}

// SetApplicationUserDisabledState enables or disables a systemuser.
func (client *Client) SetApplicationUserDisabledState(ctx context.Context, environmentHost, systemUserId, applicationId string, disabled bool) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/systemusers(%s)", constants.DATAVERSE_API_VERSION, systemUserId),
	}

	requestBody := map[string]any{
		"isdisabled":    disabled,
		"applicationid": applicationId,
	}

	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "PATCH", apiUrl.String(), nil, requestBody, []int{http.StatusNoContent, http.StatusOK}, nil)
	if err != nil {
		return err
	}
	return client.Api.HandleForbiddenResponse(resp)
}

// GetPrincipalBySystemUserId reads a single active systemuser (principal) by
// id, including its security roles.
func (client *Client) GetPrincipalBySystemUserId(ctx context.Context, environmentHost, systemUserId string) (*ApplicationUserDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/systemusers(%s)", constants.DATAVERSE_API_VERSION, systemUserId),
	}
	values := url.Values{}
	values.Add("$select", "applicationid,systemuserid,fullname,isdisabled,deletedstate,_businessunitid_value")
	values.Add("$expand", "systemuserroles_association($select=roleid,name,_businessunitid_value)")
	apiUrl.RawQuery = values.Encode()

	var response ApplicationUserDto
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if resp.HttpResponse.StatusCode == http.StatusNotFound {
		return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("principal not found for system user ID %s", systemUserId))
	}
	if response.DeletedState != 0 {
		return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("principal not found for system user ID %s", systemUserId))
	}

	sort.Slice(response.SecurityRoles, func(i, j int) bool {
		return response.SecurityRoles[i].RoleId < response.SecurityRoles[j].RoleId
	})

	return &response, nil
}

// GetDataverseSecurityRoles lists security roles, optionally filtered to one
// business unit.
func (client *Client) GetDataverseSecurityRoles(ctx context.Context, environmentHost, businessUnitId string) ([]ApplicationSecurityRoleDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/roles", constants.DATAVERSE_API_VERSION),
	}
	values := url.Values{}
	values.Add("$select", "roleid,name,_businessunitid_value")
	if businessUnitId != "" {
		values.Add("$filter", fmt.Sprintf("_businessunitid_value eq %s", businessUnitId))
	}
	apiUrl.RawQuery = values.Encode()

	var response ApplicationSecurityRoleArrayDto
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	return response.Value, nil
}

// ResolveSecurityRoleNames resolves role names to their unique roles within a
// business unit, failing on missing or ambiguous names.
func (client *Client) ResolveSecurityRoleNames(ctx context.Context, environmentHost, businessUnitId string, roleNames []string) ([]ApplicationSecurityRoleDto, error) {
	allRoles, err := client.GetDataverseSecurityRoles(ctx, environmentHost, businessUnitId)
	if err != nil {
		return nil, err
	}

	rolesByName := make(map[string][]ApplicationSecurityRoleDto)
	for _, role := range allRoles {
		rolesByName[role.Name] = append(rolesByName[role.Name], role)
	}

	resolved := make([]ApplicationSecurityRoleDto, 0, len(roleNames))
	for _, roleName := range roleNames {
		matches := rolesByName[roleName]
		if len(matches) == 0 {
			return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("security role '%s' not found in business unit '%s'", roleName, businessUnitId))
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("security role '%s' is ambiguous in business unit '%s'", roleName, businessUnitId)
		}
		resolved = append(resolved, matches[0])
	}

	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].RoleId < resolved[j].RoleId
	})

	return resolved, nil
}

// AddPrincipalSecurityRoles associates security roles with a systemuser.
func (client *Client) AddPrincipalSecurityRoles(ctx context.Context, environmentHost, systemUserId string, roleIds []string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/systemusers(%s)/systemuserroles_association/$ref", constants.DATAVERSE_API_VERSION, systemUserId),
	}

	for _, roleId := range roleIds {
		roleToAssociate := map[string]any{
			"@odata.id": fmt.Sprintf("https://%s/api/data/%s/roles(%s)", environmentHost, constants.DATAVERSE_API_VERSION, roleId),
		}
		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", apiUrl.String(), nil, roleToAssociate, []int{http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
		if err != nil {
			return err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return err
		}
	}

	return nil
}

// RemovePrincipalSecurityRoles disassociates security roles from a systemuser.
func (client *Client) RemovePrincipalSecurityRoles(ctx context.Context, environmentHost, systemUserId string, roleIds []string) error {
	for _, roleId := range roleIds {
		apiUrl := &url.URL{
			Scheme: constants.HTTPS,
			Host:   environmentHost,
			Path:   fmt.Sprintf("/api/data/%s/systemusers(%s)/systemuserroles_association/$ref", constants.DATAVERSE_API_VERSION, systemUserId),
		}
		values := url.Values{}
		values.Add("$id", fmt.Sprintf("https://%s/api/data/%s/roles(%s)", environmentHost, constants.DATAVERSE_API_VERSION, roleId))
		apiUrl.RawQuery = values.Encode()

		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
		if err != nil {
			return err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return err
		}
	}

	return nil
}

// DeleteSystemUser deletes a systemuser row. Callers choose the acceptable
// status codes so a permanent (second-pass) delete can tolerate 404.
func (client *Client) DeleteSystemUser(ctx context.Context, environmentHost, systemUserId string, expectedStatusCodes []int) error {
	// Create the Dataverse Web API URL to delete the system user
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/systemusers(%s)", constants.DATAVERSE_API_VERSION, systemUserId),
	}

	// Make the request
	_, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiUrl.String(), nil, nil, expectedStatusCodes, nil)
	if err != nil {
		return err
	}

	return nil
}
