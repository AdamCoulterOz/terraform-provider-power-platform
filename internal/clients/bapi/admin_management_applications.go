// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetAdminApplication reads a registered admin management application by its
// client id.
func (client *Client) GetAdminApplication(ctx context.Context, clientId string) (*AdminManagementApplicationDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/adminApplications/%s", clientId),
		RawQuery: url.Values{
			constants.API_VERSION_PARAM: []string{constants.ADMIN_MANAGEMENT_APP_API_VERSION},
		}.Encode(),
	}

	var adminApp AdminManagementApplicationDto
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &adminApp)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin app %s: %w", clientId, err)
	}

	// Returning pointer to local variable is acceptable for small DTOs
	return &adminApp, err
}

// RegisterAdminApplication registers a service principal as an admin
// management application.
func (client *Client) RegisterAdminApplication(ctx context.Context, clientId string) (*AdminManagementApplicationDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/adminApplications/%s", clientId),
		RawQuery: url.Values{
			constants.API_VERSION_PARAM: []string{constants.ADMIN_MANAGEMENT_APP_API_VERSION},
		}.Encode(),
	}

	var adminApp AdminManagementApplicationDto
	_, err := client.Api.Execute(ctx, client.scopes(), "PUT", apiUrl.String(), nil, nil, []int{http.StatusOK}, &adminApp)
	if err != nil {
		return nil, fmt.Errorf("failed to register admin app %s: %w", clientId, err)
	}

	// Returning pointer to local variable is acceptable for small DTOs
	return &adminApp, err
}

// UnregisterAdminApplication removes an admin management application
// registration.
func (client *Client) UnregisterAdminApplication(ctx context.Context, clientId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/adminApplications/%s", clientId),
		RawQuery: url.Values{
			constants.API_VERSION_PARAM: []string{constants.ADMIN_MANAGEMENT_APP_API_VERSION},
		}.Encode(),
	}

	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNoContent}, nil)
	if err != nil {
		return fmt.Errorf("failed to unregister admin app %s: %w", clientId, err)
	}

	return nil
}
