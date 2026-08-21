// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
)

const (
	testConnectorName = "shared_sql"
	testConnectionId  = "10000000-0000-0000-0000-00000000000a"
)

// connectionsQuery is the query string every connector-scoped connections
// operation must send: api-version=1 plus the environment $filter.
func connectionsQuery() url.Values {
	return url.Values{
		"api-version": []string{"1"},
		"$filter":     []string{fmt.Sprintf("environment eq '%s'", testEnvironmentId)},
	}
}

func connectionUrl(pathSuffix string) string {
	return fmt.Sprintf("https://%s/connectivity/connectors/%s/connections/%s%s", testEnvironmentHost, testConnectorName, testConnectionId, pathSuffix)
}

func TestUnitGetConnections_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// The list endpoint is served from the environment host and sends only
	// api-version=1 (no $filter). Anything else fails with "no responder found".
	httpmock.RegisterResponderWithQuery("GET", "https://"+testEnvironmentHost+"/connectivity/connections",
		url.Values{"api-version": []string{"1"}},
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, ConnectionArrayDto{Value: []ConnectionDto{{Name: "conn-1"}, {Name: "conn-2"}}})
		})

	connections, err := newTestClient(t).GetConnections(context.Background(), testEnvironmentId)

	assert.NoError(t, err)
	assert.Equal(t, []ConnectionDto{{Name: "conn-1"}, {Name: "conn-2"}}, connections)
}

func TestUnitGetConnection_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", connectionUrl(""), connectionsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, ConnectionDto{Name: testConnectionId, Id: "id-1"})
		})

	connection, err := newTestClient(t).GetConnection(context.Background(), testEnvironmentId, testConnectorName, testConnectionId)

	assert.NoError(t, err)
	assert.Equal(t, testConnectionId, connection.Name)
	assert.Equal(t, "id-1", connection.Id)
}

func TestUnitGetConnection_TranslatesConnectionNotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", connectionUrl(""), connectionsQuery(),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"ConnectionNotFound"}}`))

	connection, err := newTestClient(t).GetConnection(context.Background(), testEnvironmentId, testConnectorName, testConnectionId)

	assert.Nil(t, connection)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, fmt.Sprintf("Connection '%s' not found", testConnectionId))
}

func TestUnitGetConnection_UnexpectedStatusWithoutMarkerIsNotTranslated(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", connectionUrl(""), connectionsQuery(),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"SomethingElse"}}`))

	connection, err := newTestClient(t).GetConnection(context.Background(), testEnvironmentId, testConnectorName, testConnectionId)

	assert.Nil(t, connection)
	assert.NotErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, "failed to get connection")
}

func TestUnitCreateConnection_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	attempts := 0
	var gotBody ConnectionCreateDto
	// The connection id path segment is a freshly generated uuid with the
	// dashes stripped, so the path is matched with a regexp.
	httpmock.RegisterResponder("PUT",
		`=~^https://000000000000000000000000000000\.01\.environment\.api\.powerplatform\.com/connectivity/connectors/shared_sql/connections/[0-9a-f]{32}\z`,
		func(req *http.Request) (*http.Response, error) {
			attempts++
			assert.Equal(t, "1", req.URL.Query().Get("api-version"))
			assert.Equal(t, fmt.Sprintf("environment eq '%s'", testEnvironmentId), req.URL.Query().Get("$filter"))
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusCreated, ConnectionDto{Name: "created-conn"})
		})

	connectionToCreate := ConnectionCreateDto{
		Properties: ConnectionCreatePropertiesDto{
			DisplayName:          "My Connection",
			ConnectionParameters: map[string]any{"server": "sql.example.test"},
			Environment:          ConnectionCreateEnvironmentDto{Id: "/providers/Microsoft.PowerApps/environments/" + testEnvironmentId, Name: testEnvironmentId},
		},
	}

	connection, err := newTestClient(t).CreateConnection(context.Background(), testEnvironmentId, testConnectorName, connectionToCreate)

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts, "a successful create must issue exactly one request")
	assert.Equal(t, "created-conn", connection.Name)
	assert.Equal(t, connectionToCreate, gotBody)
}

func TestUnitCreateConnection_RetriesWhileEnvironmentNotYetVisible(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	attempts := 0
	httpmock.RegisterResponder("PUT",
		`=~^https://000000000000000000000000000000\.01\.environment\.api\.powerplatform\.com/connectivity/connectors/shared_sql/connections/[0-9a-f]{32}\z`,
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts <= 2 {
				return httpmock.NewStringResponse(http.StatusNotFound, `{"error":{"code":"ServiceToServiceEnvironmentNotFound","message":"environment not visible yet"}}`), nil
			}
			return httpmock.NewJsonResponse(http.StatusCreated, ConnectionDto{Name: "created-conn"})
		})

	connection, err := newTestClient(t).CreateConnection(context.Background(), testEnvironmentId, testConnectorName, ConnectionCreateDto{})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "the 404 ServiceToServiceEnvironmentNotFound responses must be retried")
	assert.Equal(t, "created-conn", connection.Name)
}

