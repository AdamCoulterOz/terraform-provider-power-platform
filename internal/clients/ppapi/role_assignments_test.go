// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
)

const (
	testEnvironmentGroupId = "30000000-0000-0000-0000-00000000000c"
	testRoleAssignmentId   = "40000000-0000-0000-0000-00000000000d"
)

// roleAssignmentsQuery pins the RBAC api-version every role assignment
// operation must send.
func roleAssignmentsQuery() url.Values {
	return url.Values{"api-version": []string{"2024-10-01"}}
}

func testRoleAssignmentRequest() RoleAssignmentRequestDto {
	return RoleAssignmentRequestDto{
		PrincipalObjectId: "50000000-0000-0000-0000-00000000000e",
		PrincipalType:     "ServicePrincipal",
		RoleDefinitionId:  "/providers/Microsoft.PowerPlatform/roleDefinitions/EnvironmentAdmin",
		Scope:             "/tenants/" + testTenantId,
	}
}

func TestUnitCreateRoleAssignment_Variants(t *testing.T) {
	request := testRoleAssignmentRequest()
	tests := []struct {
		name   string
		path   string
		status int
		call   func(client *Client) (*RoleAssignmentDto, error)
	}{
		{
			name:   "tenant level accepts 200",
			path:   "/authorization/roleAssignments",
			status: http.StatusOK,
			call: func(client *Client) (*RoleAssignmentDto, error) {
				return client.CreateRoleAssignment(context.Background(), request)
			},
		},
		{
			name:   "environment group accepts 201",
			path:   "/authorization/environmentGroups/" + testEnvironmentGroupId + "/roleAssignments",
			status: http.StatusCreated,
			call: func(client *Client) (*RoleAssignmentDto, error) {
				return client.CreateEnvironmentGroupRoleAssignment(context.Background(), testEnvironmentGroupId, request)
			},
		},
		{
			name:   "environment accepts 201",
			path:   "/authorization/environments/" + testEnvironmentId + "/roleAssignments",
			status: http.StatusCreated,
			call: func(client *Client) (*RoleAssignmentDto, error) {
				return client.CreateEnvironmentRoleAssignment(context.Background(), testEnvironmentId, request)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			attempts := 0
			var gotBody RoleAssignmentRequestDto
			httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com"+tc.path, roleAssignmentsQuery(),
				func(req *http.Request) (*http.Response, error) {
					attempts++
					if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
						return nil, err
					}
					return httpmock.NewJsonResponse(tc.status, RoleAssignmentDto{RoleAssignmentId: testRoleAssignmentId, Scope: request.Scope})
				})

			roleAssignment, err := tc.call(newTestClient(t))

			assert.NoError(t, err)
			assert.Equal(t, 1, attempts)
			assert.Equal(t, request, gotBody)
			assert.Equal(t, testRoleAssignmentId, roleAssignment.RoleAssignmentId)
		})
	}
}

func TestUnitCreateRoleAssignment_RetriesUntilScopePropagates(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Both propagation markers must be retried: the environment marker on the
	// first attempt and the environment group marker on the second.
	attempts := 0
	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/authorization/roleAssignments", roleAssignmentsQuery(),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			switch attempts {
			case 1:
				return httpmock.NewStringResponse(http.StatusBadRequest, `{"error":{"code":"EnvironmentIdInvalid"}}`), nil
			case 2:
				return httpmock.NewStringResponse(http.StatusBadRequest, `{"error":{"code":"EnvironmentGroupIdInvalid"}}`), nil
			default:
				return httpmock.NewJsonResponse(http.StatusOK, RoleAssignmentDto{RoleAssignmentId: testRoleAssignmentId})
			}
		})

	roleAssignment, err := newTestClient(t).CreateRoleAssignment(context.Background(), testRoleAssignmentRequest())

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "scope-propagation 400s must be retried until the scope becomes visible")
	assert.Equal(t, testRoleAssignmentId, roleAssignment.RoleAssignmentId)
}

func TestUnitCreateRoleAssignment_DoesNotRetryOtherBadRequests(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	attempts := 0
	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/authorization/roleAssignments", roleAssignmentsQuery(),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusBadRequest, `{"error":{"code":"PrincipalObjectIdInvalid"}}`), nil
		})

	roleAssignment, err := newTestClient(t).CreateRoleAssignment(context.Background(), testRoleAssignmentRequest())

	assert.Nil(t, roleAssignment)
	assert.Equal(t, 1, attempts, "a 400 without a scope-propagation marker must not be retried")
	var httpErr customerrors.UnexpectedHttpStatusCodeError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
}

