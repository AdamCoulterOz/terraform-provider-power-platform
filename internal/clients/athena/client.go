// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package athena

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// Client calls the Power Platform admin center (Athena) API. It is a thin,
// stateless wrapper over the shared transport in internal/clients/api.
type Client struct {
	Api *api.Client
}

// NewClient returns an Athena client using the given shared transport.
func NewClient(apiClient *api.Client) *Client {
	return &Client{Api: apiClient}
}

// scopes returns the OAuth scopes used to authenticate Athena requests.
func (client *Client) scopes() []string {
	return []string{constants.PPAC_SCOPE}
}

// host returns the admin center API host for the configured cloud.
func (client *Client) host() string {
	return client.Api.GetConfig().Urls.AdminPowerPlatformUrl
}