func TestUnitCreateConnection_DoesNotRetryOtherErrors(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	attempts := 0
	httpmock.RegisterResponder("PUT",
		`=~^https://000000000000000000000000000000\.01\.environment\.api\.powerplatform\.com/connectivity/connectors/shared_sql/connections/[0-9a-f]{32}\z`,
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusNotFound, `{"error":{"code":"ConnectorNotFound"}}`), nil
		})

	connection, err := newTestClient(t).CreateConnection(context.Background(), testEnvironmentId, testConnectorName, ConnectionCreateDto{})

	assert.Nil(t, connection)
	assert.Equal(t, 1, attempts, "a 404 without the environment-visibility marker must not be retried")
	assert.ErrorContains(t, err, "failed to create connection")
	var httpErr customerrors.UnexpectedHttpStatusCodeError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
}

func TestUnitCreateConnection_TransportErrorIsNotRetried(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// No responder is registered, so the request fails at the transport layer
	// with an error that is not an UnexpectedHttpStatusCodeError. The
	// environment-visibility predicate must treat it as non-retryable.
	connection, err := newTestClient(t).CreateConnection(context.Background(), testEnvironmentId, testConnectorName, ConnectionCreateDto{})

	assert.Nil(t, connection)
	assert.ErrorContains(t, err, "failed to create connection")
	var httpErr customerrors.UnexpectedHttpStatusCodeError
	assert.False(t, errors.As(err, &httpErr))
}

