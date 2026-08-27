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
	testEnvironmentId   = "00000000-0000-0000-0000-000000000001"
	testOrganizationId  = "00000000-0000-0000-0000-000000000002"
	testFolderId        = "00000000-0000-0000-0000-000000000003"
	testBapUrl          = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/" + testEnvironmentId + "?api-version=2023-06-01"
	testOrganizationUrl = "https://example.crm6.dynamics.com/"
	testProfilesUrl     = "https://example.crm6.dynamics.com/api/data/v9.1/synapselinkprofiles"
	testFoldersUrl      = "https://example.crm6.dynamics.com/api/data/v9.1/datalakefolders?$select=datalakefolderid,datalakefolder_uniquename"
	testDeleteUrl       = "https://athenawebservice.eau-il301.gateway.prod.island.powerapps.com/environment/" + testEnvironmentId + "/lakehouseArtifacts/" + testFolderId + "?dxt=false"
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
				"azureRegion": "australiaeast",
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

func TestUnitAthenaHostDerivesThePrefixFromTheEnvironmentRegion(t *testing.T) {
	// The prefix is the region's compass direction with the geography dropped; the geography is
	// already in the cluster suffix. eau-, seau-, eus-, wus-, neu- and weu- are DNS-confirmed; the
	// rest follow the same rule (one letter per compass word).
	tests := []struct {
		name             string
		azureRegion      string
		clusterUriSuffix string
		expectedHost     string
	}{
		{"australia east", "australiaeast", "au-il301.gateway.prod.island", "athenawebservice.eau-il301.gateway.prod.island.powerapps.com"},
		{"australia southeast", "australiasoutheast", "au-il301.gateway.prod.island", "athenawebservice.seau-il301.gateway.prod.island.powerapps.com"},
		{"east us", "eastus", "us-il101.gateway.prod.island", "athenawebservice.eus-il101.gateway.prod.island.powerapps.com"},
		{"west us", "westus", "us-il101.gateway.prod.island", "athenawebservice.wus-il101.gateway.prod.island.powerapps.com"},
		{"north europe", "northeurope", "eu-il101.gateway.prod.island", "athenawebservice.neu-il101.gateway.prod.island.powerapps.com"},
		{"west europe", "westeurope", "eu-il101.gateway.prod.island", "athenawebservice.weu-il101.gateway.prod.island.powerapps.com"},
		{"ordinal regions keep their direction", "eastus2", "us-il102.gateway.prod.island", "athenawebservice.eus-il102.gateway.prod.island.powerapps.com"},
		{"direction after the geography wins", "southafricanorth", "za-il101.gateway.prod.island", "athenawebservice.nza-il101.gateway.prod.island.powerapps.com"},
		{"two-word direction before the geography", "southeastasia", "as-il101.gateway.prod.island", "athenawebservice.seas-il101.gateway.prod.island.powerapps.com"},
		{"central regions", "centralindia", "in-il101.gateway.prod.island", "athenawebservice.cin-il101.gateway.prod.island.powerapps.com"},
		{"compound central regions", "southcentralus", "us-il103.gateway.prod.island", "athenawebservice.scus-il103.gateway.prod.island.powerapps.com"},
		{"trailing central regions", "swedencentral", "se-il101.gateway.prod.island", "athenawebservice.cse-il101.gateway.prod.island.powerapps.com"},
		{"casing and padding are normalized", " AustraliaSoutheast ", "au-il301.gateway.prod.island", "athenawebservice.seau-il301.gateway.prod.island.powerapps.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host, err := athenaHost(test.azureRegion, test.clusterUriSuffix)
			require.NoError(t, err)
			assert.Equal(t, test.expectedHost, host)
		})
	}
}

func TestUnitAthenaHostRefusesToGuessThePrefix(t *testing.T) {
	// A wrong prefix resolves to nothing at all (NXDOMAIN), so an underivable one must be an error
	// rather than a hardcoded compass point.
	tests := []struct {
		name             string
		azureRegion      string
		clusterUriSuffix string
		expectedError    string
	}{
		{"no region", "", "au-il301.gateway.prod.island", "azureRegion is empty"},
		{"no cluster suffix", "australiaeast", "  ", "cluster uriSuffix is empty"},
		{"region without a compass direction", "usgovvirginia", "usgov-il101.gateway.prod.island", "no compass direction found"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host, err := athenaHost(test.azureRegion, test.clusterUriSuffix)
			require.Error(t, err)
			assert.Empty(t, host)
			assert.Contains(t, err.Error(), test.expectedError)
		})
	}
}

