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

// GetConnectors lists the connectors available in an environment. An empty
// environment id lists the tenant-level connectors of the default environment.
func (client *Client) GetConnectors(ctx context.Context, environmentId string) ([]ConnectorDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/Microsoft.PowerApps/apis",
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.CONNECTORS_API_VERSION)
	values.Add("showApisWithToS", "true")
	values.Add("hideDlpExemptApis", "true")
	values.Add("showAllDlpEnforceableApis", "true")

	if environmentId == "" {
		environmentId = "~Default"
	}
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))

	apiUrl.RawQuery = values.Encode()

	connectorArray := ConnectorArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &connectorArray)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PowerApps connectors: %w", err)
	}

	return connectorArray.Value, nil
}
