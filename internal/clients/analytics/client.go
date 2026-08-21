// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package analytics

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
)

// Client calls the Power Platform admin analytics API. It is a thin,
// stateless wrapper over the shared transport in internal/clients/api.
type Client struct {
	Api *api.Client
}

// NewClient returns an admin analytics client using the given shared transport.
func NewClient(apiClient *api.Client) *Client {
	return &Client{Api: apiClient}
}

// scopes returns the OAuth scopes used to authenticate admin analytics requests.
func (client *Client) scopes() []string {
	return []string{client.Api.GetConfig().Urls.AnalyticsScope}
}
