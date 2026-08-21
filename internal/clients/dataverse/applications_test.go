// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testApplicationId  = "aaaabbbb-0000-0000-0000-000000000001"
	testEnvironmentId  = "11112222-0000-0000-0000-000000000001"
	testBusinessUnitId = "cccccccc-0000-0000-0000-000000000001"
	testPrincipalId    = "ddddeeee-0000-0000-0000-000000000001"

	// principalExpandQuery and principalSelectQuery are the encoded query
	// parts the application user reads send (url.Values sorts $expand before
	// $filter and $select).
	principalExpandQuery = "%24expand=systemuserroles_association%28%24select%3Droleid%2Cname%2C_businessunitid_value%29"
	principalSelectQuery = "%24select=applicationid%2Csystemuserid%2Cfullname%2Cisdisabled%2Cdeletedstate%2C_businessunitid_value"
)

func applicationUserBody(systemUserId string, deletedState int, roles string) string {
	return `{
		"fullname": "Application User",
		"applicationid": "` + testApplicationId + `",
		"systemuserid": "` + systemUserId + `",
		"_businessunitid_value": "` + testBusinessUnitId + `",
		"isdisabled": false,
		"deletedstate": ` + strconv.Itoa(deletedState) + `,
		"systemuserroles_association": [` + roles + `]
	}`
}

func TestUnitGetRootBusinessUnitId_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"businessunitid":"` + testBusinessUnitId + `","name":"org"}]}`})

	businessUnitId, err := client.GetRootBusinessUnitId(context.Background(), testEnvironmentHost, testEnvironmentId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/businessunits?%24filter=parentbusinessunitid+eq+null&%24select=businessunitid%2Cname"},
	})
	assert.Equal(t, testBusinessUnitId, businessUnitId)
}

func TestUnitGetRootBusinessUnitId_ResultCardinality(t *testing.T) {
	t.Run("no root business unit is OBJECT_NOT_FOUND", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

		_, err := client.GetRootBusinessUnitId(context.Background(), testEnvironmentHost, testEnvironmentId)

		require.Error(t, err)
		assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
		assert.Contains(t, err.Error(), "root business unit not found in environment '"+testEnvironmentId+"'")
	})
	t.Run("multiple root business units fail", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"businessunitid":"a"},{"businessunitid":"b"}]}`})

		_, err := client.GetRootBusinessUnitId(context.Background(), testEnvironmentHost, testEnvironmentId)

		require.Error(t, err)
		assert.EqualError(t, err, "expected exactly one root business unit in environment '"+testEnvironmentId+"', got 2")
	})
}

func TestUnitGetApplicationUsersByApplicationId_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + applicationUserBody(testPrincipalId, 0, "") + `]}`})

	users, err := client.GetApplicationUsersByApplicationId(context.Background(), testEnvironmentHost, testEnvironmentId, testApplicationId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers?" + principalExpandQuery + "&%24filter=applicationid+eq+" + testApplicationId + "&" + principalSelectQuery},
	})
	require.Len(t, users, 1)
	assert.Equal(t, testPrincipalId, users[0].SystemUserId)
}

func TestUnitGetApplicationUsersByApplicationId_EmptyResultIsProviderError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

	_, err := client.GetApplicationUsersByApplicationId(context.Background(), testEnvironmentHost, testEnvironmentId, testApplicationId)

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
	assert.Contains(t, err.Error(), "application user '"+testApplicationId+"' not found in environment '"+testEnvironmentId+"'")
}

func TestUnitGetApplicationUser_FiltersDeletedAndSortsRoles(t *testing.T) {
	client := newTestClient(t)
	roles := `{"roleid":"bbbbbbbb-0000-0000-0000-000000000002","name":"Second"},{"roleid":"aaaaaaaa-0000-0000-0000-000000000001","name":"First"}`
	activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` +
		applicationUserBody("99990000-0000-0000-0000-000000000009", 2, "") + `,` +
		applicationUserBody(testPrincipalId, 0, roles) + `]}`})

	user, err := client.GetApplicationUser(context.Background(), testEnvironmentHost, testEnvironmentId, testApplicationId)

	require.NoError(t, err)
	assert.Equal(t, testPrincipalId, user.SystemUserId, "soft-deleted rows must be ignored")
	require.Len(t, user.SecurityRoles, 2)
	assert.Equal(t, "First", user.SecurityRoles[0].Name, "roles must be sorted by role id")
}

func TestUnitGetApplicationUser_MultipleActiveUsersFail(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` +
		applicationUserBody("99990000-0000-0000-0000-000000000009", 0, "") + `,` +
		applicationUserBody(testPrincipalId, 0, "") + `]}`})

	_, err := client.GetApplicationUser(context.Background(), testEnvironmentHost, testEnvironmentId, testApplicationId)

	require.Error(t, err)
	assert.EqualError(t, err, "multiple active application users found for application '"+testApplicationId+"' in environment '"+testEnvironmentId+"': 99990000-0000-0000-0000-000000000009, "+testPrincipalId)
}

