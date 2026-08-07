// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

import (
	"context"
	"net/http"
	"testing"

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

func TestUnitDeleteFabricLinkDeletesDatalakeFolderViaAthena(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerFabricLinkEnvironment(t)

	deleteCalls := 0
	httpmock.RegisterResponder(http.MethodDelete, testDeleteUrl, func(_ *http.Request) (*http.Response, error) {
		deleteCalls++
		return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
	})

	err := newTestFabricLinkClient().DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	require.NoError(t, err)
	assert.Equal(t, 1, deleteCalls, "unlink issues exactly one DELETE against the derived athena host")
}

func TestUnitDeleteFabricLinkTreatsNotFoundAsUnlinked(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerFabricLinkEnvironment(t)

	httpmock.RegisterResponder(http.MethodDelete, testDeleteUrl, func(_ *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(http.StatusNotFound, `{"error":"folder context not found"}`), nil
	})

	err := newTestFabricLinkClient().DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	require.NoError(t, err, "a missing datalakefolder means the link is already gone")
}

func TestUnitDeleteFabricLinkEmptyFolderIdErrors(t *testing.T) {
	err := newTestFabricLinkClient().DeleteFabricLink(context.Background(), testEnvironmentId, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "datalake folder id is empty")
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

func newTestFabricLinkClient() *client {
	cfg := config.ProviderConfig{
		TestMode:        true,
		TelemetryOptout: true,
		Urls: config.ProviderConfigUrls{
			BapiUrl:        "api.bap.microsoft.com",
			PowerAppsScope: "https://service.powerapps.com/.default",
		},
	}
	apiClient := api.NewApiClientBase(&cfg, api.NewAuthBase(&cfg))
	return &client{Api: apiClient}
}
