// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetTableData reads rows of a Dataverse table, optionally narrowed by a raw
// OData query string, and unmarshals the response into responseObj.
func (client *Client) GetTableData(ctx context.Context, environmentHost, tableName, odataQuery string, responseObj any) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/v9.2/%s", tableName),
	}
	if odataQuery != "" {
		apiUrl.RawQuery = odataQuery
	}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &responseObj)
	if err != nil {
		return err
	} else if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	} else if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return err
	}
	return nil
}
