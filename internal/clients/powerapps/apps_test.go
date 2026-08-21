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

// TestUnitGetEnvironmentPowerApps_RequestShape pins the admin apps listing:
// GET on the PowerApps host with the BAP api-version, and unwrapping of the
// value array.
func TestUnitGetEnvironmentPowerApps_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	expectedUrl := "https://api.powerapps.com/providers/Microsoft.PowerApps/scopes/admin/environments/00000000-0000-0000-0000-000000000001/apps?api-version=2023-06-01"
	gotUrl := ""
	httpmock.RegisterResponder(http.MethodGet, expectedUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, `{
				"value": [
					{
						"name": "11111111-2222-3333-4444-555555555555",
						"properties": {
							"displayName": "App One",
							"createdTime": "2023-09-27T07:08:47.1964785Z",
							"owner": {"displayName": "Owner", "id": "user-1", "userPrincipalName": "owner@contoso.com"},
							"environment": {
								"id": "/providers/Microsoft.PowerApps/environments/00000000-0000-0000-0000-000000000001",
								"location": "unitedstates",
								"name": "00000000-0000-0000-0000-000000000001"
							}
						}
					}
				]
			}`), nil
		})

	apps, err := client.GetEnvironmentPowerApps(context.Background(), "00000000-0000-0000-0000-000000000001")

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, gotUrl)
	require.Len(t, apps, 1)
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", apps[0].Name)
	assert.Equal(t, "App One", apps[0].Properties.DisplayName)
	assert.Equal(t, "2023-09-27T07:08:47.1964785Z", apps[0].Properties.CreatedTime)
	assert.Equal(t, "owner@contoso.com", apps[0].Properties.Owner.UserPrincipalName)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", apps[0].Properties.Environment.Name)
}

// TestUnitGetEnvironmentPowerApps_UnexpectedStatus pins that a status outside
// the accepted [200] surfaces the customerrors unexpected-status error,
// unwrapped, after exactly one attempt (404 is not retryable).
func TestUnitGetEnvironmentPowerApps_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://api.powerapps.com/providers/Microsoft.PowerApps/scopes/admin/environments/env-missing/apps",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusNotFound, "no such environment"), nil
		})

	apps, err := client.GetEnvironmentPowerApps(context.Background(), "env-missing")

	require.Error(t, err)
	assert.Nil(t, apps)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
	assert.Equal(t, statusErr.Error(), err.Error(), "the operation must not add context around the transport error")
	assert.Equal(t, 1, attempts, "a non-retryable status must not be replayed")
}

// TestUnitGetEnvironmentPowerApps_RetriesTransientStatus pins that the
// operation goes through Execute's transient-status retry (instant in
// TestMode).
func TestUnitGetEnvironmentPowerApps_RetriesTransientStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://api.powerapps.com/providers/Microsoft.PowerApps/scopes/admin/environments/env-1/apps",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusServiceUnavailable, "busy"), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, `{"value": []}`), nil
		})

	apps, err := client.GetEnvironmentPowerApps(context.Background(), "env-1")

	require.NoError(t, err)
	assert.Empty(t, apps)
	assert.Equal(t, 2, attempts, "a transient 503 must be retried")
}
