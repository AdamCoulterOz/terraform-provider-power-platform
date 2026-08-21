// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTenantId = "88888888-8888-8888-8888-888888888888"

func tenantIsolationPolicyUrl(tenantId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/PowerPlatform.Governance/v1/tenants/%s/tenantIsolationPolicy", tenantId)
}

func tenantIsolationPolicyBody(isDisabled bool) string {
	return fmt.Sprintf(`{"properties":{"tenantId":"%s","isDisabled":%t,"allowedTenants":[{"tenantId":"99999999-9999-9999-9999-999999999999","direction":{"inbound":true,"outbound":false}}]}}`, testTenantId, isDisabled)
}

func TestUnitGetTenantIsolationPolicy_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotRawQuery string
	httpmock.RegisterResponder(http.MethodGet, tenantIsolationPolicyUrl(testTenantId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotRawQuery = req.URL.RawQuery
			return httpmock.NewStringResponse(http.StatusOK, tenantIsolationPolicyBody(false)), nil
		})

	policy, err := client.GetTenantIsolationPolicy(context.Background(), testTenantId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Empty(t, gotRawQuery, "the tenant isolation governance endpoint is not versioned with an api-version query")
	assert.Equal(t, testTenantId, policy.Properties.TenantId)
	require.Len(t, policy.Properties.AllowedTenants, 1)
	assert.Equal(t, "99999999-9999-9999-9999-999999999999", policy.Properties.AllowedTenants[0].TenantId)
	require.NotNil(t, policy.Properties.AllowedTenants[0].Direction.Inbound)
	assert.True(t, *policy.Properties.AllowedTenants[0].Direction.Inbound)
}

func TestUnitGetTenantIsolationPolicy_NotFoundMeansNoPolicy(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, tenantIsolationPolicyUrl(testTenantId),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"PolicyNotFound"}}`))

	policy, err := client.GetTenantIsolationPolicy(context.Background(), testTenantId)

	assert.NoError(t, err, "a missing tenant isolation policy is not an error")
	assert.Nil(t, policy)
}

func newTenantIsolationPolicyRequest() TenantIsolationPolicyDto {
	return TenantIsolationPolicyDto{
		Properties: TenantIsolationPolicyPropertiesDto{
			TenantId:   testTenantId,
			IsDisabled: boolPtr(false),
			AllowedTenants: []AllowedTenantDto{
				{
					TenantId:  "99999999-9999-9999-9999-999999999999",
					Direction: DirectionDto{Inbound: boolPtr(true), Outbound: boolPtr(false)},
				},
			},
		},
	}
}

func TestUnitCreateOrUpdateTenantIsolationPolicy_SynchronousResult(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	httpmock.RegisterResponder(http.MethodPut, tenantIsolationPolicyUrl(testTenantId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, tenantIsolationPolicyBody(false)), nil
		})

	policy, err := client.CreateOrUpdateTenantIsolationPolicy(context.Background(), testTenantId, newTenantIsolationPolicyRequest())

	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.JSONEq(t, tenantIsolationPolicyBody(false), gotBody)
	assert.Equal(t, testTenantId, policy.Properties.TenantId)
}

func TestUnitCreateOrUpdateTenantIsolationPolicy_AsynchronousPollsAndRefetches(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	operationUrl := "https://api.bap.microsoft.com/operations/tenant-isolation-op-1"
	putAttempts := 0
	httpmock.RegisterResponder(http.MethodPut, tenantIsolationPolicyUrl(testTenantId),
		func(req *http.Request) (*http.Response, error) {
			putAttempts++
			return newResponseWithLocation(http.StatusAccepted, operationUrl), nil
		})
	pollAttempts := 0
	httpmock.RegisterResponder(http.MethodGet, operationUrl,
		func(req *http.Request) (*http.Response, error) {
			pollAttempts++
			if pollAttempts == 1 {
				resp := httpmock.NewStringResponse(http.StatusAccepted, "")
				resp.Header.Set("Retry-After", "30")
				return resp, nil
			}
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})
	httpmock.RegisterResponder(http.MethodGet, tenantIsolationPolicyUrl(testTenantId),
		httpmock.NewStringResponder(http.StatusOK, tenantIsolationPolicyBody(true)))

	policy, err := client.CreateOrUpdateTenantIsolationPolicy(context.Background(), testTenantId, newTenantIsolationPolicyRequest())

	require.NoError(t, err)
	assert.Equal(t, 1, putAttempts, "the PUT must not be replayed while the async operation runs")
	assert.Equal(t, 2, pollAttempts, "polling must continue until the operation reports 200")
	require.NotNil(t, policy)
	require.NotNil(t, policy.Properties.IsDisabled)
	assert.True(t, *policy.Properties.IsDisabled, "the fresh state read after the async operation must be returned")
}

func TestUnitCreateOrUpdateTenantIsolationPolicy_AsynchronousWithoutLocationFails(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPut, tenantIsolationPolicyUrl(testTenantId),
		httpmock.NewStringResponder(http.StatusAccepted, ""))

	policy, err := client.CreateOrUpdateTenantIsolationPolicy(context.Background(), testTenantId, newTenantIsolationPolicyRequest())

	assert.Nil(t, policy)
	require.Error(t, err)
	assert.ErrorContains(t, err, "no Location or Operation-Location header found in async response")
}
