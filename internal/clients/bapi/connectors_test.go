// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitGetUnblockableConnectorsMetadata_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotRawQuery string
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.bap.microsoft.com/providers/PowerPlatform.Governance/v1/connectors/metadata/unblockable",
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotRawQuery = req.URL.RawQuery
			return httpmock.NewStringResponse(http.StatusOK,
				`[{"id":"/providers/Microsoft.PowerApps/apis/shared_teams","metadata":{"unblockable":true}}]`), nil
		})

	connectors, err := client.GetUnblockableConnectorsMetadata(context.Background())

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Empty(t, gotRawQuery, "the connector governance metadata endpoints are unversioned")
	require.Len(t, connectors, 1)
	assert.Equal(t, "/providers/Microsoft.PowerApps/apis/shared_teams", connectors[0].Id)
	assert.True(t, connectors[0].Metadata.Unblockable)
}

func TestUnitGetVirtualConnectorsMetadata_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotRawQuery string
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.bap.microsoft.com/providers/PowerPlatform.Governance/v1/connectors/metadata/virtual",
		func(req *http.Request) (*http.Response, error) {
			gotRawQuery = req.URL.RawQuery
			return httpmock.NewStringResponse(http.StatusOK,
				`[{"id":"/providers/Microsoft.PowerApps/apis/shared_cardsforpowerapps","metadata":{"virtualConnector":true,"name":"shared_cardsforpowerapps","type":"Microsoft.PowerApps/apis","displayName":"Cards"}}]`), nil
		})

	connectors, err := client.GetVirtualConnectorsMetadata(context.Background())

	require.NoError(t, err)
	assert.Empty(t, gotRawQuery)
	require.Len(t, connectors, 1)
	assert.True(t, connectors[0].Metadata.VirtualConnector)
	assert.Equal(t, "Cards", connectors[0].Metadata.DisplayName)
}

func TestUnitGetUnblockableConnectorsMetadata_UnexpectedStatusIsWrapped(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet,
		"https://api.bap.microsoft.com/providers/PowerPlatform.Governance/v1/connectors/metadata/unblockable",
		httpmock.NewStringResponder(http.StatusBadRequest, `{"error":{"code":"BadRequest"}}`))

	connectors, err := client.GetUnblockableConnectorsMetadata(context.Background())

	assert.Nil(t, connectors)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to fetch unblockable connectors metadata")
}
