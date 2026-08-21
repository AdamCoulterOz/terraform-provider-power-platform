// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetGatewayCluster retrieves information about the gateway cluster of a tenant.
func (client *Client) GetGatewayCluster(ctx context.Context, tenantId string) (*GatewayClusterDto, error) {
	tenantApiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.tenantHost(tenantId),
		Path:   "gateway/cluster",
		RawQuery: url.Values{
			"api-version": []string{"1"},
		}.Encode(),
	}

	gatewayCluster := GatewayClusterDto{}

	_, err := client.Api.Execute(ctx, client.scopes(), "GET", tenantApiUrl.String(), nil, nil, []int{http.StatusOK}, &gatewayCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetGatewayCluster API request: %w", err)
	}

	return &gatewayCluster, nil
}
