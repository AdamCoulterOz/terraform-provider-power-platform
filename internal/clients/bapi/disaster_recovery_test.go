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

func TestUnitEnableAndDisableDisasterRecovery_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	tests := []struct {
		name          string
		expectedState string
		call          func(ctx context.Context, environmentId string) error
	}{
		{name: "enable", expectedState: "Enabled", call: client.EnableDisasterRecovery},
		{name: "disable", expectedState: "Disabled", call: client.DisableDisasterRecovery},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/dr-op-" + tc.expectedState
			registerLifecycleSucceeded(lifecycleUrl)

			var gotMethod, gotBody string
			httpmock.RegisterResponder(http.MethodPatch,
				fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?api-version=2021-04-01", testEnvironmentId),
				func(req *http.Request) (*http.Response, error) {
					gotMethod = req.Method
					gotBody = readBody(t, req)
					return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
				})

			err := tc.call(context.Background(), testEnvironmentId)

			require.NoError(t, err)
			assert.Equal(t, http.MethodPatch, gotMethod)
			assert.JSONEq(t, fmt.Sprintf(`{"properties":{"states":{"disasterRecovery":{"id":"%s"}}}}`, tc.expectedState), gotBody)
		})
	}
}
