// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSystemUserId = "5ad55f2c-51f9-4d2f-b199-9ea23cf47dcd"
	testAadObjectId  = "aaaaaaaa-0000-0000-0000-000000000001"
	testRoleId       = "11111111-2222-3333-4444-555555555555"

	// systemUserExpandQuery is the encoded $expand both user reads send.
	systemUserExpandQuery = "%24expand=systemuserroles_association%28%24select%3Droleid%2Cname%2Cismanaged%2C_businessunitid_value%29"

	testUserBody = `{
		"systemuserid": "5ad55f2c-51f9-4d2f-b199-9ea23cf47dcd",
		"domainname": "user@contoso.com",
		"firstname": "First",
		"lastname": "Last",
		"azureactivedirectoryobjectid": "aaaaaaaa-0000-0000-0000-000000000001",
		"_businessunitid_value": "cccccccc-0000-0000-0000-000000000001",
		"systemuserroles_association": [
			{"roleid": "11111111-2222-3333-4444-555555555555", "name": "Sales Person", "ismanaged": true, "_businessunitid_value": "cccccccc-0000-0000-0000-000000000001"}
		]
	}`
)

func TestUnitGetUsers_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + testUserBody + `]}`})

	users, err := client.GetUsers(context.Background(), testEnvironmentHost)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers"},
	})
	require.Len(t, users, 1)
	assert.Equal(t, "user@contoso.com", users[0].DomainName)
}

func TestUnitGetUserBySystemUserId_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: testUserBody})

	user, err := client.GetUserBySystemUserId(context.Background(), testEnvironmentHost, testSystemUserId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29?" + systemUserExpandQuery},
	})
	assert.Equal(t, testSystemUserId, user.Id)
	assert.Equal(t, []string{testRoleId}, user.SecurityRolesArray())
}

func TestUnitGetUserBySystemUserId_NotFound(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusNotFound, body: `{"error":{"code":"0x80040217"}}`})

	_, err := client.GetUserBySystemUserId(context.Background(), testEnvironmentHost, testSystemUserId)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource not found at 'https://env.crm.dynamics.com/api/data/v9.2/systemusers%28"+testSystemUserId+"%29")
}

// TestUnitGetUserByAadObjectId_PollsUntilVisible pins the read-after-write
// poll: newly created users are not immediately queryable in Dataverse, so the
// lookup by AAD object id repeats until a row appears. TestMode makes the
// inter-attempt sleeps instant.
func TestUnitGetUserByAadObjectId_PollsUntilVisible(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusOK, body: `{"value":[]}`},
		step{status: http.StatusOK, body: `{"value":[]}`},
		step{status: http.StatusOK, body: `{"value":[` + testUserBody + `]}`},
	)

	user, err := client.GetUserByAadObjectId(context.Background(), testEnvironmentHost, testAadObjectId)

	require.NoError(t, err)
	require.Len(t, *recorded, 3, "the lookup must repeat until the user row is visible")
	wantURL := "https://env.crm.dynamics.com/api/data/v9.2/systemusers?" + systemUserExpandQuery + "&%24filter=azureactivedirectoryobjectid+eq+" + testAadObjectId
	for _, req := range *recorded {
		assert.Equal(t, "GET", req.method)
		assert.Equal(t, wantURL, req.url)
	}
	assert.Equal(t, testSystemUserId, user.Id)
}

func TestUnitGetUserByAadObjectId_GivesUpAfterMaxRetries(t *testing.T) {
	client := newTestClient(t)
	emptySteps := make([]step, 0, 30)
	for range 30 {
		emptySteps = append(emptySteps, step{status: http.StatusOK, body: `{"value":[]}`})
	}
	recorded := activateRecorder(t, emptySteps...)

	_, err := client.GetUserByAadObjectId(context.Background(), testEnvironmentHost, testAadObjectId)

	require.Error(t, err)
	require.Len(t, *recorded, 30, "the poll is capped at 30 attempts")
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound), "exhaustion must surface as OBJECT_NOT_FOUND")
	assert.EqualError(t, err, "OBJECT_NOT_FOUND: Dataverse user with AAD object id '"+testAadObjectId+"' not found after 30 retries")
}

