// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAadObjectId = "cccccccc-cccc-cccc-cccc-cccccccccccc"

func roleAssignmentsUrl(environmentId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/roleAssignments?api-version=2021-04-01", environmentId)
}

func modifyRoleAssignmentsUrl(environmentId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/modifyRoleAssignments?api-version=2021-04-01", environmentId)
}

func roleAssignmentJson(assignmentName, roleDefinitionName, principalId, displayName string) string {
	return fmt.Sprintf(`{"id":"/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/roleAssignments/%s","type":"roleAssignment","name":"%s","properties":{"roleDefinition":{"id":"roledef","name":"%s"},"principal":{"id":"%s","displayName":"%s","type":"User"}}}`,
		testEnvironmentId, assignmentName, assignmentName, roleDefinitionName, principalId, displayName)
}

func TestUnitGetEnvironmentUserByAadObjectId_RequestShapeAndAssembly(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotUrl string
	body := fmt.Sprintf(`{"value":[%s,%s,%s]}`,
		roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, "Ada Lovelace"),
		roleAssignmentJson("assignment-maker", "EnvironmentMaker", testAadObjectId, "Ada Lovelace"),
		roleAssignmentJson("assignment-other", "EnvironmentAdmin", "dddddddd-dddd-dddd-dddd-dddddddddddd", "Someone Else"))
	httpmock.RegisterResponder(http.MethodGet, roleAssignmentsUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, body), nil
		})

	user, err := client.GetEnvironmentUserByAadObjectId(context.Background(), testEnvironmentId, testAadObjectId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, roleAssignmentsUrl(testEnvironmentId), gotUrl)
	assert.Equal(t, testAadObjectId, user.Id)
	assert.Equal(t, testAadObjectId, user.AadObjectId)
	assert.Equal(t, "Ada Lovelace", user.DomainName)
	assert.Equal(t, []EnvironmentUserSecurityRoleDto{
		{RoleId: RoleEnvironmentAdmin, Name: "assignment-admin"},
		{RoleId: RoleEnvironmentMaker, Name: "assignment-maker"},
	}, user.SecurityRoles, "only the requested principal's admin and maker assignments must be mapped")
}

func TestUnitAddEnvironmentUserSecurityRoles_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, modifyRoleAssignmentsUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK,
				fmt.Sprintf(`{"add":[{"roleAssignment":%s,"httpStatus":"OK"}]}`,
					roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, "Ada Lovelace"))), nil
		})

	user, err := client.AddEnvironmentUserSecurityRoles(context.Background(), testEnvironmentId, testAadObjectId, []string{"Environment Admin"})

	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"add":[{"properties":{"roleDefinition":{"id":"/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/roleDefinitions/environmentadmin"},"principal":{"id":"%s","type":"User"}}}]}`, testEnvironmentId, testAadObjectId),
		gotBody, "role names must be lowercased and despaced into the role definition id")
	assert.Equal(t, "Ada Lovelace", user.DomainName)
	assert.Equal(t, []EnvironmentUserSecurityRoleDto{{RoleId: "EnvironmentAdmin", Name: "assignment-admin"}}, user.SecurityRoles)
}

func TestUnitAddEnvironmentUserSecurityRoles_PollsForDisplayName(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPost, modifyRoleAssignmentsUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusOK,
			fmt.Sprintf(`{"add":[{"roleAssignment":%s,"httpStatus":"OK"}]}`,
				roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, ""))))
	getAttempts := 0
	httpmock.RegisterResponder(http.MethodGet, roleAssignmentsUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			getAttempts++
			return httpmock.NewStringResponse(http.StatusOK,
				fmt.Sprintf(`{"value":[%s]}`, roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, "Resolved User"))), nil
		})

	user, err := client.AddEnvironmentUserSecurityRoles(context.Background(), testEnvironmentId, testAadObjectId, []string{"Environment Admin"})

	require.NoError(t, err)
	assert.Equal(t, 1, getAttempts, "a missing display name must be re-read from the role assignments")
	assert.Equal(t, "Resolved User", user.DomainName)
}

func TestUnitAddEnvironmentUserSecurityRoles_ToleratesUnresolvableDisplayName(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPost, modifyRoleAssignmentsUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusOK,
			fmt.Sprintf(`{"add":[{"roleAssignment":%s,"httpStatus":"OK"}]}`,
				roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, ""))))
	getAttempts := 0
	httpmock.RegisterResponder(http.MethodGet, roleAssignmentsUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			getAttempts++
			return httpmock.NewStringResponse(http.StatusOK,
				fmt.Sprintf(`{"value":[%s]}`, roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, ""))), nil
		})

	user, err := client.AddEnvironmentUserSecurityRoles(context.Background(), testEnvironmentId, testAadObjectId, []string{"Environment Admin"})

	require.NoError(t, err, "a display name that never resolves is not an error")
	assert.Equal(t, environmentUserDisplayNameRetryCount, getAttempts)
	assert.Empty(t, user.DomainName)
}

func TestUnitCreateEnvironmentUser_DelegatesToModifyRoleAssignments(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPost, modifyRoleAssignmentsUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusOK,
			fmt.Sprintf(`{"add":[{"roleAssignment":%s,"httpStatus":"OK"}]}`,
				roleAssignmentJson("assignment-maker", "EnvironmentMaker", testAadObjectId, "Ada Lovelace"))))

	user, err := client.CreateEnvironmentUser(context.Background(), testEnvironmentId, testAadObjectId, []string{"Environment Maker"})

	require.NoError(t, err)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
	assert.Equal(t, []EnvironmentUserSecurityRoleDto{{RoleId: "EnvironmentMaker", Name: "assignment-maker"}}, user.SecurityRoles)
}

func TestUnitRemoveEnvironmentUserSecurityRoles_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	getAttempts := 0
	httpmock.RegisterResponder(http.MethodGet, roleAssignmentsUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			getAttempts++
			if getAttempts == 1 {
				return httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{"value":[%s,%s]}`,
					roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, "Ada Lovelace"),
					roleAssignmentJson("assignment-maker", "EnvironmentMaker", testAadObjectId, "Ada Lovelace"))), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{"value":[%s]}`,
				roleAssignmentJson("assignment-maker", "EnvironmentMaker", testAadObjectId, "Ada Lovelace"))), nil
		})

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, modifyRoleAssignmentsUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, `{"remove":[]}`), nil
		})

	savedRoles := []EnvironmentUserSecurityRoleDto{{RoleId: "Environment Admin", Name: "assignment-admin"}}
	user, err := client.RemoveEnvironmentUserSecurityRoles(context.Background(), testEnvironmentId, testAadObjectId, []string{"Environment Admin"}, savedRoles)

	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"remove":[{"id":"/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/roleAssignments/assignment-admin"}]}`, testEnvironmentId),
		gotBody, "removals must reference the saved role assignment by name")
	assert.Equal(t, 2, getAttempts, "the user must be read before and after the removal")
	assert.Equal(t, "Ada Lovelace", user.DomainName)
	assert.Equal(t, []EnvironmentUserSecurityRoleDto{{RoleId: RoleEnvironmentMaker, Name: "assignment-maker"}}, user.SecurityRoles)
}

