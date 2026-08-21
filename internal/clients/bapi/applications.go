// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// AddApplicationUser registers a service principal as an application user of
// an environment through the BAPI admin addAppUser operation.
func (client *Client) AddApplicationUser(ctx context.Context, environmentId string, applicationId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/addAppUser", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.ADMIN_MANAGEMENT_APP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	// Create the request body
	requestBody := map[string]string{
		"servicePrincipalAppId": applicationId,
	}

	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, requestBody, []int{http.StatusOK}, nil)
	if err != nil {
		return err
	}

	return nil
}