func TestUnitDeleteFabricLinkTargetsTheHostOfANonEastRegion(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(http.MethodGet, testBapUrl, func(_ *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(http.StatusOK, `{
			"properties": {
				"azureRegion": "australiasoutheast",
				"cluster": { "uriSuffix": "au-il301.gateway.prod.island" },
				"linkedEnvironmentMetadata": {
					"resourceId": "`+testOrganizationId+`",
					"instanceUrl": "`+testOrganizationUrl+`"
				}
			}
		}`), nil
	})

	deleteCalls := 0
	httpmock.RegisterResponder(http.MethodDelete,
		"https://athenawebservice.seau-il301.gateway.prod.island.powerapps.com/environment/"+testEnvironmentId+"/lakehouseArtifacts/"+testFolderId+"?dxt=false",
		func(_ *http.Request) (*http.Response, error) {
			deleteCalls++
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	err := newTestFabricLinkClient().DeleteFabricLink(context.Background(), testEnvironmentId, testFolderId)
	require.NoError(t, err, "an environment outside an east-something region must resolve to its own athena host")
	assert.Equal(t, 1, deleteCalls)
}

// stockDatalakeFolders is what an organization that has never been linked to anything already
// carries: cds2_workspace and cds3_workspace are stock rows, not artefacts of a link.
const stockDatalakeFolders = `{"value":[
	{"datalakefolderid":"00000000-0000-0000-0000-0000000000a1","datalakefolder_uniquename":"msdyn_analytics"},
	{"datalakefolderid":"00000000-0000-0000-0000-0000000000a2","datalakefolder_uniquename":"msdyn_processadvisor"},
	{"datalakefolderid":"00000000-0000-0000-0000-0000000000a3","datalakefolder_uniquename":"cds2_workspace"},
	{"datalakefolderid":"00000000-0000-0000-0000-0000000000a4","datalakefolder_uniquename":"cds3_workspace"}
]}`

func registerDatalakeReads(t *testing.T, profilesBody, foldersBody string) {
	t.Helper()
	httpmock.RegisterResponder(http.MethodGet, testProfilesUrl, func(_ *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(http.StatusOK, profilesBody), nil
	})
	httpmock.RegisterResponder(http.MethodGet, testFoldersUrl, func(_ *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(http.StatusOK, foldersBody), nil
	})
}

func TestUnitGetDatalakeFolderIdPicksTheFolderTheProfileReferences(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerDatalakeReads(t, `{"value":[
		{"synapselinkprofileid":"00000000-0000-0000-0000-0000000000b1","_datalakefolderid_value":"00000000-0000-0000-0000-0000000000a4"}
	]}`, stockDatalakeFolders)

	folderId, err := newTestFabricLinkClient().getDatalakeFolderId(context.Background(), testOrganizationUrl)
	require.NoError(t, err)
	assert.Equal(t, "00000000-0000-0000-0000-0000000000a4", folderId, "the unlink must target the folder a synapselinkprofile actually anchors to")
}

func TestUnitGetDatalakeFolderIdReadsAnUnprefixedLookupColumn(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerDatalakeReads(t, `{"value":[
		{"synapselinkprofileid":"00000000-0000-0000-0000-0000000000b1","datalakefolderid":"00000000-0000-0000-0000-0000000000a3"}
	]}`, stockDatalakeFolders)

	folderId, err := newTestFabricLinkClient().getDatalakeFolderId(context.Background(), testOrganizationUrl)
	require.NoError(t, err)
	assert.Equal(t, "00000000-0000-0000-0000-0000000000a3", folderId)
}

func TestUnitGetDatalakeFolderIdRefusesToPickAStockFolderOnAnUnlinkedOrg(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	// Zero synapselinkprofiles, ten-odd stock datalakefolders: the old positional fallback returned
	// value[0] here, which DELETE would then have removed.
	registerDatalakeReads(t, `{"value":[]}`, stockDatalakeFolders)

	folderId, err := newTestFabricLinkClient().getDatalakeFolderId(context.Background(), testOrganizationUrl)
	require.Error(t, err)
	assert.Empty(t, folderId, "an organization with no link must never yield a folder id to delete")
	assert.Contains(t, err.Error(), "no synapselinkprofile")
}

func TestUnitGetDatalakeFolderIdRefusesWhenTheReferencedFolderIsGone(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerDatalakeReads(t, `{"value":[
		{"synapselinkprofileid":"00000000-0000-0000-0000-0000000000b1","_datalakefolderid_value":"00000000-0000-0000-0000-0000000000ff"}
	]}`, stockDatalakeFolders)

	folderId, err := newTestFabricLinkClient().getDatalakeFolderId(context.Background(), testOrganizationUrl)
	require.Error(t, err)
	assert.Empty(t, folderId)
	assert.Contains(t, err.Error(), "refusing to guess")
}

func TestUnitGetDatalakeFolderIdPrefersTheFabricFolderWhenSeveralAreLinked(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerDatalakeReads(t, `{"value":[
		{"synapselinkprofileid":"00000000-0000-0000-0000-0000000000b1","_datalakefolderid_value":"00000000-0000-0000-0000-0000000000a1"},
		{"synapselinkprofileid":"00000000-0000-0000-0000-0000000000b2","_datalakefolderid_value":"00000000-0000-0000-0000-0000000000a4"}
	]}`, stockDatalakeFolders)

	folderId, err := newTestFabricLinkClient().getDatalakeFolderId(context.Background(), testOrganizationUrl)
	require.NoError(t, err)
	assert.Equal(t, "00000000-0000-0000-0000-0000000000a4", folderId, "cds3_workspace is the captured Link to Fabric anchor")
}

func TestUnitGetDatalakeFolderIdRefusesWhenSeveralNonFabricFoldersAreLinked(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerDatalakeReads(t, `{"value":[
		{"synapselinkprofileid":"00000000-0000-0000-0000-0000000000b1","_datalakefolderid_value":"00000000-0000-0000-0000-0000000000a1"},
		{"synapselinkprofileid":"00000000-0000-0000-0000-0000000000b2","_datalakefolderid_value":"00000000-0000-0000-0000-0000000000a2"}
	]}`, stockDatalakeFolders)

	folderId, err := newTestFabricLinkClient().getDatalakeFolderId(context.Background(), testOrganizationUrl)
	require.Error(t, err)
	assert.Empty(t, folderId)
	assert.Contains(t, err.Error(), "more than one linked datalakefolder")
}
