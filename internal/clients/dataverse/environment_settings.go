// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

// GetEnvironmentBackendSettings reads the environment's backend settings via
// the RetrieveSettingList() function.
func (client *Client) GetEnvironmentBackendSettings(ctx context.Context, environmentHost string) (*EnvironmentBackendSettingsValueDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.0/RetrieveSettingList()",
	}

	environmentBackendSettings := EnvironmentBackendSettingsValueDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &environmentBackendSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to execute API request for environment backend settings %s: %w", environmentHost, err)
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	return &environmentBackendSettings, nil
}

// GetEnvironmentOrgSettings reads the environment's organization settings.
func (client *Client) GetEnvironmentOrgSettings(ctx context.Context, environmentHost string) (*EnvironmentOrgSettingsDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.0/organizations",
	}

	environmentOrgSettings := EnvironmentOrgSettingsValueDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &environmentOrgSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to execute API request for environment org settings %s: %w", environmentHost, err)
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	return &environmentOrgSettings.Value[0], nil
}

// UpdateEnvironmentOrgSettings patches the environment's organization settings
// and reads them back.
func (client *Client) UpdateEnvironmentOrgSettings(ctx context.Context, environmentHost string, orgSettings EnvironmentOrgSettingsDto) (*EnvironmentOrgSettingsDto, error) {
	orgSettingsResponse, err := client.GetEnvironmentOrgSettings(ctx, environmentHost)
	if err != nil {
		return nil, err
	}

	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/v9.0/organizations(%s)", *orgSettingsResponse.OrganizationId),
	}

	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "PATCH", apiUrl.String(), nil, orgSettings, []int{http.StatusNoContent, http.StatusInternalServerError, http.StatusForbidden, http.StatusNotFound}, nil)
	if resp != nil && resp.HttpResponse.StatusCode == http.StatusInternalServerError {
		return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_ENVIRONMENT_SETTINGS_FAILED), string(resp.BodyAsBytes))
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute API request for updating environment org settings %s: %w", environmentHost, err)
	}

	return client.GetEnvironmentOrgSettings(ctx, environmentHost)
}

// UpdateEnvironmentBackendSettings saves each backend setting via the
// SaveSettingValue() action and reads the settings back.
func (client *Client) UpdateEnvironmentBackendSettings(ctx context.Context, environmentHost string, backendSettings EnvironmentBackendSettingsValueDto) (*EnvironmentBackendSettingsValueDto, error) {
	for _, setting := range backendSettings.SettingDetailCollection {
		if setting.Name == "" || setting.Value == "" {
			continue
		}

		apiUrl := &url.URL{
			Scheme: constants.HTTPS,
			Host:   environmentHost,
			Path:   "/api/data/v9.0/SaveSettingValue()",
		}

		settingValue := SetSettingValueDto{
			SettingName: setting.Name,
			Value:       setting.Value,
		}

		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", apiUrl.String(), nil, settingValue, []int{http.StatusNoContent, http.StatusInternalServerError, http.StatusForbidden, http.StatusNotFound}, nil)
		if resp != nil && resp.HttpResponse.StatusCode == http.StatusInternalServerError {
			return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_ENVIRONMENT_SETTINGS_FAILED), string(resp.BodyAsBytes))
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return nil, err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, fmt.Errorf("failed to execute API request for updating environment backend settings %s: %w", environmentHost, err)
		}
	}
	return client.GetEnvironmentBackendSettings(ctx, environmentHost)
}