func TestUnitCreateApplicationUser_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{
		status: http.StatusNoContent,
		header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/systemusers(" + testPrincipalId + ")"),
	})

	systemUserId, err := client.CreateApplicationUser(context.Background(), testEnvironmentHost, testApplicationId, testBusinessUnitId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{
			method: "POST",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/systemusers",
			body:   `{"accessmode":4,"applicationid":"` + testApplicationId + `","businessunitid@odata.bind":"/businessunits(` + testBusinessUnitId + `)","isdisabled":false}`,
		},
	})
	assert.Equal(t, testPrincipalId, systemUserId)
}

func TestUnitCreateApplicationUser_MissingEntityIdHeaderFails(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusNoContent})

	_, err := client.CreateApplicationUser(context.Background(), testEnvironmentHost, testApplicationId, testBusinessUnitId)

	require.Error(t, err)
	assert.EqualError(t, err, "no entity record id returned from the odata-entityid header")
}

func TestUnitSetApplicationUserDisabledState_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusNoContent})

	err := client.SetApplicationUserDisabledState(context.Background(), testEnvironmentHost, testPrincipalId, testApplicationId, true)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{
			method: "PATCH",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testPrincipalId + "%29",
			body:   `{"applicationid":"` + testApplicationId + `","isdisabled":true}`,
		},
	})
}

func TestUnitGetPrincipalBySystemUserId_RequestShape(t *testing.T) {
	client := newTestClient(t)
	roles := `{"roleid":"bbbbbbbb-0000-0000-0000-000000000002","name":"Second"},{"roleid":"aaaaaaaa-0000-0000-0000-000000000001","name":"First"}`
	recorded := activateRecorder(t, step{status: http.StatusOK, body: applicationUserBody(testPrincipalId, 0, roles)})

	principal, err := client.GetPrincipalBySystemUserId(context.Background(), testEnvironmentHost, testPrincipalId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testPrincipalId + "%29?" + principalExpandQuery + "&" + principalSelectQuery},
	})
	assert.Equal(t, testPrincipalId, principal.SystemUserId)
	require.Len(t, principal.SecurityRoles, 2)
	assert.Equal(t, "First", principal.SecurityRoles[0].Name, "roles must be sorted by role id")
}

func TestUnitGetPrincipalBySystemUserId_DeletedIsProviderError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: applicationUserBody(testPrincipalId, 2, "")})

	_, err := client.GetPrincipalBySystemUserId(context.Background(), testEnvironmentHost, testPrincipalId)

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
	assert.Contains(t, err.Error(), "principal not found for system user ID "+testPrincipalId)
}

func TestUnitGetDataverseSecurityRoles_RequestShape(t *testing.T) {
	tests := []struct {
		name           string
		businessUnitId string
		wantURL        string
	}{
		{
			name:           "unfiltered",
			businessUnitId: "",
			wantURL:        "https://env.crm.dynamics.com/api/data/v9.2/roles?%24select=roleid%2Cname%2C_businessunitid_value",
		},
		{
			name:           "filtered by business unit",
			businessUnitId: testBusinessUnitId,
			wantURL:        "https://env.crm.dynamics.com/api/data/v9.2/roles?%24filter=_businessunitid_value+eq+" + testBusinessUnitId + "&%24select=roleid%2Cname%2C_businessunitid_value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"roleid":"aaaaaaaa-0000-0000-0000-000000000001","name":"Basic User","_businessunitid_value":"` + testBusinessUnitId + `"}]}`})

			roles, err := client.GetDataverseSecurityRoles(context.Background(), testEnvironmentHost, tt.businessUnitId)

			require.NoError(t, err)
			assertRequests(t, recorded, []recordedRequest{
				{method: "GET", url: tt.wantURL},
			})
			require.Len(t, roles, 1)
		})
	}
}

