// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUnitGetTenantCapacity_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/tenantCapacity",
		url.Values{"api-version": []string{"2022-03-01-preview"}},
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, TenantCapacityDetailsDto{
				TenantId:         testTenantId,
				LicenseModelType: "StorageDriven",
				CapacitySummary:  CapacitySummaryDto{Status: "Ok"},
			})
		})

	capacity, err := newTestClient(t).GetTenantCapacity(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, testTenantId, capacity.TenantId)
	assert.Equal(t, "StorageDriven", capacity.LicenseModelType)
	assert.Equal(t, "Ok", capacity.CapacitySummary.Status)
}

// TestUnitGetTenantCapacity_RetriesTransientStatuses pins the shared Execute
// retry semantics for ppapi operations: a transient 503 is replayed (TestMode
// makes the backoff sleep a no-op) and the retried call succeeds.
func TestUnitGetTenantCapacity_RetriesTransientStatuses(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	attempts := 0
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/tenantCapacity",
		url.Values{"api-version": []string{"2022-03-01-preview"}},
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusServiceUnavailable, "busy"), nil
			}
			return httpmock.NewJsonResponse(http.StatusOK, TenantCapacityDetailsDto{TenantId: testTenantId})
		})

	capacity, err := newTestClient(t).GetTenantCapacity(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts, "transient statuses must be retried by Execute")
	assert.Equal(t, testTenantId, capacity.TenantId)
}

func TestUnitGetTenantCapacity_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	attempts := 0
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/tenantCapacity",
		url.Values{"api-version": []string{"2022-03-01-preview"}},
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusBadRequest, "bad request"), nil
		})

	capacity, err := newTestClient(t).GetTenantCapacity(context.Background())

	assert.Nil(t, capacity)
	assert.Equal(t, 1, attempts, "a non-transient status must not be retried")
	assert.ErrorContains(t, err, "failed to get tenant capacity details")
}
