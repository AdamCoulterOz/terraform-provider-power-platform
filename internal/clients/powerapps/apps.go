// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerapps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetEnvironmentPowerApps lists the Power Apps of a single environment.
func (client *Client) GetEnvironmentPowerApps(ctx context.Context, environmentName string) ([]PowerAppBapiDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.PowerApps/scopes/admin/environments/%s/apps", environmentName),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	appsArray := PowerAppArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &appsArray)
	if err != nil {
		return nil, err
	}
	return appsArray.Value, nil
}
