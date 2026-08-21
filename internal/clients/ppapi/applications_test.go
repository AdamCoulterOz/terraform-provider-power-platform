// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
)

const testApplicationUniqueName = "msdyn_AppOne"

// applicationsQuery pins the api-version every appmanagement operation must send.
func applicationsQuery() url.Values {
	return url.Values{"api-version": []string{"2022-03-01-preview"}}
}

func TestUnitGetTenantApplications_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/appmanagement/applicationPackages", applicationsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, TenantApplicationArrayDto{Value: []TenantApplicationDto{
				{ApplicationId: "app-1", UniqueName: testApplicationUniqueName},
			}})
		})

	applications, err := newTestClient(t).GetTenantApplications(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, []TenantApplicationDto{{ApplicationId: "app-1", UniqueName: testApplicationUniqueName}}, applications)
}

func TestUnitGetApplicationsByEnvironmentId_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Environment application packages are served from the flat host with the
	// environment id in the path, not from the environment-scoped host.
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/appmanagement/environments/"+testEnvironmentId+"/applicationPackages", applicationsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, EnvironmentApplicationArrayDto{Value: []EnvironmentApplicationDto{
				{ApplicationId: "app-1", UniqueName: testApplicationUniqueName, State: "Installed"},
			}})
		})

	applications, err := newTestClient(t).GetApplicationsByEnvironmentId(context.Background(), testEnvironmentId)

	assert.NoError(t, err)
	assert.Equal(t, []EnvironmentApplicationDto{{ApplicationId: "app-1", UniqueName: testApplicationUniqueName, State: "Installed"}}, applications)
}

func TestUnitInstallApplicationInEnvironment_PollsLifecycleUntilSucceeded(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	operationUrl := "https://api.powerplatform.com/appmanagement/environments/" + testEnvironmentId + "/operations/op-1"
	installedApplicationId := "11112222-3333-4444-5555-666677778888"

	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/appmanagement/environments/"+testEnvironmentId+"/applicationPackages/"+testApplicationUniqueName+"/install", applicationsQuery(),
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(http.StatusAccepted, "")
			resp.Header.Set("Operation-Location", operationUrl)
			return resp, nil
		})

	polls := 0
	httpmock.RegisterResponder("GET", operationUrl,
		func(req *http.Request) (*http.Response, error) {
			polls++
			if polls == 1 {
				// A 409 is an accepted status that means "check again".
				return httpmock.NewStringResponse(http.StatusConflict, ""), nil
			}
			return httpmock.NewJsonResponse(http.StatusOK, EnvironmentApplicationLifecycleDto{
				Status:          "Succeeded",
				CreatedDateTime: "lifecycleOperations/" + installedApplicationId,
			})
		})

	applicationId, err := newTestClient(t).InstallApplicationInEnvironment(context.Background(), testEnvironmentId, testApplicationUniqueName)

	assert.NoError(t, err)
	assert.Equal(t, 2, polls, "the lifecycle operation must be polled until it reports Succeeded")
	assert.Equal(t, installedApplicationId, applicationId, "the application id is the last path segment of createdDateTime")
}

func TestUnitInstallApplicationInEnvironment_FailedLifecycle(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	operationUrl := "https://api.powerplatform.com/appmanagement/environments/" + testEnvironmentId + "/operations/op-2"

	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/appmanagement/environments/"+testEnvironmentId+"/applicationPackages/"+testApplicationUniqueName+"/install", applicationsQuery(),
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(http.StatusAccepted, "")
			resp.Header.Set("Operation-Location", operationUrl)
			return resp, nil
		})

	httpmock.RegisterResponder("GET", operationUrl,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, EnvironmentApplicationLifecycleDto{
				Status: "Failed",
				Error:  EnvironmentApplicationLifecycleErrorDto{Message: "dependency missing"},
			})
		})

	applicationId, err := newTestClient(t).InstallApplicationInEnvironment(context.Background(), testEnvironmentId, testApplicationUniqueName)

	assert.Empty(t, applicationId)
	assert.ErrorContains(t, err, "application installation failed. status message: dependency missing")
}

func TestUnitInstallApplicationInEnvironment_LifecyclePollErrorPropagates(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	operationUrl := "https://api.powerplatform.com/appmanagement/environments/" + testEnvironmentId + "/operations/op-3"

	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/appmanagement/environments/"+testEnvironmentId+"/applicationPackages/"+testApplicationUniqueName+"/install", applicationsQuery(),
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(http.StatusAccepted, "")
			resp.Header.Set("Operation-Location", operationUrl)
			return resp, nil
		})

	httpmock.RegisterResponder("GET", operationUrl,
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	applicationId, err := newTestClient(t).InstallApplicationInEnvironment(context.Background(), testEnvironmentId, testApplicationUniqueName)

	assert.Empty(t, applicationId)
	var httpErr customerrors.UnexpectedHttpStatusCodeError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
}

func TestUnitInstallApplicationInEnvironment_MissingOperationLocationHeader(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/appmanagement/environments/"+testEnvironmentId+"/applicationPackages/"+testApplicationUniqueName+"/install", applicationsQuery(),
		httpmock.NewStringResponder(http.StatusAccepted, ""))

	applicationId, err := newTestClient(t).InstallApplicationInEnvironment(context.Background(), testEnvironmentId, testApplicationUniqueName)

	assert.Empty(t, applicationId)
	assert.ErrorContains(t, err, "missing operation location header in response")
}
