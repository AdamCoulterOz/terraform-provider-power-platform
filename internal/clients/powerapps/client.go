// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerapps

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
)

// Client calls the PowerApps API. It is a thin, stateless wrapper over the
// shared transport in internal/clients/api.
type Client struct {
	Api *api.Client
}

// NewClient returns a PowerApps API client using the given shared transport.
func NewClient(apiClient *api.Client) *Client {
	return &Client{Api: apiClient}
}

// scopes returns the OAuth scopes used to authenticate PowerApps API requests.
func (client *Client) scopes() []string {
	return []string{client.Api.GetConfig().Urls.PowerAppsScope}
}

// host returns the PowerApps API host for the configured cloud.
func (client *Client) host() string {
	return client.Api.GetConfig().Urls.PowerAppsUrl
}
