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

const tenantResponseBody = `{"tenantId":"44444444-4444-4444-4444-444444444444","state":"Enabled","location":"europe","aadCountryGeo":"europe","dataStorageGeo":"europe","defaultEnvironmentGeo":"europe","aadDataBoundary":"none","fedRAMPHighCertificationRequired":false}`

func tenantUrl(apiVersion string) string {
	return "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/tenant?api-version=" + apiVersion
}

func TestUnitGetTenant_PinsDefaultApiVersion(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotUrl string
	httpmock.RegisterResponder(http.MethodGet, tenantUrl("2021-04-01"),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, tenantResponseBody), nil
		})

	tenant, err := client.GetTenant(context.Background())

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, tenantUrl("2021-04-01"), gotUrl, "GetTenant must call the tenant endpoint at api-version 2021-04-01")
	assert.Equal(t, "44444444-4444-4444-4444-444444444444", tenant.TenantId)
	assert.Equal(t, "Enabled", tenant.State)
	assert.Equal(t, "europe", tenant.DefaultEnvironmentGeo)
}

func TestUnitGetTenantAtVersion_UsesCallerSuppliedApiVersion(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotUrl string
	httpmock.RegisterResponder(http.MethodGet, tenantUrl("2020-08-01"),
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, tenantResponseBody), nil
		})

	tenant, err := client.GetTenantAtVersion(context.Background(), "2020-08-01")

	require.NoError(t, err)
	assert.Equal(t, tenantUrl("2020-08-01"), gotUrl)
	assert.Equal(t, "44444444-4444-4444-4444-444444444444", tenant.TenantId)
}

func TestUnitGetTenant_UnexpectedStatusIsWrapped(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, tenantUrl("2021-04-01"),
		httpmock.NewStringResponder(http.StatusForbidden, `{"error":{"code":"Forbidden"}}`))

	tenant, err := client.GetTenant(context.Background())

	assert.Nil(t, tenant)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to execute GET tenant API call")
}
