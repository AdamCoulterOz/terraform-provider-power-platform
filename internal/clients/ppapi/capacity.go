// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetTenantCapacity reads the capacity details of the tenant.
func (client *Client) GetTenantCapacity(ctx context.Context) (*TenantCapacityDetailsDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/licensing/tenantCapacity",
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.APPLICATION_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	var dto TenantCapacityDetailsDto

	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &dto)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant capacity details: %w", err)
	}

	return &dto, nil
}
