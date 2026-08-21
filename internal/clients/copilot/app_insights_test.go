// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package copilot

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnitGetAppInsightsConfiguration_RequestShape pins the configuration
// read: GET on the caller-provided per-environment PowerVirtualAgents host
// with no query parameters, carrying the tenant id in the x-cci-tenantid
// header exactly as the service expects it (lower-case, non-canonical), and
// stamping the caller's environment and bot ids onto the decoded result.
func TestUnitGetAppInsightsConfiguration_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	expectedUrl := "https://powervamg.us-il101.gateway.prod.island.powerapps.com/api/botmanagement/2022-01-15/environments/00000000-0000-0000-0000-000000000001/bots/40000000-0000-0000-0000-000000000000/applicationinsightsconfiguration"
	gotUrl := ""
	var gotHeader map[string][]string
	httpmock.RegisterResponder(http.MethodGet, expectedUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			gotHeader = map[string][]string(req.Header.Clone())
			return httpmock.NewStringResponse(http.StatusOK, `{
				"environmentId": "server-reported-environment",
				"botId": "server-reported-bot",
				"appInsightsConnectionString": "InstrumentationKey=abc",
				"includeSensitiveInformation": true,
				"includeActivities": true,
				"includeActions": false,
				"errors": "",
				"networkIsolation": "PublicNetwork"
			}`), nil
		})

	configuration, err := client.GetAppInsightsConfiguration(
		context.Background(),
		"powervamg.us-il101.gateway.prod.island.powerapps.com",
		"10000000-0000-0000-0000-000000000000",
		"00000000-0000-0000-0000-000000000001",
		"40000000-0000-0000-0000-000000000000",
	)

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, gotUrl)
	// The header is sent under the literal lower-case key, so it must be read
	// back the same way: http.Header.Get would canonicalize the name and miss
	// it.
	assert.Equal(t, []string{"10000000-0000-0000-0000-000000000000"}, gotHeader["x-cci-tenantid"])
	require.NotNil(t, configuration)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", configuration.EnvironmentId, "the caller's environment id must overwrite the server-reported one")
	assert.Equal(t, "40000000-0000-0000-0000-000000000000", configuration.BotId, "the caller's bot id must overwrite the server-reported one")
	assert.Equal(t, "InstrumentationKey=abc", configuration.AppInsightsConnectionString)
	assert.True(t, configuration.IncludeSensitiveInformation)
	assert.True(t, configuration.IncludeActivities)
	assert.False(t, configuration.IncludeActions)
	assert.Equal(t, "PublicNetwork", configuration.NetworkIsolation)
}

// TestUnitGetAppInsightsConfiguration_UnexpectedStatus pins that a status
// outside the accepted [200] surfaces the customerrors unexpected-status
// error wrapped in the operation's context.
func TestUnitGetAppInsightsConfiguration_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "https://pva.example.com/api/botmanagement/2022-01-15/environments/env-1/bots/bot-1/applicationinsightsconfiguration",
		httpmock.NewStringResponder(http.StatusNotFound, "no such bot"))

	configuration, err := client.GetAppInsightsConfiguration(context.Background(), "pva.example.com", "tenant-1", "env-1", "bot-1")

	require.Error(t, err)
	assert.Nil(t, configuration)
	assert.ErrorContains(t, err, "failed to get Copilot Studio application insights configuration:")
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
}

// TestUnitGetAppInsightsConfiguration_RetriesTransientStatus pins that the
// operation goes through Execute's transient-status retry (instant in
// TestMode).
func TestUnitGetAppInsightsConfiguration_RetriesTransientStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://pva.example.com/api/botmanagement/2022-01-15/environments/env-1/bots/bot-1/applicationinsightsconfiguration",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusServiceUnavailable, "busy"), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, `{"errors": ""}`), nil
		})

	configuration, err := client.GetAppInsightsConfiguration(context.Background(), "pva.example.com", "tenant-1", "env-1", "bot-1")

	require.NoError(t, err)
	require.NotNil(t, configuration)
	assert.Equal(t, 2, attempts, "a transient 503 must be retried")
}

