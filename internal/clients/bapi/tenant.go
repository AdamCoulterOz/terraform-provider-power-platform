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

type TenantDto struct {
	TenantId                         string `json:"tenantId"`
	State                            string `json:"state"`
	Location                         string `json:"location"`
	AadCountryGeo                    string `json:"aadCountryGeo"`
	DataStorageGeo                   string `json:"dataStorageGeo"`
	DefaultEnvironmentGeo            string `json:"defaultEnvironmentGeo"`
	AadDataBoundary                  string `json:"aadDataBoundary"`
	FedRAMPHighCertificationRequired bool   `json:"fedRAMPHighCertificationRequired"`
}

// GetTenant reads the tenant behind the current credentials.
func (client *Client) GetTenant(ctx context.Context) (*TenantDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/Microsoft.BusinessAppPlatform/tenant",
	}

	values := url.Values{}
	values.Add("api-version", "2021-04-01")
	apiUrl.RawQuery = values.Encode()

	var dto TenantDto

	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &dto)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GET tenant API call: %w", err)
	}

	return &dto, nil
}