func TestUnitUpdateConnection_ReadFailureShortCircuits(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Only the GET is registered; a PUT attempt would fail with a transport
	// error instead of the translated not-found error asserted below.
	httpmock.RegisterResponderWithQuery("GET", connectionUrl(""), connectionsQuery(),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"ConnectionNotFound"}}`))

	updated, err := newTestClient(t).UpdateConnection(context.Background(), testEnvironmentId, testConnectorName, testConnectionId, "new name", nil, nil)

	assert.Nil(t, updated)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
}

func TestUnitUpdateConnection_ReadsThenPutsModifiedConnection(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", connectionUrl(""), connectionsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, ConnectionDto{
				Name: testConnectionId,
				Properties: ConnectionPropertiesDto{
					ApiId:                "/providers/Microsoft.PowerApps/apis/shared_sql",
					DisplayName:          "old name",
					ConnectionParameters: map[string]any{"old": "param"},
				},
			})
		})

	var gotBody ConnectionDto
	httpmock.RegisterResponderWithQuery("PUT", connectionUrl(""), connectionsQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusOK, ConnectionDto{Name: testConnectionId, Properties: ConnectionPropertiesDto{DisplayName: "new name"}})
		})

	newParams := map[string]any{"server": "sql.example.test"}
	newParamsSet := map[string]any{"name": "oauth"}
	updated, err := newTestClient(t).UpdateConnection(context.Background(), testEnvironmentId, testConnectorName, testConnectionId, "new name", newParams, newParamsSet)

	assert.NoError(t, err)
	assert.Equal(t, "new name", updated.Properties.DisplayName)
	assert.Equal(t, "new name", gotBody.Properties.DisplayName)
	assert.Equal(t, newParams, gotBody.Properties.ConnectionParameters)
	assert.Equal(t, newParamsSet, gotBody.Properties.ConnectionParametersSet)
	assert.Equal(t, "/providers/Microsoft.PowerApps/apis/shared_sql", gotBody.Properties.ApiId, "unmodified fields must round-trip from the read")
}

func TestUnitDeleteConnection_Statuses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"200 is accepted", http.StatusOK, "{}", false},
		{"204 is not an accepted status", http.StatusNoContent, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponderWithQuery("DELETE", connectionUrl(""), connectionsQuery(),
				httpmock.NewStringResponder(tc.status, tc.body))

			err := newTestClient(t).DeleteConnection(context.Background(), testEnvironmentId, testConnectorName, testConnectionId)

			if tc.wantErr {
				assert.ErrorContains(t, err, "failed to delete connection")
				var httpErr customerrors.UnexpectedHttpStatusCodeError
				assert.ErrorAs(t, err, &httpErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnitShareConnection_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var gotBody ShareConnectionRequestDto
	httpmock.RegisterResponderWithQuery("POST", connectionUrl("/modifyPermissions"), connectionsQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewStringResponse(http.StatusOK, "{}"), nil
		})

	err := newTestClient(t).ShareConnection(context.Background(), testEnvironmentId, testConnectorName, testConnectionId, "CanEdit", "20000000-0000-0000-0000-00000000000b")

	assert.NoError(t, err)
	assert.Equal(t, ShareConnectionRequestDto{
		Put: []ShareConnectionRequestPutDto{
			{
				Properties: ShareConnectionRequestPutPropertiesDto{
					RoleName:     "CanEdit",
					Capabilities: []any{},
					Principal: ShareConnectionRequestPutPropertiesPrincipalDto{
						Id:       "20000000-0000-0000-0000-00000000000b",
						Type:     "ServicePrincipal",
						TenantId: nil,
					},
					NotifyShareTargetOption: "Notify",
				},
			},
		},
		Delete: []ShareConnectionRequestDeleteDto{},
	}, gotBody)
}

func TestUnitGetConnectionShares_SortsByPrincipalId(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", connectionUrl("/permissions"), connectionsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, ShareConnectionResponseArrayDto{Value: []ShareConnectionResponseDto{
				{Name: "share-b", Properties: ShareConnectionResponsePropertiesDto{Principal: map[string]any{"id": "bbb"}}},
				{Name: "share-a", Properties: ShareConnectionResponsePropertiesDto{Principal: map[string]any{"id": "aaa"}}},
				{Name: "share-c", Properties: ShareConnectionResponsePropertiesDto{Principal: map[string]any{"id": "ccc"}}},
			}})
		})

	shares, err := newTestClient(t).GetConnectionShares(context.Background(), testEnvironmentId, testConnectorName, testConnectionId)

	assert.NoError(t, err)
	names := make([]string, 0, len(shares.Value))
	for _, share := range shares.Value {
		names = append(names, share.Name)
	}
	assert.Equal(t, []string{"share-a", "share-b", "share-c"}, names)
}

func TestUnitGetConnectionShare_Behaviors(t *testing.T) {
	shares := ShareConnectionResponseArrayDto{Value: []ShareConnectionResponseDto{
		{Name: "share-1", Properties: ShareConnectionResponsePropertiesDto{Principal: map[string]any{"id": "principal-1", "displayName": "Principal One"}}},
		{Name: "share-2", Properties: ShareConnectionResponsePropertiesDto{Principal: map[string]any{"id": "principal-2", "displayName": ""}}},
	}}

	tests := []struct {
		name        string
		principalId string
		wantShare   string
		wantErrPart string
	}{
		{"found by principal id", "principal-1", "share-1", ""},
		{"unknown principal", "principal-9", "", "Share for principal 'principal-9' not found"},
		{"empty display name", "principal-2", "", "Display name is empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponderWithQuery("GET", connectionUrl("/permissions"), connectionsQuery(),
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(http.StatusOK, shares)
				})

			share, err := newTestClient(t).GetConnectionShare(context.Background(), testEnvironmentId, testConnectorName, testConnectionId, tc.principalId)

			if tc.wantErrPart != "" {
				assert.Nil(t, share)
				assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
				assert.ErrorContains(t, err, tc.wantErrPart)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantShare, share.Name)
			}
		})
	}
}

func TestUnitUpdateConnectionShare_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	share := ShareConnectionRequestDto{
		Put:    []ShareConnectionRequestPutDto{{Properties: ShareConnectionRequestPutPropertiesDto{RoleName: "CanView", Capabilities: []any{}, NotifyShareTargetOption: "Notify"}}},
		Delete: []ShareConnectionRequestDeleteDto{},
	}

	var gotBody ShareConnectionRequestDto
	httpmock.RegisterResponderWithQuery("POST", connectionUrl("/modifyPermissions"), connectionsQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewStringResponse(http.StatusOK, "{}"), nil
		})

	err := newTestClient(t).UpdateConnectionShare(context.Background(), testEnvironmentId, testConnectorName, testConnectionId, share)

	assert.NoError(t, err)
	assert.Equal(t, share, gotBody)
}

func TestUnitDeleteConnectionShare_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var gotBody ShareConnectionRequestDto
	httpmock.RegisterResponderWithQuery("POST", connectionUrl("/modifyPermissions"), connectionsQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewStringResponse(http.StatusOK, "{}"), nil
		})

	err := newTestClient(t).DeleteConnectionShare(context.Background(), testEnvironmentId, testConnectorName, testConnectionId, "share-9")

	assert.NoError(t, err)
	assert.Equal(t, ShareConnectionRequestDto{
		Put: []ShareConnectionRequestPutDto{},
		Delete: []ShareConnectionRequestDeleteDto{
			{Id: fmt.Sprintf("/providers/Microsoft.PowerApps/apis/%s/connections/%s/permissions/share-9", testConnectorName, testConnectionId)},
		},
	}, gotBody)
}
func TestUnitGetPrincipalString(t *testing.T) {
	principal := map[string]any{"id": "principal-1", "count": 42}

	id, err := GetPrincipalString(principal, "id")
	assert.NoError(t, err)
	assert.Equal(t, "principal-1", id)

	_, err = GetPrincipalString(principal, "missing")
	assert.ErrorContains(t, err, "principal key missing does not exist")

	_, err = GetPrincipalString(principal, "count")
	assert.ErrorContains(t, err, "failed to convert principal count to string")
}
