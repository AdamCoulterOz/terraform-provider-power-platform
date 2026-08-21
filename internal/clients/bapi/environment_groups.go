// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

// CreateEnvironmentGroup creates an environment group.
func (client *Client) CreateEnvironmentGroup(ctx context.Context, environmentGroup EnvironmentGroupDto) (*EnvironmentGroupDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/Microsoft.BusinessAppPlatform/environmentGroups",
	}

	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	newEnvironmentGroup := EnvironmentGroupDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, environmentGroup, []int{http.StatusCreated}, &newEnvironmentGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment group: %w", err)
	}

	return &newEnvironmentGroup, nil
}

// DeleteEnvironmentGroup deletes an environment group.
func (client *Client) DeleteEnvironmentGroup(ctx context.Context, environmentGroupId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/Microsoft.BusinessAppPlatform/environmentGroups/" + environmentGroupId,
	}

	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	resp, err := client.Api.Execute(ctx, client.scopes(), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusConflict}, nil)
	if err != nil {
		if resp.HttpResponse.StatusCode == http.StatusConflict {
			if len(resp.BodyAsBytes) == 0 {
				return errors.New("failed to delete environment group")
			}

			body := string(resp.BodyAsBytes[:])
			if strings.Contains(body, "EnvironmentsInEnvironmentGroup") {
				return customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_ENVIRONMENTS_IN_ENV_GROUP), "Failed to delete environment group because it contains environments")
			} else if strings.Contains(body, "PolicyAssignedToEnvironmentGroup") {
				return customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_POLICY_ASSIGNED_TO_ENV_GROUP), "Failed to delete environment group because it has a policy assigned")
			}
			return errors.New(body)
		}
		return err
	}
	return nil
}

// UpdateEnvironmentGroup updates an environment group.
func (client *Client) UpdateEnvironmentGroup(ctx context.Context, environmentGroupId string, environmentGroup EnvironmentGroupDto) (*EnvironmentGroupDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/Microsoft.BusinessAppPlatform/environmentGroups/" + environmentGroupId,
	}

	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	updatedEnvironmentGroup := EnvironmentGroupDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "PUT", apiUrl.String(), nil, environmentGroup, []int{http.StatusOK}, &updatedEnvironmentGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to update environment group: %w", err)
	}

	return &updatedEnvironmentGroup, nil
}

// GetEnvironmentGroup gets an environment group, returning nil when it does not exist.
func (client *Client) GetEnvironmentGroup(ctx context.Context, environmentGroupId string) (*EnvironmentGroupDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/Microsoft.BusinessAppPlatform/environmentGroups/" + environmentGroupId,
	}

	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	environmentGroup := EnvironmentGroupDto{}
	httpResponse, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNotFound}, &environmentGroup)
	if httpResponse.HttpResponse.StatusCode == http.StatusNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get environment group: %w", err)
	}

	return &environmentGroup, nil
}

// GetEnvironmentsInEnvironmentGroup lists the environments whose parent is the
// given environment group.
func (client *Client) GetEnvironmentsInEnvironmentGroup(ctx context.Context, environmentGroupId string) ([]EnvironmentDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "providers/Microsoft.BusinessAppPlatform/scopes/admin/environments",
	}

	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	values.Add("$filter", fmt.Sprintf("properties/parentEnvironmentGroup/id eq %s", environmentGroupId))
	apiUrl.RawQuery = values.Encode()

	environments := EnvironmentArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &environments)
	if err != nil {
		return nil, fmt.Errorf("failed to get environments in environment group: %w", err)
	}

	return environments.Value, nil
}
