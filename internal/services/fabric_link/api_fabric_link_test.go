// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
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
	var sentOrganizationId string
	httpmock.RegisterResponder(http.MethodDelete, testDeleteUrl, func(req *http.Request) (*http.Response, error) {
		deleteCalls++
		sentOrganizationId = req.Header.Get("x-ms-organization-id")
		return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
	})

	err := newTestFabricLinkClient().DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	require.NoError(t, err)
	assert.Equal(t, testOrganizationId, sentOrganizationId, "athena resolves the organization from this header on the unlink")
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

func TestUnitDeleteFabricLinkParentEnvironmentGone(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(http.MethodGet, testBapUrl, func(_ *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(http.StatusNotFound, `{"error":{"code":"EnvironmentNotFound","message":"The environment could not be found."}}`), nil
	})

	err := newTestFabricLinkClient().DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound),
		"a parent environment deleted out of band must surface as ErrObjectNotFound so Read/Delete can treat the link as gone")
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

func TestUnitDeleteFabricLinkStopsRetryingAtTheDeleteBound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerFabricLinkEnvironment(t)

	deleteCalls := 0
	httpmock.RegisterResponder(http.MethodDelete, testDeleteUrl, func(_ *http.Request) (*http.Response, error) {
		deleteCalls++
		return httpmock.NewStringResponse(http.StatusServiceUnavailable, "athena is unavailable"), nil
	})

	fabricLinkClient := newTestFabricLinkClient()
	fabricLinkClient.deleteRetryTimeout = 50 * time.Millisecond

	started := time.Now()
	err := fabricLinkClient.DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)

	require.Error(t, err, "a service that never succeeds must not retry forever")
	assert.Less(t, time.Since(started), 30*time.Second, "the delete must give up at its own bound")
	assert.Greater(t, deleteCalls, 0, "the delete should have been attempted")
	assert.Contains(t, err.Error(), "after retrying within", "the error should say the unlink was bounded")
}