// TestUnitUpdateAppInsightsConfiguration_RequestShape pins the configuration
// update: PUT on the caller-provided per-environment PowerVirtualAgents host
// at the environment id carried inside the configuration, sending the full
// configuration as the JSON body and the tenant id in the x-cci-tenantid
// header, and stamping the caller's environment and bot ids onto the decoded
// result.
func TestUnitUpdateAppInsightsConfiguration_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	expectedUrl := "https://powervamg.us-il101.gateway.prod.island.powerapps.com/api/botmanagement/2022-01-15/environments/00000000-0000-0000-0000-000000000001/bots/40000000-0000-0000-0000-000000000000/applicationinsightsconfiguration"
	gotUrl := ""
	var gotHeader map[string][]string
	var gotBody []byte
	httpmock.RegisterResponder(http.MethodPut, expectedUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			gotHeader = map[string][]string(req.Header.Clone())
			body, readErr := io.ReadAll(req.Body)
			if readErr != nil {
				return nil, readErr
			}
			gotBody = body
			return httpmock.NewStringResponse(http.StatusOK, `{
				"environmentId": "server-reported-environment",
				"botId": "server-reported-bot",
				"appInsightsConnectionString": "InstrumentationKey=abc",
				"includeSensitiveInformation": false,
				"includeActivities": true,
				"includeActions": true,
				"errors": "",
				"networkIsolation": "PublicNetwork"
			}`), nil
		})

	updated, err := client.UpdateAppInsightsConfiguration(
		context.Background(),
		"powervamg.us-il101.gateway.prod.island.powerapps.com",
		"10000000-0000-0000-0000-000000000000",
		CopilotStudioAppInsightsDto{
			EnvironmentId:               "00000000-0000-0000-0000-000000000001",
			BotId:                       "40000000-0000-0000-0000-000000000000",
			AppInsightsConnectionString: "InstrumentationKey=abc",
			IncludeSensitiveInformation: false,
			IncludeActivities:           true,
			IncludeActions:              true,
			NetworkIsolation:            "PublicNetwork",
		},
		"40000000-0000-0000-0000-000000000000",
	)

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, gotUrl)
	assert.Equal(t, []string{"10000000-0000-0000-0000-000000000000"}, gotHeader["x-cci-tenantid"])
	assert.JSONEq(t, `{
		"environmentId": "00000000-0000-0000-0000-000000000001",
		"botId": "40000000-0000-0000-0000-000000000000",
		"appInsightsConnectionString": "InstrumentationKey=abc",
		"includeSensitiveInformation": false,
		"includeActivities": true,
		"includeActions": true,
		"errors": "",
		"networkIsolation": "PublicNetwork"
	}`, string(gotBody))
	require.NotNil(t, updated)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", updated.EnvironmentId, "the configuration's environment id must overwrite the server-reported one")
	assert.Equal(t, "40000000-0000-0000-0000-000000000000", updated.BotId, "the caller's bot id must overwrite the server-reported one")
	assert.Equal(t, "InstrumentationKey=abc", updated.AppInsightsConnectionString)
}

// TestUnitUpdateAppInsightsConfiguration_ServerReportedErrors pins the
// Errors-field check: a 200 response whose errors field is non-empty must be
// surfaced as an error carrying that text.
func TestUnitUpdateAppInsightsConfiguration_ServerReportedErrors(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodPut, "https://pva.example.com/api/botmanagement/2022-01-15/environments/env-1/bots/bot-1/applicationinsightsconfiguration",
		httpmock.NewStringResponder(http.StatusOK, `{"errors": "invalid connection string"}`))

	updated, err := client.UpdateAppInsightsConfiguration(context.Background(), "pva.example.com", "tenant-1",
		CopilotStudioAppInsightsDto{EnvironmentId: "env-1", BotId: "bot-1"}, "bot-1")

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, "error updating Application Insights configuration: invalid connection string", err.Error())
}

// TestUnitUpdateAppInsightsConfiguration_Accepts500AsTerminalError pins that
// 500 is an accepted status for the update: Execute must not retry it, and
// the operation must turn it into an error carrying the raw response body.
func TestUnitUpdateAppInsightsConfiguration_Accepts500AsTerminalError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodPut, "https://pva.example.com/api/botmanagement/2022-01-15/environments/env-1/bots/bot-1/applicationinsightsconfiguration",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusInternalServerError, `{"message": "bot update failed"}`), nil
		})

	updated, err := client.UpdateAppInsightsConfiguration(context.Background(), "pva.example.com", "tenant-1",
		CopilotStudioAppInsightsDto{EnvironmentId: "env-1", BotId: "bot-1"}, "bot-1")

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, `error updating Application Insights configuration: {"message": "bot update failed"}`, err.Error())
	assert.Equal(t, 1, attempts, "an accepted 500 must be terminal, not retried")
}

// TestUnitUpdateAppInsightsConfiguration_UnexpectedStatus pins that a status
// outside the accepted [200, 500] surfaces the customerrors
// unexpected-status error wrapped in the operation's context.
func TestUnitUpdateAppInsightsConfiguration_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodPut, "https://pva.example.com/api/botmanagement/2022-01-15/environments/env-1/bots/bot-1/applicationinsightsconfiguration",
		httpmock.NewStringResponder(http.StatusBadRequest, "malformed configuration"))

	updated, err := client.UpdateAppInsightsConfiguration(context.Background(), "pva.example.com", "tenant-1",
		CopilotStudioAppInsightsDto{EnvironmentId: "env-1", BotId: "bot-1"}, "bot-1")

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.ErrorContains(t, err, "failed to update Copilot Studio application insights configuration:")
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK, http.StatusInternalServerError}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
}
