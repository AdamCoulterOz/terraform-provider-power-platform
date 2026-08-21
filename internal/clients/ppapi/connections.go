// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

// CreateConnection creates a connection for a connector in an environment. It
// retries while the connectivity endpoint has not yet caught up with a freshly
// provisioned environment.
func (client *Client) CreateConnection(ctx context.Context, environmentId, connectorName string, connectionToCreate ConnectionCreateDto) (*ConnectionDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s", connectorName, strings.ReplaceAll(uuid.New().String(), "-", "")),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	connection := ConnectionDto{}
	deadline := time.Now().Add(constants.CONNECTIVITY_ENVIRONMENT_POLL_TIMEOUT)
	for {
		_, err := client.Api.Execute(ctx, client.scopes(), "PUT", apiUrl.String(), nil, connectionToCreate, []int{http.StatusCreated}, &connection)
		if err == nil {
			return &connection, nil
		}
		if !isEnvironmentNotYetVisible(err) || time.Now().After(deadline) {
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}

		tflog.Debug(ctx, fmt.Sprintf("Environment '%s' is not yet visible to the connectivity endpoint, retrying connection creation", environmentId))
		if sleepErr := client.Api.SleepWithContext(ctx, constants.CONNECTIVITY_ENVIRONMENT_POLL_INTERVAL); sleepErr != nil {
			return nil, sleepErr
		}
	}
}

// isEnvironmentNotYetVisible reports whether the connectivity endpoint has not caught up with a
// recently provisioned environment, which it signals with a 404 ServiceToServiceEnvironmentNotFound.
func isEnvironmentNotYetVisible(err error) bool {
	var httpErr customerrors.UnexpectedHttpStatusCodeError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == http.StatusNotFound && strings.Contains(string(httpErr.Body), "ServiceToServiceEnvironmentNotFound")
}

// UpdateConnection updates the display name and connection parameters of a
// connection.
func (client *Client) UpdateConnection(ctx context.Context, environmentId, connectorName, connectionId, displayName string, connParams, connParamsSet map[string]any) (*ConnectionDto, error) {
	conn, err := client.GetConnection(ctx, environmentId, connectorName, connectionId)
	if err != nil {
		return nil, err
	}

	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s", connectorName, connectionId),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	conn.Properties.DisplayName = displayName
	conn.Properties.ConnectionParametersSet = connParamsSet
	conn.Properties.ConnectionParameters = connParams

	updatedConnection := ConnectionDto{}
	_, err = client.Api.Execute(ctx, client.scopes(), "PUT", apiUrl.String(), nil, conn, []int{http.StatusOK}, &updatedConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to update connection: %w", err)
	}

	return &updatedConnection, nil
}

// GetConnection reads a single connection.
func (client *Client) GetConnection(ctx context.Context, environmentId, connectorName, connectionId string) (*ConnectionDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s", connectorName, connectionId),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	connection := ConnectionDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &connection)
	if err != nil {
		if strings.Contains(err.Error(), "ConnectionNotFound") {
			return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("Connection '%s' not found", connectionId))
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	return &connection, nil
}

// GetConnections lists the connections in an environment.
func (client *Client) GetConnections(ctx context.Context, environmentId string) ([]ConnectionDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   "/connectivity/connections",
	}

	values := url.Values{}
	values.Add("api-version", "1")
	apiUrl.RawQuery = values.Encode()

	connetionsArray := ConnectionArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &connetionsArray)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	return connetionsArray.Value, nil
}

// DeleteConnection deletes a connection.
func (client *Client) DeleteConnection(ctx context.Context, environmentId, connectorName, connectionId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s", connectorName, connectionId),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK}, nil)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}
	return nil
}

