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

// GetUnblockableConnectorsMetadata lists the governance metadata of connectors
// that cannot be blocked in a Data Loss Prevention policy.
func (client *Client) GetUnblockableConnectorsMetadata(ctx context.Context) ([]UnblockableConnectorDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/PowerPlatform.Governance/v1/connectors/metadata/unblockable",
	}
	unblockableConnectorArray := []UnblockableConnectorDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &unblockableConnectorArray)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unblockable connectors metadata: %w", err)
	}
	return unblockableConnectorArray, nil
}

// GetVirtualConnectorsMetadata lists the governance metadata of virtual
// connectors.
func (client *Client) GetVirtualConnectorsMetadata(ctx context.Context) ([]VirtualConnectorDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/PowerPlatform.Governance/v1/connectors/metadata/virtual",
	}
	virtualConnectorArray := []VirtualConnectorDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &virtualConnectorArray)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch virtual connectors metadata: %w", err)
	}
	return virtualConnectorArray, nil
}
