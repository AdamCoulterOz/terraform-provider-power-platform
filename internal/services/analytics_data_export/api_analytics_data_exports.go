// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package analytics_data_export

import (
	"context"
	"fmt"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/analytics"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/ppapi"
)

func NewAnalyticsExportClient(apiClient *api.Client, tenantClient *bapi.Client) Client {
	return Client{
		Client:     analytics.NewClient(apiClient),
		TenantApi:  tenantClient,
		GatewayApi: ppapi.NewClient(apiClient),
	}
}

type Client struct {
	*analytics.Client
	TenantApi  *bapi.Client
	GatewayApi *ppapi.Client
}

// GetGatewayCluster retrieves information about the tenant's gateway cluster.
func (client *Client) GetGatewayCluster(ctx context.Context) (*ppapi.GatewayClusterDto, error) {
	// Get tenant information using the tenant client
	tenantInfo, err := client.TenantApi.GetTenant(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant information: %w", err)
	}

	return client.GatewayApi.GetGatewayCluster(ctx, tenantInfo.TenantId)
}

func (client *Client) GetAnalyticsDataExport(ctx context.Context) ([]analytics.AnalyticsDataDto, error) {
	// Get the gateway cluster
	gatewayCluster, err := client.GetGatewayCluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway cluster: %w", err)
	}

	// Get the analytics data exports from the region of the gateway cluster
	return client.Client.GetAnalyticsDataExport(ctx, gatewayCluster.GeoName)
}
