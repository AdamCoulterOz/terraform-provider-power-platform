// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package connection

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/ppapi"
)

func newConnectionsClient(apiClient *api.Client) client {
	return client{
		Client: ppapi.NewClient(apiClient),
	}
}

// client embeds the Power Platform API boundary client that owns the
// connection and connection-share operations. The cross-boundary environment
// existence checks use the *bapi.Client held as a named field on the resource
// structs.
type client struct {
	*ppapi.Client
}