func TestUnitResolveSecurityRoleNames(t *testing.T) {
	rolesBody := `{"value":[
		{"roleid":"bbbbbbbb-0000-0000-0000-000000000002","name":"System Administrator","_businessunitid_value":"` + testBusinessUnitId + `"},
		{"roleid":"aaaaaaaa-0000-0000-0000-000000000001","name":"Basic User","_businessunitid_value":"` + testBusinessUnitId + `"},
		{"roleid":"cccccccc-0000-0000-0000-000000000003","name":"Duplicated","_businessunitid_value":"` + testBusinessUnitId + `"},
		{"roleid":"dddddddd-0000-0000-0000-000000000004","name":"Duplicated","_businessunitid_value":"` + testBusinessUnitId + `"}
	]}`

	t.Run("resolves and sorts by role id", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusOK, body: rolesBody})

		resolved, err := client.ResolveSecurityRoleNames(context.Background(), testEnvironmentHost, testBusinessUnitId, []string{"System Administrator", "Basic User"})

		require.NoError(t, err)
		require.Len(t, resolved, 2)
		assert.Equal(t, "Basic User", resolved[0].Name)
		assert.Equal(t, "System Administrator", resolved[1].Name)
	})
	t.Run("missing role is OBJECT_NOT_FOUND", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusOK, body: rolesBody})

		_, err := client.ResolveSecurityRoleNames(context.Background(), testEnvironmentHost, testBusinessUnitId, []string{"Ghost"})

		require.Error(t, err)
		assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
		assert.Contains(t, err.Error(), "security role 'Ghost' not found in business unit '"+testBusinessUnitId+"'")
	})
	t.Run("ambiguous role fails", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusOK, body: rolesBody})

		_, err := client.ResolveSecurityRoleNames(context.Background(), testEnvironmentHost, testBusinessUnitId, []string{"Duplicated"})

		require.Error(t, err)
		assert.EqualError(t, err, "security role 'Duplicated' is ambiguous in business unit '"+testBusinessUnitId+"'")
	})
}

func TestUnitAddPrincipalSecurityRoles_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusNoContent},
		step{status: http.StatusNoContent},
	)

	roleIds := []string{"aaaaaaaa-0000-0000-0000-000000000001", "bbbbbbbb-0000-0000-0000-000000000002"}
	err := client.AddPrincipalSecurityRoles(context.Background(), testEnvironmentHost, testPrincipalId, roleIds)

	require.NoError(t, err)
	refURL := "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testPrincipalId + "%29/systemuserroles_association/$ref"
	assertRequests(t, recorded, []recordedRequest{
		{method: "POST", url: refURL, body: `{"@odata.id":"https://env.crm.dynamics.com/api/data/v9.2/roles(` + roleIds[0] + `)"}`},
		{method: "POST", url: refURL, body: `{"@odata.id":"https://env.crm.dynamics.com/api/data/v9.2/roles(` + roleIds[1] + `)"}`},
	})
}

func TestUnitRemovePrincipalSecurityRoles_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusNoContent})

	roleId := "aaaaaaaa-0000-0000-0000-000000000001"
	err := client.RemovePrincipalSecurityRoles(context.Background(), testEnvironmentHost, testPrincipalId, []string{roleId})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{
			method: "DELETE",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testPrincipalId + "%29/systemuserroles_association/$ref?%24id=https%3A%2F%2Fenv.crm.dynamics.com%2Fapi%2Fdata%2Fv9.2%2Froles%28" + roleId + "%29",
		},
	})
}

// TestUnitDeleteSystemUser_CallerChosenStatusCodes pins the contract that the
// caller supplies the acceptable statuses, so a second-pass permanent delete
// can tolerate 404 while a first-pass delete treats it as unexpected.
func TestUnitDeleteSystemUser_CallerChosenStatusCodes(t *testing.T) {
	t.Run("404 outside the accepted list is an error", func(t *testing.T) {
		client := newTestClient(t)
		recorded := activateRecorder(t, step{status: http.StatusNotFound, body: `{}`})

		err := client.DeleteSystemUser(context.Background(), testEnvironmentHost, testPrincipalId, []int{http.StatusNoContent})

		require.Error(t, err)
		var statusErr customerrors.UnexpectedHttpStatusCodeError
		require.ErrorAs(t, err, &statusErr)
		assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
		assertRequests(t, recorded, []recordedRequest{
			{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testPrincipalId + "%29"},
		})
	})
	t.Run("404 inside the accepted list succeeds", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusNotFound, body: `{}`})

		err := client.DeleteSystemUser(context.Background(), testEnvironmentHost, testPrincipalId, []int{http.StatusNoContent, http.StatusNotFound})

		require.NoError(t, err)
	})
}
