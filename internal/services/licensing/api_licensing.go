// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package licensing

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/ppapi"
)

func NewLicensingClient(apiClient *api.Client) Client {
	return Client{
		Client: ppapi.NewClient(apiClient),
	}
}

// Client is the licensing service client. Billing policy operations are
// promoted from the embedded Power Platform API client.
type Client struct {
	*ppapi.Client
}
