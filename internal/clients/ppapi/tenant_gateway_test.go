// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUnitGetGatewayCluster_UsesTenantHost(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// The gateway cluster endpoint is served from the tenant-scoped host.
	httpmock.RegisterResponderWithQuery("GET", "https://"+testTenantHost+"/gateway/cluster",
		url.Values{"api-version": []string{"1"}},
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, GatewayClusterDto{
				ClusterNumber: "42",
				GeoName:       "AU",
				ClusterType:   "Commercial",
			})
		})

	gatewayCluster, err := newTestClient(t).GetGatewayCluster(context.Background(), testTenantId)

	assert.NoError(t, err)
	assert.Equal(t, "42", gatewayCluster.ClusterNumber)
	assert.Equal(t, "AU", gatewayCluster.GeoName)
	assert.Equal(t, "Commercial", gatewayCluster.ClusterType)
}

func TestUnitGetGatewayCluster_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://"+testTenantHost+"/gateway/cluster",
		url.Values{"api-version": []string{"1"}},
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	gatewayCluster, err := newTestClient(t).GetGatewayCluster(context.Background(), testTenantId)

	assert.Nil(t, gatewayCluster)
	assert.ErrorContains(t, err, "failed to execute GetGatewayCluster API request")
}