func TestUnitUpdateUser_PatchesThenReadsBack(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusOK},
		step{status: http.StatusOK, body: testUserBody},
	)

	user, err := client.UpdateUser(context.Background(), testEnvironmentHost, testSystemUserId, &UserDto{DomainName: "user@contoso.com"})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{
			method: "PATCH",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29",
			body:   `{"systemuserid":"","domainname":"user@contoso.com","firstname":"","lastname":"","azureactivedirectoryobjectid":"","_businessunitid_value":""}`,
		},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29?" + systemUserExpandQuery},
	})
	assert.Equal(t, testSystemUserId, user.Id)
}

func TestUnitDeleteUser_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusNoContent})

	err := client.DeleteUser(context.Background(), testEnvironmentHost, testSystemUserId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29"},
	})
}

func TestUnitDeleteUser_UnexpectedStatusCode(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{}`})

	err := client.DeleteUser(context.Background(), testEnvironmentHost, testSystemUserId)

	require.Error(t, err)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusOK, statusErr.StatusCode)
}

func TestUnitAddSecurityRoles_AssociatesThenReadsBack(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusNoContent},
		step{status: http.StatusOK, body: testUserBody},
	)

	user, err := client.AddSecurityRoles(context.Background(), testEnvironmentHost, testSystemUserId, []string{testRoleId})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{
			method: "POST",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29/systemuserroles_association/$ref",
			body:   `{"@odata.id":"https://env.crm.dynamics.com/api/data/v9.2/roles(` + testRoleId + `)"}`,
		},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29?" + systemUserExpandQuery},
	})
	assert.Equal(t, testSystemUserId, user.Id)
}

func TestUnitAddSecurityRoles_InvalidRoleIsTranslated(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusBadRequest, body: `{"error":{"code":"0x80060888","message":"role ` + testRoleId + ` does not exist"}}`})

	_, err := client.AddSecurityRoles(context.Background(), testEnvironmentHost, testSystemUserId, []string{testRoleId})

	require.Error(t, err)
	assert.EqualError(t, err, "role with id '"+testRoleId+"' is not valid")
}

func TestUnitRemoveSecurityRoles_DisassociatesThenReadsBack(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusNoContent},
		step{status: http.StatusOK, body: testUserBody},
	)

	user, err := client.RemoveSecurityRoles(context.Background(), testEnvironmentHost, testSystemUserId, []string{testRoleId})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{
			method: "DELETE",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29/systemuserroles_association/$ref?%24id=https%3A%2F%2Fenv.crm.dynamics.com%2Fapi%2Fdata%2Fv9.2%2Froles%28" + testRoleId + "%29",
		},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/systemusers%28" + testSystemUserId + "%29?" + systemUserExpandQuery},
	})
	assert.Equal(t, testSystemUserId, user.Id)
}

func TestUnitRemoveSecurityRoles_InvalidRoleIsTranslated(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusBadRequest, body: `{"error":{"code":"0x80060888","message":"role ` + testRoleId + ` does not exist"}}`})

	_, err := client.RemoveSecurityRoles(context.Background(), testEnvironmentHost, testSystemUserId, []string{testRoleId})

	require.Error(t, err)
	assert.EqualError(t, err, "role with id '"+testRoleId+"' is not valid")
}

func TestUnitGetSecurityRoles_RequestShape(t *testing.T) {
	tests := []struct {
		name           string
		businessUnitId string
		wantURL        string
	}{
		{
			name:           "unfiltered",
			businessUnitId: "",
			wantURL:        "https://env.crm.dynamics.com/api/data/v9.2/roles",
		},
		{
			name:           "filtered by business unit",
			businessUnitId: "cccccccc-0000-0000-0000-000000000001",
			wantURL:        "https://env.crm.dynamics.com/api/data/v9.2/roles?%24filter=_businessunitid_value+eq+cccccccc-0000-0000-0000-000000000001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"roleid":"` + testRoleId + `","name":"Sales Person","ismanaged":false,"_businessunitid_value":"cccccccc-0000-0000-0000-000000000001"}]}`})

			roles, err := client.GetSecurityRoles(context.Background(), testEnvironmentHost, tt.businessUnitId)

			require.NoError(t, err)
			assertRequests(t, recorded, []recordedRequest{
				{method: "GET", url: tt.wantURL},
			})
			require.Len(t, roles, 1)
			assert.Equal(t, "Sales Person", roles[0].Name)
		})
	}
}
