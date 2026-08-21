// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testApplicationId = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

func addAppUserUrl(environmentId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/addAppUser?api-version=2020-10-01", environmentId)
}

func TestUnitAddApplicationUser_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	httpmock.RegisterResponder(http.MethodPost, addAppUserUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.AddApplicationUser(context.Background(), testEnvironmentId, testApplicationId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.JSONEq(t, fmt.Sprintf(`{"servicePrincipalAppId":"%s"}`, testApplicationId), gotBody)
}

func TestUnitAddApplicationUser_UnexpectedStatusReturnsStatusError(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPost, addAppUserUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusBadRequest, `{"error":{"code":"InvalidServicePrincipal"}}`))

	err := client.AddApplicationUser(context.Background(), testEnvironmentId, testApplicationId)

	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
}
