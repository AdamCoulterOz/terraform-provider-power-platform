// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEnvironmentId  = "00000000-0000-0000-0000-000000000001"
	testOrganizationId = "00000000-0000-0000-0000-000000000002"
	testFolderId       = "00000000-0000-0000-0000-000000000003"
	testBapUrl         = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/" + testEnvironmentId + "?api-version=2020-10-01-alpha"
	testDeleteUrl      = "https://athenawebservice.eau-il301.gateway.prod.island.powerapps.com/environment/" + testEnvironmentId + "/lakehouseArtifacts/" + testFolderId + "?dxt=false"
)

func TestUnitDeleteFabricLinkSendsOrganizationHeader(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerFabricLinkEnvironment(t)

	deleteCalls := 0
	httpmock.RegisterResponder(http.MethodDelete, testDeleteUrl, func(req *http.Request) (*http.Response, error) {
		deleteCalls++
		assert.Equal(t, testOrganizationId, req.Header.Get(athenaOrganizationIdHeader))
		return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
	})

	err := newTestFabricLinkClient(time.Second).DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	require.NoError(t, err)
	assert.Equal(t, 1, deleteCalls)
}

func TestUnitDeleteFabricLinkDoesNotTreatNotFoundAsDeleted(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerFabricLinkEnvironment(t)

	httpmock.RegisterResponder(http.MethodDelete, testDeleteUrl, func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, testOrganizationId, req.Header.Get(athenaOrganizationIdHeader))
		return httpmock.NewStringResponse(http.StatusNotFound, `{"error":"folder context not found"}`), nil
	})

	err := newTestFabricLinkClient(time.Second).DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "last HTTP status 404")
	assert.Contains(t, err.Error(), "folder context not found")
}

func TestUnitDeleteFabricLinkRetriesWithinBoundAndReportsLastFailure(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerFabricLinkEnvironment(t)

	deleteCalls := 0
	httpmock.RegisterResponder(http.MethodDelete, testDeleteUrl, func(req *http.Request) (*http.Response, error) {
		deleteCalls++
		assert.Equal(t, testOrganizationId, req.Header.Get(athenaOrganizationIdHeader))
		return httpmock.NewStringResponse(http.StatusServiceUnavailable, `{"error":"athena unavailable"}`), nil
	})

	started := time.Now()
	err := newTestFabricLinkClient(10*time.Millisecond).DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	elapsed := time.Since(started)

	require.Error(t, err)
	assert.Greater(t, deleteCalls, 1, "retryable responses should be retried within the bound")
	assert.Less(t, elapsed, time.Second, "the local delete retry ceiling must stop the loop")
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Contains(t, err.Error(), "last HTTP status 503")
	assert.Contains(t, err.Error(), "athena unavailable")
}

func registerFabricLinkEnvironment(t *testing.T) {
	t.Helper()
	httpmock.RegisterResponder(http.MethodGet, testBapUrl, func(_ *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(http.StatusOK, `{
			"properties": {
				"cluster": { "uriSuffix": "au-il301.gateway.prod.island" },
				"linkedEnvironmentMetadata": {
					"resourceId": "`+testOrganizationId+`",
					"instanceUrl": "https://example.crm6.dynamics.com/"
				}
			}
		}`), nil
	})
}

func newTestFabricLinkClient(deleteRetryTimeout time.Duration) *client {
	cfg := config.ProviderConfig{
		TestMode:        true,
		TelemetryOptout: true,
		Urls: config.ProviderConfigUrls{
			BapiUrl:        "api.bap.microsoft.com",
			PowerAppsScope: "https://service.powerapps.com/.default",
		},
	}
	apiClient := api.NewApiClientBase(&cfg, api.NewAuthBase(&cfg))
	return &client{
		Api:                apiClient,
		deleteRetryTimeout: deleteRetryTimeout,
	}
}
