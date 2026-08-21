// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package copilot

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// Client calls the Copilot Studio API. It is a thin, stateless wrapper over
// the shared transport in internal/clients/api.
type Client struct {
	Api *api.Client
}

// NewClient returns a Copilot Studio client using the given shared transport.
func NewClient(apiClient *api.Client) *Client {
	return &Client{Api: apiClient}
}

// scopes returns the OAuth scopes used to authenticate Copilot Studio requests.
func (client *Client) scopes() []string {
	return []string{constants.COPILOT_SCOPE}
}
