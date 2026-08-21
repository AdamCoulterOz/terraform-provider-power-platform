// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// RemoveEnvironmentFromEnvironmentGroup removes an environment from an
// environment group on the tenant-scoped host.
func (client *Client) RemoveEnvironmentFromEnvironmentGroup(ctx context.Context, tenantId, environmentGroupId, environmentId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.tenantHost(tenantId),
		Path:   fmt.Sprintf("/environmentmanagement/environmentGroups/%s/removeEnvironment/%s", environmentGroupId, environmentId),
	}

	values := url.Values{}
	values.Add("api-version", "1")
	apiUrl.RawQuery = values.Encode()

	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, nil, []int{http.StatusAccepted}, nil)
	if err != nil {
		return fmt.Errorf("failed to remove environment from environment group: %w", err)
	}
	return nil
}
