// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

// Client calls the Power Platform API. It is a thin, stateless wrapper over
// the shared transport in internal/clients/api.
type Client struct {
	Api *api.Client
}

// NewClient returns a Power Platform API client using the given shared transport.
func NewClient(apiClient *api.Client) *Client {
	return &Client{Api: apiClient}
}

// scopes returns the OAuth scopes used to authenticate Power Platform API requests.
func (client *Client) scopes() []string {
	return []string{client.Api.GetConfig().Urls.PowerPlatformScope}
}

// host returns the Power Platform API host for the configured cloud.
func (client *Client) host() string {
	return client.Api.GetConfig().Urls.PowerPlatformUrl
}

// tenantHost returns the tenant-scoped Power Platform API host for the configured cloud.
func (client *Client) tenantHost(tenantId string) string {
	return helpers.BuildTenantHostUri(tenantId, client.Api.GetConfig().Urls.PowerPlatformUrl)
}
