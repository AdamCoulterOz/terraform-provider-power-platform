// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerapps

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnitGetConnectors_RequestShape pins the connectors listing: GET on the
// PowerApps host with the connectors api-version, the three DLP visibility
// flags, and the environment $filter — defaulting to ~Default when no
// environment id is given.
func TestUnitGetConnectors_RequestShape(t *testing.T) {
	tests := []struct {
		name          string
		environmentId string
		expectedUrl   string
	}{
		{
			name:          "explicit environment id",
			environmentId: "00000000-0000-0000-0000-000000000001",
			expectedUrl:   "https://api.powerapps.com/providers/Microsoft.PowerApps/apis?%24filter=environment+eq+%2700000000-0000-0000-0000-000000000001%27&api-version=2019-05-01&hideDlpExemptApis=true&showAllDlpEnforceableApis=true&showApisWithToS=true",
		},
		{
			name:          "empty environment id falls back to ~Default",
			environmentId: "",
			expectedUrl:   "https://api.powerapps.com/providers/Microsoft.PowerApps/apis?%24filter=environment+eq+%27~Default%27&api-version=2019-05-01&hideDlpExemptApis=true&showAllDlpEnforceableApis=true&showApisWithToS=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			client := newTestClient(t)

			gotUrl := ""
			httpmock.RegisterResponder(http.MethodGet, tt.expectedUrl,
				func(req *http.Request) (*http.Response, error) {
					gotUrl = req.URL.String()
					return httpmock.NewStringResponse(http.StatusOK, `{
						"value": [
							{
								"name": "shared_sql",
								"id": "/providers/Microsoft.PowerApps/apis/shared_sql",
								"type": "Microsoft.PowerApps/apis",
								"properties": {
									"displayName": "SQL Server",
									"description": "Connect to SQL Server",
									"tier": "Premium",
									"publisher": "Microsoft"
								}
							}
						]
					}`), nil
				})

			connectors, err := client.GetConnectors(context.Background(), tt.environmentId)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedUrl, gotUrl)
			require.Len(t, connectors, 1)
			assert.Equal(t, "shared_sql", connectors[0].Name)
			assert.Equal(t, "/providers/Microsoft.PowerApps/apis/shared_sql", connectors[0].Id)
			assert.Equal(t, "Microsoft.PowerApps/apis", connectors[0].Type)
			assert.Equal(t, "SQL Server", connectors[0].Properties.DisplayName)
			assert.Equal(t, "Premium", connectors[0].Properties.Tier)
			assert.Equal(t, "Microsoft", connectors[0].Properties.Publisher)
		})
	}
}

// TestUnitGetConnectors_UnexpectedStatus pins that a status outside the
// accepted [200] surfaces the customerrors unexpected-status error wrapped in
// the operation's fetch-failure context.
func TestUnitGetConnectors_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://api.powerapps.com/providers/Microsoft.PowerApps/apis",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusForbidden, "no access"), nil
		})

	connectors, err := client.GetConnectors(context.Background(), "00000000-0000-0000-0000-000000000001")

	require.Error(t, err)
	assert.Nil(t, connectors)
	assert.ErrorContains(t, err, "failed to fetch PowerApps connectors:")
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusForbidden, statusErr.StatusCode)
	assert.Equal(t, 1, attempts, "a non-retryable status must not be replayed")
}

// TestUnitGetConnectors_RetriesTransientStatus pins that the operation goes
// through Execute's transient-status retry (instant in TestMode).
func TestUnitGetConnectors_RetriesTransientStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://api.powerapps.com/providers/Microsoft.PowerApps/apis",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusTooManyRequests, "slow down"), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, `{"value": []}`), nil
		})

	connectors, err := client.GetConnectors(context.Background(), "")

	require.NoError(t, err)
	assert.Empty(t, connectors)
	assert.Equal(t, 2, attempts, "a transient 429 must be retried")
}