func TestUnitRemoveEnvironmentUserSecurityRoles_ToleratesInternalServerError(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, roleAssignmentsUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusOK, fmt.Sprintf(`{"value":[%s]}`,
			roleAssignmentJson("assignment-admin", "EnvironmentAdmin", testAadObjectId, "Ada Lovelace"))))
	postAttempts := 0
	httpmock.RegisterResponder(http.MethodPost, modifyRoleAssignmentsUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			postAttempts++
			return httpmock.NewStringResponse(http.StatusInternalServerError, `{"error":{"code":"RemovalFailed"}}`), nil
		})

	savedRoles := []EnvironmentUserSecurityRoleDto{{RoleId: "Environment Admin", Name: "assignment-admin"}}
	user, err := client.RemoveEnvironmentUserSecurityRoles(context.Background(), testEnvironmentId, testAadObjectId, []string{"Environment Admin"}, savedRoles)

	require.NoError(t, err, "a 500 on removal is an accepted status and only logged")
	assert.Equal(t, 1, postAttempts, "the accepted 500 must not be retried by the transport")
	assert.NotNil(t, user)
}

func addUserUrl(environmentId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/addUser?api-version=2023-06-01", environmentId)
}

func TestUnitAddUserToEnvironment_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	attempts := 0
	httpmock.RegisterResponder(http.MethodPost, addUserUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.AddUserToEnvironment(context.Background(), testEnvironmentId, testAadObjectId)

	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"objectId":"%s"}`, testAadObjectId), gotBody)
	assert.Equal(t, 1, attempts)
}

func TestUnitAddUserToEnvironment_RetriesWhileUserNotLicensed(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodPost, addUserUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusBadRequest, `{"error":{"code":"userNotLicensed","message":"license assignment pending"}}`), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.AddUserToEnvironment(context.Background(), testEnvironmentId, testAadObjectId)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "a userNotLicensed failure must be retried until the license propagates")
}

func TestUnitAddUserToEnvironment_OtherErrorsAreNotRetried(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodPost, addUserUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusBadRequest, `{"error":{"code":"UserNotFoundInTenant"}}`), nil
		})

	err := client.AddUserToEnvironment(context.Background(), testEnvironmentId, testAadObjectId)

	require.Error(t, err)
	assert.ErrorContains(t, err, "UserNotFoundInTenant")
	assert.Equal(t, 1, attempts)
}
