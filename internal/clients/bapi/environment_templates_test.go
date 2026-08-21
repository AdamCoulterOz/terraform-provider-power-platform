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

const environmentTemplatesUrl = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/locations/europe/templates?api-version=2023-06-01"

func TestUnitGetEnvironmentTemplatesByLocation_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotUrl string
	httpmock.RegisterResponder(http.MethodGet, environmentTemplatesUrl,
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK,
				`{"standard":[{"name":"D365_Sales","properties":{"displayName":"Sales","isDisabled":false}}],"developer":[{"name":"D365_Dev","properties":{"displayName":"Developer","isDisabled":true,"disabledReason":{"code":"NoLicense","message":"missing license"}}}]}`), nil
		})

	templates, err := client.GetEnvironmentTemplatesByLocation(context.Background(), "europe")

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, environmentTemplatesUrl, gotUrl)
	require.Len(t, templates.Standard, 1)
	assert.Equal(t, "D365_Sales", templates.Standard[0].Name)
	require.Len(t, templates.Developer, 1)
	assert.True(t, templates.Developer[0].Properties.IsDisabled)
	assert.Equal(t, "NoLicense", templates.Developer[0].Properties.DisabledReason.Code)
	assert.Empty(t, templates.Premium)
}

func TestUnitGetEnvironmentTemplatesByLocation_UnexpectedStatusIsWrapped(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, environmentTemplatesUrl,
		httpmock.NewStringResponder(http.StatusBadRequest, `{"error":{"code":"InvalidLocation"}}`))

	_, err := client.GetEnvironmentTemplatesByLocation(context.Background(), "europe")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to execute API request for environment templates")
}
