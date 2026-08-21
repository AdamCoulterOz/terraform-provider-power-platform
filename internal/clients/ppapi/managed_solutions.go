// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

// GetConnections lists the connections in an environment.
func (client *Client) GetConnections(ctx context.Context, environmentId string) ([]ConnectionDto, error) {
	apiURL := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   "/connectivity/connections",
	}
	values := url.Values{}
	values.Add("api-version", "1")
	apiURL.RawQuery = values.Encode()

	connections := ConnectionArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiURL.String(), nil, nil, []int{http.StatusOK}, &connections)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	return connections.Value, nil
}
