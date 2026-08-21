// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package connectors

import (
	"context"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/powerapps"
)

func newConnectorsClient(apiClient *api.Client) client {
	return client{
		Client:                  powerapps.NewClient(apiClient),
		connectorMetadataClient: bapi.NewClient(apiClient),
	}
}

type client struct {
	*powerapps.Client
	connectorMetadataClient *bapi.Client
}

// GetConnectors merges the PowerApps connector list of an environment with the
// BAPI governance metadata (unblockable and virtual connectors), so the
// composition lives here in the service.
func (client *client) GetConnectors(ctx context.Context, environmentId string) ([]powerapps.ConnectorDto, error) {
	connectors, err := client.Client.GetConnectors(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	unblockableConnectorArray, err := client.connectorMetadataClient.GetUnblockableConnectorsMetadata(ctx)
	if err != nil {
		return nil, err
	}

	for inx, connector := range connectors {
		for _, unblockableConnector := range unblockableConnectorArray {
			if connector.Id == unblockableConnector.Id {
				connectors[inx].Properties.Unblockable = unblockableConnector.Metadata.Unblockable
			}
		}
	}

	virtualConnectorArray, err := client.connectorMetadataClient.GetVirtualConnectorsMetadata(ctx)
	if err != nil {
		return nil, err
	}
	for _, virutualConnector := range virtualConnectorArray {
		connectors = append(connectors, powerapps.ConnectorDto{
			Id:   virutualConnector.Id,
			Name: virutualConnector.Metadata.Name,
			Type: virutualConnector.Metadata.Type,
			Properties: powerapps.ConnectorPropertiesDto{
				DisplayName: virutualConnector.Metadata.DisplayName,
				Unblockable: false,
				Tier:        "Built-in",
				Publisher:   "Microsoft",
				Description: "",
			},
		})
	}

	for inx, connector := range connectors {
		nameSplit := strings.Split(connector.Id, "/")
		connectors[inx].Name = nameSplit[len(nameSplit)-1]
	}

	return connectors, nil
}