func TestUnitCreateRoleAssignment_TransportErrorIsNotRetried(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// No responder is registered, so the request fails at the transport layer
	// with an error that is not an UnexpectedHttpStatusCodeError. The
	// scope-propagation predicate must treat it as non-retryable.
	roleAssignment, err := newTestClient(t).CreateRoleAssignment(context.Background(), testRoleAssignmentRequest())

	assert.Nil(t, roleAssignment)
	assert.Error(t, err)
	var httpErr customerrors.UnexpectedHttpStatusCodeError
	assert.False(t, errors.As(err, &httpErr))
}

func TestUnitListRoleAssignments_Variants(t *testing.T) {
	tests := []struct {
		name string
		path string
		call func(client *Client) ([]RoleAssignmentDto, error)
	}{
		{
			name: "tenant level",
			path: "/authorization/roleAssignments",
			call: func(client *Client) ([]RoleAssignmentDto, error) {
				return client.ListRoleAssignments(context.Background())
			},
		},
		{
			name: "environment group",
			path: "/authorization/environmentGroups/" + testEnvironmentGroupId + "/roleAssignments",
			call: func(client *Client) ([]RoleAssignmentDto, error) {
				return client.ListEnvironmentGroupRoleAssignments(context.Background(), testEnvironmentGroupId)
			},
		},
		{
			name: "environment",
			path: "/authorization/environments/" + testEnvironmentId + "/roleAssignments",
			call: func(client *Client) ([]RoleAssignmentDto, error) {
				return client.ListEnvironmentRoleAssignments(context.Background(), testEnvironmentId)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com"+tc.path, roleAssignmentsQuery(),
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(http.StatusOK, RoleAssignmentsListDto{Value: []RoleAssignmentDto{{RoleAssignmentId: testRoleAssignmentId}}})
				})

			roleAssignments, err := tc.call(newTestClient(t))

			assert.NoError(t, err)
			assert.Equal(t, []RoleAssignmentDto{{RoleAssignmentId: testRoleAssignmentId}}, roleAssignments)
		})
	}
}

func TestUnitDeleteRoleAssignment_Variants(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		status  int
		wantErr bool
		call    func(client *Client) error
	}{
		{
			name:   "tenant level accepts 200",
			path:   "/authorization/roleAssignments/" + testRoleAssignmentId,
			status: http.StatusOK,
			call: func(client *Client) error {
				return client.DeleteRoleAssignment(context.Background(), testRoleAssignmentId)
			},
		},
		{
			name:   "environment group accepts 204",
			path:   "/authorization/environmentGroups/" + testEnvironmentGroupId + "/roleAssignments/" + testRoleAssignmentId,
			status: http.StatusNoContent,
			call: func(client *Client) error {
				return client.DeleteEnvironmentGroupRoleAssignment(context.Background(), testEnvironmentGroupId, testRoleAssignmentId)
			},
		},
		{
			name:   "environment accepts 204",
			path:   "/authorization/environments/" + testEnvironmentId + "/roleAssignments/" + testRoleAssignmentId,
			status: http.StatusNoContent,
			call: func(client *Client) error {
				return client.DeleteEnvironmentRoleAssignment(context.Background(), testEnvironmentId, testRoleAssignmentId)
			},
		},
		{
			name:    "202 is not an accepted status",
			path:    "/authorization/roleAssignments/" + testRoleAssignmentId,
			status:  http.StatusAccepted,
			wantErr: true,
			call: func(client *Client) error {
				return client.DeleteRoleAssignment(context.Background(), testRoleAssignmentId)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponderWithQuery("DELETE", "https://api.powerplatform.com"+tc.path, roleAssignmentsQuery(),
				httpmock.NewStringResponder(tc.status, ""))

			err := tc.call(newTestClient(t))

			if tc.wantErr {
				var httpErr customerrors.UnexpectedHttpStatusCodeError
				assert.ErrorAs(t, err, &httpErr)
				assert.Equal(t, tc.status, httpErr.StatusCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnitListRoleDefinitions_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/authorization/roleDefinitions", roleAssignmentsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, RoleDefinitionsListDto{Value: []RoleDefinitionDto{
				{RoleDefinitionId: "rd-1", RoleDefinitionName: "Environment Admin", Permissions: []string{"*"}},
			}})
		})

	roleDefinitions, err := newTestClient(t).ListRoleDefinitions(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, []RoleDefinitionDto{{RoleDefinitionId: "rd-1", RoleDefinitionName: "Environment Admin", Permissions: []string{"*"}}}, roleDefinitions)
}