// ShareConnection grants a principal a role on a connection.
func (client *Client) ShareConnection(ctx context.Context, environmentId, connectorName, connectionId, roleName, entraUserObjectId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s/modifyPermissions", connectorName, connectionId),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	share := ShareConnectionRequestDto{
		Put: []ShareConnectionRequestPutDto{
			{
				Properties: ShareConnectionRequestPutPropertiesDto{
					RoleName:     roleName,
					Capabilities: []any{},
					Principal: ShareConnectionRequestPutPropertiesPrincipalDto{
						Id:       entraUserObjectId,
						Type:     "ServicePrincipal",
						TenantId: nil,
					},
					NotifyShareTargetOption: "Notify",
				},
			},
		},
		Delete: []ShareConnectionRequestDeleteDto{},
	}

	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, share, []int{http.StatusOK}, nil)
	if err != nil {
		return fmt.Errorf("failed to share connection: %w", err)
	}
	return nil
}

// GetPrincipalString reads a string attribute from a connection share
// principal, which the API returns as an untyped map.
func GetPrincipalString(principal map[string]any, key string) (string, error) {
	raw, exists := principal[key]
	if !exists {
		return "", fmt.Errorf("principal key %s does not exist", key)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("failed to convert principal %s to string", key)
	}
	return value, nil
}

// GetConnectionShares lists the shares of a connection, ordered by principal id.
func (client *Client) GetConnectionShares(ctx context.Context, environmentId, connectorName, connectionId string) (*ShareConnectionResponseArrayDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s/permissions", connectorName, connectionId),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	share := ShareConnectionResponseArrayDto{}

	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &share)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection shares: %w", err)
	}

	sort.SliceStable(share.Value, func(i, j int) bool {
		idI, errI := GetPrincipalString(share.Value[i].Properties.Principal, "id")
		idJ, errJ := GetPrincipalString(share.Value[j].Properties.Principal, "id")
		if errI != nil || errJ != nil {
			return false
		}
		return idI < idJ
	})

	return &share, nil
}

// GetConnectionShare finds the share of a connection held by a principal.
func (client *Client) GetConnectionShare(ctx context.Context, environmentId, connectorName, connectionId, principalId string) (*ShareConnectionResponseDto, error) {
	shares, err := client.GetConnectionShares(ctx, environmentId, connectorName, connectionId)
	if err != nil {
		return nil, err
	}

	for _, share := range shares.Value {
		id, err := GetPrincipalString(share.Properties.Principal, "id")
		if err != nil {
			return nil, err
		}

		if id == principalId {
			displayName, err := GetPrincipalString(share.Properties.Principal, "displayName")
			if err != nil || displayName == "" {
				return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND),
					fmt.Sprintf("Share for principal '%s' not found. Display name is empty", principalId))
			}
			return &share, nil
		}
	}
	return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND),
		fmt.Sprintf("Share for principal '%s' not found", principalId))
}

// UpdateConnectionShare applies a share modification request to a connection.
func (client *Client) UpdateConnectionShare(ctx context.Context, environmentId, connectorName, connectionId string, share ShareConnectionRequestDto) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s/modifyPermissions", connectorName, connectionId),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, share, []int{http.StatusOK}, nil)
	if err != nil {
		return fmt.Errorf("failed to update connection share: %w", err)
	}
	return nil
}

// DeleteConnectionShare removes a share from a connection.
func (client *Client) DeleteConnectionShare(ctx context.Context, environmentId, connectorName, connectionId, shareId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   helpers.BuildEnvironmentHostUri(environmentId, client.host()),
		Path:   fmt.Sprintf("/connectivity/connectors/%s/connections/%s/modifyPermissions", connectorName, connectionId),
	}
	values := url.Values{}
	values.Add("api-version", "1")
	values.Add("$filter", fmt.Sprintf("environment eq '%s'", environmentId))
	apiUrl.RawQuery = values.Encode()

	share := ShareConnectionRequestDto{
		Put: []ShareConnectionRequestPutDto{},
		Delete: []ShareConnectionRequestDeleteDto{
			{
				Id: fmt.Sprintf("/providers/Microsoft.PowerApps/apis/%s/connections/%s/permissions/%s", connectorName, connectionId, shareId),
			},
		},
	}

	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, share, []int{http.StatusOK}, nil)
	if err != nil {
		return fmt.Errorf("failed to delete connection share: %w", err)
	}
	return nil
}
