// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetEnvironmentTemplatesByLocation lists the environment templates available
// in a location, grouped by category.
func (client *Client) GetEnvironmentTemplatesByLocation(ctx context.Context, location string) (EnvironmentTemplateDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/locations/%s/templates", location),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	templates := EnvironmentTemplateDto{}

	response, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, nil)
	if err != nil {
		return templates, fmt.Errorf("failed to execute API request for environment templates: %w", err)
	}

	defer response.HttpResponse.Body.Close()

	err = json.Unmarshal(response.BodyAsBytes, &templates)
	if err != nil {
		return templates, fmt.Errorf("failed to unmarshal environment templates response: %w", err)
	}

	return templates, nil
}
