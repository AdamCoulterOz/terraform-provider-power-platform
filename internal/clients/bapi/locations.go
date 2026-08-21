// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetLocations lists the geographic locations available to the tenant.
func (client *Client) GetLocations(ctx context.Context) (*LocationArrayDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/Microsoft.BusinessAppPlatform/locations",
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	locationsArray := LocationArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &locationsArray)
	if err != nil {
		return nil, err
	}
	return &locationsArray, nil
}

// GetCurrenciesByLocation lists the environment currencies available in a location.
func (client *Client) GetCurrenciesByLocation(ctx context.Context, location string) (CurrenciesArrayDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/locations/%s/environmentCurrencies", location),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	currencies := CurrenciesArrayDto{}

	response, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, nil)
	if err != nil {
		return currencies, err
	}

	defer response.HttpResponse.Body.Close()

	err = json.Unmarshal(response.BodyAsBytes, &currencies)
	if err != nil {
		return currencies, err
	}

	return currencies, nil
}

// GetLanguagesByLocation lists the environment languages available in a location.
func (client *Client) GetLanguagesByLocation(ctx context.Context, location string) (LanguagesArrayDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/locations/%s/environmentLanguages", location),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	languages := LanguagesArrayDto{}

	response, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, nil)
	if err != nil {
		return languages, err
	}

	defer response.HttpResponse.Body.Close()

	if len(response.BodyAsBytes) == 0 {
		return languages, errors.New("empty response body")
	}
	err = json.Unmarshal(response.BodyAsBytes, &languages)
	if err != nil {
		return languages, err
	}

	return languages, nil
}
