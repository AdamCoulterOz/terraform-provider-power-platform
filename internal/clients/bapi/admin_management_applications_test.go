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

const testAdminApplicationId = "77777777-7777-7777-7777-777777777777"

func adminApplicationUrl(clientId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/adminApplications/%s?api-version=2020-10-01", clientId)
}

func TestUnitGetAdminApplication_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotUrl string
	httpmock.RegisterResponder(http.MethodGet, adminApplicationUrl(testAdminApplicationId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{"applicationId":"%s"}`, testAdminApplicationId)), nil
		})

	adminApp, err := client.GetAdminApplication(context.Background(), testAdminApplicationId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, adminApplicationUrl(testAdminApplicationId), gotUrl)
	assert.Equal(t, testAdminApplicationId, adminApp.ClientId)
}

func TestUnitGetAdminApplication_UnexpectedStatusIsWrapped(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, adminApplicationUrl(testAdminApplicationId),
		httpmock.NewStringResponder(http.StatusBadRequest, `{"error":{"code":"InvalidClientId"}}`))

	adminApp, err := client.GetAdminApplication(context.Background(), testAdminApplicationId)

	assert.Nil(t, adminApp)
	require.Error(t, err)
	assert.ErrorContains(t, err, fmt.Sprintf("failed to get admin app %s", testAdminApplicationId))
}

func TestUnitRegisterAdminApplication_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	httpmock.RegisterResponder(http.MethodPut, adminApplicationUrl(testAdminApplicationId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{"applicationId":"%s"}`, testAdminApplicationId)), nil
		})

	adminApp, err := client.RegisterAdminApplication(context.Background(), testAdminApplicationId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Empty(t, gotBody, "registration must not send a request body")
	assert.Equal(t, testAdminApplicationId, adminApp.ClientId)
}

func TestUnitUnregisterAdminApplication_AcceptsOkAndNoContent(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			httpmock.RegisterResponder(http.MethodDelete, adminApplicationUrl(testAdminApplicationId),
				httpmock.NewStringResponder(status, ""))

			err := client.UnregisterAdminApplication(context.Background(), testAdminApplicationId)

			assert.NoError(t, err)
		})
	}
}

func TestUnitUnregisterAdminApplication_UnexpectedStatusIsWrapped(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodDelete, adminApplicationUrl(testAdminApplicationId),
		httpmock.NewStringResponder(http.StatusForbidden, `{"error":{"code":"Forbidden"}}`))

	err := client.UnregisterAdminApplication(context.Background(), testAdminApplicationId)

	require.Error(t, err)
	assert.ErrorContains(t, err, fmt.Sprintf("failed to unregister admin app %s", testAdminApplicationId))
}
