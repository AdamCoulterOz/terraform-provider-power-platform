// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"net/http"
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitGetTableData_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"name":"row1"}]}`})

	var response map[string]any
	err := client.GetTableData(context.Background(), testEnvironmentHost, "accounts", "$select=name&$top=2", &response)

	require.NoError(t, err)
	// The raw OData query string is passed through verbatim, unencoded.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts?$select=name&$top=2"},
	})
	require.Len(t, response["value"], 1)
}

func TestUnitGetTableData_RequestShape_NoQuery(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

	var response map[string]any
	err := client.GetTableData(context.Background(), testEnvironmentHost, "contacts", "", &response)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/contacts"},
	})
}

func TestUnitGetTableData_ErrorTranslation(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantErrPart string
	}{
		{
			name:        "403 is accepted then translated to access denied",
			status:      http.StatusForbidden,
			body:        `{"error":{"message":"missing privilege"}}`,
			wantErrPart: "access denied to resource at 'https://env.crm.dynamics.com/api/data/v9.2/accounts'",
		},
		{
			name:        "404 is accepted then translated to resource not found",
			status:      http.StatusNotFound,
			body:        "",
			wantErrPart: "resource not found at 'https://env.crm.dynamics.com/api/data/v9.2/accounts'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			recorded := activateRecorder(t, step{status: tt.status, body: tt.body})

			var response map[string]any
			err := client.GetTableData(context.Background(), testEnvironmentHost, "accounts", "", &response)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrPart)
			require.Len(t, *recorded, 1)
		})
	}
}

func TestUnitGetTableData_UnexpectedStatusCode(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusBadRequest, body: `{"error":"bad request"}`})

	var response map[string]any
	err := client.GetTableData(context.Background(), testEnvironmentHost, "accounts", "", &response)

	require.Error(t, err)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
	// 400 is not retryable, so exactly one attempt is made.
	require.Len(t, *recorded, 1)
}

// TestUnitGetTableData_RetriesTransientStatuses pins the retry contract of the
// Execute path every dataverse operation uses: transient statuses (503, 429)
// are replayed until an acceptable status arrives; TestMode makes the backoff
// sleeps instant.
func TestUnitGetTableData_RetriesTransientStatuses(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusServiceUnavailable, body: `{"error":"busy"}`},
		step{status: http.StatusTooManyRequests, body: `{"error":"throttled"}`},
		step{status: http.StatusOK, body: `{"value":[]}`},
	)

	var response map[string]any
	err := client.GetTableData(context.Background(), testEnvironmentHost, "accounts", "", &response)

	require.NoError(t, err)
	require.Len(t, *recorded, 3, "transient statuses must be retried until success")
	for _, req := range *recorded {
		assert.Equal(t, "GET", req.method)
		assert.Equal(t, "https://env.crm.dynamics.com/api/data/v9.2/accounts", req.url)
	}
}
