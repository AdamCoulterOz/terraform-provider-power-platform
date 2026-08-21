// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

// GetUsers lists the systemusers of an environment.
func (client *Client) GetUsers(ctx context.Context, environmentHost string) ([]UserDto, error) {
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, "/api/data/v9.2/systemusers", nil)
	userArray := UserArrayDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &userArray)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	return userArray.Value, nil
}

// GetUserBySystemUserId reads a systemuser and its security roles by id.
func (client *Client) GetUserBySystemUserId(ctx context.Context, environmentHost, systemUserId string) (*UserDto, error) {
	values := url.Values{}
	values.Add("$expand", "systemuserroles_association($select=roleid,name,ismanaged,_businessunitid_value)")
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, "/api/data/v9.2/systemusers("+systemUserId+")", values)

	user := UserDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &user)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByAadObjectId reads a systemuser and its security roles by Entra
// object id, retrying until the user record is visible in Dataverse.
func (client *Client) GetUserByAadObjectId(ctx context.Context, environmentHost, aadObjectId string) (*UserDto, error) {
	values := url.Values{}
	values.Add("$filter", fmt.Sprintf("azureactivedirectoryobjectid eq %s", aadObjectId))
	values.Add("$expand", "systemuserroles_association($select=roleid,name,ismanaged,_businessunitid_value)")
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, "/api/data/v9.2/systemusers", values)

	const maxRetries = 30
	for attempt := 0; attempt < maxRetries; attempt++ {
		user := UserArrayDto{}
		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &user)
		if err != nil {
			return nil, err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return nil, err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return nil, err
		}

		if len(user.Value) > 0 {
			return &user.Value[0], nil
		}

		if attempt < maxRetries-1 {
			if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
				return nil, err
			}
		}
	}
	return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("Dataverse user with AAD object id '%s' not found after %d retries", aadObjectId, maxRetries))
}

// UpdateUser patches a systemuser and reads it back.
func (client *Client) UpdateUser(ctx context.Context, environmentHost, systemUserId string, userUpdate *UserDto) (*UserDto, error) {
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, "/api/data/v9.2/systemusers("+systemUserId+")", nil)

	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "PATCH", apiUrl, nil, userUpdate, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	user, err := client.GetUserBySystemUserId(ctx, environmentHost, systemUserId)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// DeleteUser deletes a systemuser.
func (client *Client) DeleteUser(ctx context.Context, environmentHost, systemUserId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/systemusers(" + systemUserId + ")",
	}

	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	} else if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return err
	}

	return nil
}

// RemoveSecurityRoles disassociates security roles from a systemuser and reads
// the user back.
func (client *Client) RemoveSecurityRoles(ctx context.Context, environmentHost, systemUserId string, securityRolesIds []string) (*UserDto, error) {
	for _, roleId := range securityRolesIds {
		apiUrl := &url.URL{
			Scheme: constants.HTTPS,
			Host:   environmentHost,
			Path:   "/api/data/v9.2/systemusers(" + systemUserId + ")/systemuserroles_association/$ref",
		}
		values := url.Values{}
		values.Add("$id", fmt.Sprintf("https://%s/api/data/v9.2/roles(%s)", environmentHost, roleId))
		apiUrl.RawQuery = values.Encode()

		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
		if err != nil {
			if strings.Contains(err.Error(), "0x80060888") && strings.Contains(err.Error(), roleId) {
				return nil, fmt.Errorf("role with id '%s' is not valid", roleId)
			}
			return nil, err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return nil, err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return nil, err
		}
	}

	user, err := client.GetUserBySystemUserId(ctx, environmentHost, systemUserId)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// AddSecurityRoles associates security roles to a systemuser and reads the
// user back.
func (client *Client) AddSecurityRoles(ctx context.Context, environmentHost, systemUserId string, securityRolesIds []string) (*UserDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/systemusers(" + systemUserId + ")/systemuserroles_association/$ref",
	}

	for _, roleId := range securityRolesIds {
		roleToassociate := map[string]any{
			"@odata.id": fmt.Sprintf("https://%s/api/data/v9.2/roles(%s)", environmentHost, roleId),
		}
		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", apiUrl.String(), nil, roleToassociate, []int{http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
		if err != nil {
			if strings.Contains(err.Error(), "0x80060888") && strings.Contains(err.Error(), roleId) {
				return nil, fmt.Errorf("role with id '%s' is not valid", roleId)
			}
			return nil, err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return nil, err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return nil, err
		}
	}
	user, err := client.GetUserBySystemUserId(ctx, environmentHost, systemUserId)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetSecurityRoles lists the security roles of an environment, optionally
// filtered to one business unit.
func (client *Client) GetSecurityRoles(ctx context.Context, environmentHost, businessUnitId string) ([]SecurityRoleDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/roles",
	}
	if businessUnitId != "" {
		var values = url.Values{}
		values.Add("$filter", fmt.Sprintf("_businessunitid_value eq %s", businessUnitId))
		apiUrl.RawQuery = values.Encode()
	}
	securityRoleArray := SecurityRoleArrayDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &securityRoleArray)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	return securityRoleArray.Value, nil
}
