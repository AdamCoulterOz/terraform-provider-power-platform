// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEnvironmentGroupId   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	environmentGroupsUrl     = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/environmentGroups?api-version=2021-04-01"
	environmentGroupUrl      = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/environmentGroups/" + testEnvironmentGroupId + "?api-version=2021-04-01"
	environmentGroupBodyJson = `{"displayName":"Group","description":"Grouped environments","id":"` + testEnvironmentGroupId + `"}`
)

func TestUnitCreateEnvironmentGroup_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	httpmock.RegisterResponder(http.MethodPost, environmentGroupsUrl,
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusCreated, environmentGroupBodyJson), nil
		})

	group, err := client.CreateEnvironmentGroup(context.Background(), EnvironmentGroupDto{
		DisplayName: "Group",
		Description: "Grouped environments",
	})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.JSONEq(t, `{"displayName":"Group","description":"Grouped environments","createdBy":{}}`, gotBody)
	assert.Equal(t, testEnvironmentGroupId, group.Id)
}

func TestUnitGetEnvironmentGroup_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotUrl string
	httpmock.RegisterResponder(http.MethodGet, environmentGroupUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, environmentGroupBodyJson), nil
		})

	group, err := client.GetEnvironmentGroup(context.Background(), testEnvironmentGroupId)

	require.NoError(t, err)
	assert.Equal(t, environmentGroupUrl, gotUrl)
	assert.Equal(t, "Group", group.DisplayName)
}

func TestUnitGetEnvironmentGroup_NotFoundMeansNoGroup(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, environmentGroupUrl,
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"EnvironmentGroupNotFound"}}`))

	group, err := client.GetEnvironmentGroup(context.Background(), testEnvironmentGroupId)

	assert.NoError(t, err, "a missing environment group is not an error")
	assert.Nil(t, group)
}

func TestUnitUpdateEnvironmentGroup_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	httpmock.RegisterResponder(http.MethodPut, environmentGroupUrl,
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, environmentGroupBodyJson), nil
		})

	group, err := client.UpdateEnvironmentGroup(context.Background(), testEnvironmentGroupId, EnvironmentGroupDto{
		DisplayName: "Group",
		Description: "Grouped environments",
	})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.JSONEq(t, `{"displayName":"Group","description":"Grouped environments","createdBy":{}}`, gotBody)
	assert.Equal(t, testEnvironmentGroupId, group.Id)
}

func TestUnitDeleteEnvironmentGroup_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod string
	httpmock.RegisterResponder(http.MethodDelete, environmentGroupUrl,
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.DeleteEnvironmentGroup(context.Background(), testEnvironmentGroupId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
}

// A 409 is in the delete call's accepted status list, so the transport reports
// it without an error and the conflict-translation branch (environments still
// in the group, policy assigned to the group) is unreachable today: the delete
// reports success. This pins the current observed behavior.
func TestUnitDeleteEnvironmentGroup_ConflictIsCurrentlyReportedAsSuccess(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodDelete, environmentGroupUrl,
		httpmock.NewStringResponder(http.StatusConflict, `{"error":{"code":"EnvironmentsInEnvironmentGroup"}}`))

	err := client.DeleteEnvironmentGroup(context.Background(), testEnvironmentGroupId)

	assert.NoError(t, err)
}

func TestUnitGetEnvironmentsInEnvironmentGroup_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	expectedUrl := "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments?%24filter=properties%2FparentEnvironmentGroup%2Fid+eq+" + testEnvironmentGroupId + "&api-version=2021-04-01"
	var gotUrl string
	httpmock.RegisterResponder(http.MethodGet, expectedUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, `{"value":[{"name":"env-1","properties":{"displayName":"One"}}]}`), nil
		})

	envs, err := client.GetEnvironmentsInEnvironmentGroup(context.Background(), testEnvironmentGroupId)

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, gotUrl, "the list must filter on the parent environment group id")
	require.Len(t, envs, 1)
	assert.Equal(t, "env-1", envs[0].Name)
}
