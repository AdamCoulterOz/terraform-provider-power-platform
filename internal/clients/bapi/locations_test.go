// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	locationsUrl           = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/locations?api-version=2023-06-01"
	europeCurrenciesUrl    = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/locations/europe/environmentCurrencies?api-version=2023-06-01"
	europeLanguagesUrl     = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/locations/europe/environmentLanguages?api-version=2023-06-01"
	locationsResponseBody  = `{"value":[{"id":"/providers/Microsoft.BusinessAppPlatform/locations/europe","name":"europe","properties":{"displayName":"Europe","code":"EU","isDefault":true,"azureRegions":["westeurope","northeurope"]}}]}`
	currenciesResponseBody = `{"value":[{"name":"USD","properties":{"code":"USD","symbol":"$","isTenantDefault":true}},{"name":"EUR","properties":{"code":"EUR"}}]}`
	languagesResponseBody  = `{"value":[{"name":"1033","properties":{"localeId":1033,"displayName":"English (United States)","isTenantDefault":true}}]}`
)

func TestUnitGetLocations_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotUrl string
	httpmock.RegisterResponder(http.MethodGet, locationsUrl,
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, locationsResponseBody), nil
		})

	locations, err := client.GetLocations(context.Background())

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, locationsUrl, gotUrl)
	require.Len(t, locations.Value, 1)
	assert.Equal(t, "europe", locations.Value[0].Name)
	assert.Equal(t, []string{"westeurope", "northeurope"}, locations.Value[0].Properties.AzureRegions)
	assert.True(t, locations.Value[0].Properties.IsDefault)
}

func TestUnitGetLocations_UnexpectedStatusReturnsStatusError(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, locationsUrl,
		httpmock.NewStringResponder(http.StatusForbidden, `{"error":{"code":"Forbidden"}}`))

	locations, err := client.GetLocations(context.Background())

	assert.Nil(t, locations)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusForbidden, statusErr.StatusCode)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
}

func TestUnitGetCurrenciesByLocation_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotUrl string
	httpmock.RegisterResponder(http.MethodGet, europeCurrenciesUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, currenciesResponseBody), nil
		})

	currencies, err := client.GetCurrenciesByLocation(context.Background(), "europe")

	require.NoError(t, err)
	assert.Equal(t, europeCurrenciesUrl, gotUrl)
	require.Len(t, currencies.Value, 2)
	assert.Equal(t, "USD", currencies.Value[0].Name)
	assert.Equal(t, "$", currencies.Value[0].Properties.Symbol)
}

func TestUnitGetLanguagesByLocation_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotUrl string
	httpmock.RegisterResponder(http.MethodGet, europeLanguagesUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, languagesResponseBody), nil
		})

	languages, err := client.GetLanguagesByLocation(context.Background(), "europe")

	require.NoError(t, err)
	assert.Equal(t, europeLanguagesUrl, gotUrl)
	require.Len(t, languages.Value, 1)
	assert.Equal(t, "1033", languages.Value[0].Name)
	assert.Equal(t, int64(1033), languages.Value[0].Properties.LocaleID)
}

func TestUnitGetLanguagesByLocation_EmptyBodyIsAnError(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, europeLanguagesUrl,
		httpmock.NewStringResponder(http.StatusOK, ""))

	_, err := client.GetLanguagesByLocation(context.Background(), "europe")

	assert.EqualError(t, err, "empty response body")
}
