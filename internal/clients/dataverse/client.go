// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
)

// Client calls per-environment Dataverse Web API endpoints. It is a thin,
// stateless wrapper over the shared transport in internal/clients/api.
type Client struct {
	Api *api.Client
}

// NewClient returns a Dataverse client using the given shared transport.
func NewClient(apiClient *api.Client) *Client {
	return &Client{Api: apiClient}
}

// scopes returns the OAuth scopes for a given environment host.
func scopes(environmentHost string) []string {
	return []string{"https://" + environmentHost + "/.default"}
}
