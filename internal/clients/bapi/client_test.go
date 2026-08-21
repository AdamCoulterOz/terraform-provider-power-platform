// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"io"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient returns a BAPI client wired to a TestMode transport: token
// acquisition returns a mock token and SleepWithContext is a no-op, so retry
// loops run instantly against httpmock responders.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	cfg := &config.ProviderConfig{
		TestMode: true,
		Urls: config.ProviderConfigUrls{
			BapiUrl:               "api.bap.microsoft.com",
			PowerAppsUrl:          "api.powerapps.com",
			PowerAppsScope:        "https://service.powerapps.com/.default",
			PowerPlatformUrl:      "api.powerplatform.com",
			PowerPlatformScope:    "https://api.powerplatform.com/.default",
			AdminPowerPlatformUrl: "api.admin.powerplatform.microsoft.com",
			PowerAppsAdvisor:      "api.advisor.powerapps.com",
			PowerAppsAdvisorScope: "https://advisor.powerapps.com/.default",
			AnalyticsScope:        "https://adminanalytics.powerplatform.microsoft.com/.default",
		},
	}
	return NewClient(api.NewApiClientBase(cfg, api.NewAuthBase(cfg)))
}

// activateHttpMocks activates httpmock on the default transport for the
// duration of the test.
func activateHttpMocks(t *testing.T) {
	t.Helper()
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)
}

// readBody drains the request body and returns it as a string, tolerating
// bodyless requests.
func readBody(t *testing.T, req *http.Request) string {
	t.Helper()
	if req.Body == nil {
		return ""
	}
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	return string(body)
}

// boolPtr returns a pointer to the given bool, for building DTO fixtures.
func boolPtr(value bool) *bool {
	return &value
}

// newResponseWithLocation builds a fresh response of the given status carrying
// a Location header, as BAPI returns when it starts a lifecycle operation.
func newResponseWithLocation(status int, location string) *http.Response {
	resp := httpmock.NewStringResponse(status, "")
	resp.Header.Set("Location", location)
	return resp
}

// registerLifecycleSucceeded registers a lifecycle-operation poll endpoint
// that immediately reports the operation as succeeded.
func registerLifecycleSucceeded(lifecycleUrl string) {
	httpmock.RegisterResponder(http.MethodGet, lifecycleUrl,
		httpmock.NewStringResponder(http.StatusOK, `{"state":{"id":"Succeeded"}}`))
}

func TestUnitClientScopes_IsPowerAppsScopeFromConfig(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, []string{"https://service.powerapps.com/.default"}, client.scopes())
}

func TestUnitClientHost_IsBapiUrlFromConfig(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, "api.bap.microsoft.com", client.host())
}

func TestUnitNewClient_WrapsSharedTransport(t *testing.T) {
	cfg := &config.ProviderConfig{TestMode: true}
	apiClient := api.NewApiClientBase(cfg, api.NewAuthBase(cfg))

	client := NewClient(apiClient)

	require.NotNil(t, client)
	assert.Same(t, apiClient, client.Api)
}
