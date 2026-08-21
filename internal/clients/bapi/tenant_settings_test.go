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

const (
	listTenantSettingsUrl   = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/listTenantSettings?api-version=2023-06-01"
	updateTenantSettingsUrl = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/updateTenantSettings?api-version=2023-06-01"
)

func TestUnitGetTenantSettings_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	httpmock.RegisterResponder(http.MethodPost, listTenantSettingsUrl,
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, `{"walkMeOptOut":true,"disableNewsletterSendout":false}`), nil
		})

	settings, err := client.GetTenantSettings(context.Background())

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod, "reading tenant settings is a POST to listTenantSettings")
	assert.Empty(t, gotBody, "the list call must not send a request body")
	require.NotNil(t, settings.WalkMeOptOut)
	assert.True(t, *settings.WalkMeOptOut)
	require.NotNil(t, settings.DisableNewsletterSendout)
	assert.False(t, *settings.DisableNewsletterSendout)
}

func TestUnitUpdateTenantSettings_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, updateTenantSettingsUrl,
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, `{"walkMeOptOut":true,"disableSurveyFeedback":true}`), nil
		})

	walkMeOptOut := true
	settings, err := client.UpdateTenantSettings(context.Background(), TenantSettingsDto{WalkMeOptOut: &walkMeOptOut})

	require.NoError(t, err)
	assert.JSONEq(t, `{"walkMeOptOut":true}`, gotBody)
	require.NotNil(t, settings.DisableSurveyFeedback)
	assert.True(t, *settings.DisableSurveyFeedback, "the settings reported back by the API must be returned")
}
