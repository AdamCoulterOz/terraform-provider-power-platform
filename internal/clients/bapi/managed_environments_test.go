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

func governanceConfigurationUrl(environmentId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/environments/%s/governanceConfiguration?api-version=2021-04-01", environmentId)
}

func TestUnitEnableManagedEnvironment_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	httpmock.RegisterResponder(http.MethodPost, governanceConfigurationUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	err := client.EnableManagedEnvironment(context.Background(), GovernanceConfigurationDto{ProtectionLevel: "Standard"}, testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.JSONEq(t, `{"protectionLevel":"Standard"}`, gotBody)
}

func TestUnitEnableManagedEnvironment_AcceptedWaitsForLifecycle(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/enable-op-1"
	registerLifecycleSucceeded(lifecycleUrl)
	httpmock.RegisterResponder(http.MethodPost, governanceConfigurationUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})

	err := client.EnableManagedEnvironment(context.Background(), GovernanceConfigurationDto{ProtectionLevel: "Standard"}, testEnvironmentId)

	assert.NoError(t, err)
}

func TestUnitEnableManagedEnvironment_ConflictOnAlreadyManagedIsSuccess(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	postAttempts := 0
	httpmock.RegisterResponder(http.MethodPost, governanceConfigurationUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			postAttempts++
			return httpmock.NewStringResponse(http.StatusConflict, `{"error":{"code":"RequestConflict"}}`), nil
		})
	registerGetEnvironmentResponder(testEnvironmentId,
		`{"displayName":"Test","governanceConfiguration":{"protectionLevel":"Standard"}}`)

	err := client.EnableManagedEnvironment(context.Background(), GovernanceConfigurationDto{ProtectionLevel: "Standard"}, testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, 1, postAttempts, "an already-managed environment must not trigger a retry")
}

func TestUnitEnableManagedEnvironment_ConflictOnUnmanagedEnvironmentRetries(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	postAttempts := 0
	httpmock.RegisterResponder(http.MethodPost, governanceConfigurationUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			postAttempts++
			if postAttempts == 1 {
				return httpmock.NewStringResponse(http.StatusConflict, `{"error":{"code":"RequestConflict"}}`), nil
			}
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})
	registerGetEnvironmentResponder(testEnvironmentId,
		`{"displayName":"Test","governanceConfiguration":{"protectionLevel":"Basic"}}`)

	err := client.EnableManagedEnvironment(context.Background(), GovernanceConfigurationDto{ProtectionLevel: "Standard"}, testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, 2, postAttempts, "a conflict on a genuinely unmanaged environment must be retried")
}

func TestUnitDisableManagedEnvironment_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/disable-op-1"
	registerLifecycleSucceeded(lifecycleUrl)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, governanceConfigurationUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})

	err := client.DisableManagedEnvironment(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.JSONEq(t, `{"protectionLevel":"Basic"}`, gotBody, "disabling must reset the governance configuration to basic protection")
}
