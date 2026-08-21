// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/require"
)

func newSolutionStagingTestClient(t *testing.T) *Client {
	t.Helper()
	cfg := &config.ProviderConfig{
		TestMode: true,
		Urls: config.ProviderConfigUrls{
			BapiUrl:               "api.bap.microsoft.com",
			PowerAppsUrl:          "api.powerapps.com",
			PowerAppsScope:        "https://service.powerapps.com/.default",
			PowerPlatformUrl:      "api.powerplatform.com",
			PowerPlatformScope:    "https://api.powerplatform.com/.default",
			AdminPowerPlatformUrl: "api.admin.powerplatform.microsoft.com",
			PowerAppsAdvisor:      "api.advisor.powerapps.com",
			PowerAppsAdvisorScope: "https://advisor.powerapps.com/.default",
			AnalyticsScope:        "https://adminanalytics.powerplatform.microsoft.com/.default",
		},
	}
	return NewClient(api.NewApiClientBase(cfg, api.NewAuthBase(cfg)))
}

func TestUnitSolutionStagingScopeMapping(t *testing.T) {
	require.Equal(t,
		[]string{"https://example.crm.dynamics.com/.default"},
		scopes("example.crm.dynamics.com"))
}

func TestUnitStageSolution_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var requestBody map[string]any
	httpmock.RegisterResponder("POST", "https://example.crm.dynamics.com/api/data/v9.2/StageSolution",
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, &requestBody))
			return httpmock.NewStringResponse(http.StatusOK, `{
  "StageSolutionResults":{
    "StageSolutionUploadId":"upload-id",
    "StageSolutionStatus":"Passed",
    "SolutionValidationResults":[],
    "MissingDependencies":[],
    "SolutionDetails":{"SolutionUniqueName":"MetaForm"}
  }
}`), nil
		})

	client := newSolutionStagingTestClient(t)
	stageResponse, err := client.StageSolution(context.Background(), "example.crm.dynamics.com", []byte("managed-package"))

	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"CustomizationFile": base64.StdEncoding.EncodeToString([]byte("managed-package")),
	}, requestBody)
	require.Equal(t, "upload-id", stageResponse.StageSolutionResults.StageSolutionUploadId)
	require.Equal(t, "MetaForm", stageResponse.StageSolutionResults.SolutionDetails.SolutionUniqueName)
	require.Equal(t, 1, httpmock.GetTotalCallCount())
}

func TestUnitStageSolution_NilContentFailsBeforeAnyRequest(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := newSolutionStagingTestClient(t)
	_, err := client.StageSolution(context.Background(), "example.crm.dynamics.com", nil)

	require.EqualError(t, err, "solution content is nil")
	require.Equal(t, 0, httpmock.GetTotalCallCount())
}

func TestUnitStageSolution_JoinsStagingFailures(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://example.crm.dynamics.com/api/data/v9.2/StageSolution",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, `{
  "StageSolutionResults":{
    "StageSolutionUploadId":"",
    "StageSolutionStatus":"Failed",
    "SolutionValidationResults":[{"Message":"solution manifest is invalid"}],
    "MissingDependencies":[{"RequiredComponentSchemaName":"msdyn_Dependency"}],
    "SolutionDetails":{"SolutionUniqueName":"MetaForm"}
  }
}`), nil
		})

	client := newSolutionStagingTestClient(t)
	_, err := client.StageSolution(context.Background(), "example.crm.dynamics.com", []byte("managed-package"))

	require.EqualError(t, err,
		"solution validation failed: solution manifest is invalid\n"+
			"missing dependency: 'msdyn_Dependency'\n"+
			"solution failed with status: 'Failed'")
}

func TestUnitStageSolution_TranslatesStatuses(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		checkError func(t *testing.T, err error)
	}{
		{
			name:   "forbidden maps to access denied",
			status: http.StatusForbidden,
			body:   `{"error":"forbidden"}`,
			checkError: func(t *testing.T, err error) {
				var accessDenied customerrors.AccessDeniedError
				require.ErrorAs(t, err, &accessDenied)
			},
		},
		{
			name:   "not found is reported",
			status: http.StatusNotFound,
			body:   `{"error":"not found"}`,
			checkError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "resource not found at ")
			},
		},
		{
			name:   "unexpected status is a status code error",
			status: http.StatusConflict,
			body:   `{"error":"conflict"}`,
			checkError: func(t *testing.T, err error) {
				var unexpected customerrors.UnexpectedHttpStatusCodeError
				require.ErrorAs(t, err, &unexpected)
				require.Equal(t, http.StatusConflict, unexpected.StatusCode)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponder("POST", "https://example.crm.dynamics.com/api/data/v9.2/StageSolution",
				httpmock.NewStringResponder(test.status, test.body))

			client := newSolutionStagingTestClient(t)
			_, err := client.StageSolution(context.Background(), "example.crm.dynamics.com", []byte("managed-package"))

			require.Error(t, err)
			test.checkError(t, err)
		})
	}
}

func TestUnitStartStagedSolutionImport_RequestShape(t *testing.T) {
	tests := []struct {
		name                          string
		operation                     StagedImportOperation
		url                           string
		componentParameters           []any
		skipProductUpdateDependencies bool
		wantComponentParameters       bool
	}{
		{
			name:                          "install without component parameters omits them",
			operation:                     StagedImportOperationInstall,
			url:                           "https://example.crm.dynamics.com/api/data/v9.2/ImportSolutionAsync",
			componentParameters:           nil,
			skipProductUpdateDependencies: true,
			wantComponentParameters:       false,
		},
		{
			name:      "stage and upgrade carries component parameters",
			operation: StagedImportOperationStageAndUpgrade,
			url:       "https://example.crm.dynamics.com/api/data/v9.2/StageAndUpgradeAsync",
			componentParameters: []any{StagedImportConnectionReferenceDto{
				Type:                           "Microsoft.Dynamics.CRM.connectionreference",
				ConnectionReferenceLogicalName: "new_ref",
				ConnectorId:                    "/providers/Microsoft.PowerApps/apis/shared_office365",
				ConnectionId:                   "11112222-3333-4444-5555-666677778888",
			}},
			skipProductUpdateDependencies: false,
			wantComponentParameters:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			var requestBody map[string]any
			httpmock.RegisterResponder("POST", test.url,
				func(req *http.Request) (*http.Response, error) {
					body, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(body, &requestBody))
					return httpmock.NewStringResponse(http.StatusOK, `{"ImportJobKey":"22222222-2222-2222-2222-222222222222","AsyncOperationId":"11111111-1111-1111-1111-111111111111"}`), nil
				})

			client := newSolutionStagingTestClient(t)
			importResponse, err := client.StartStagedSolutionImport(
				context.Background(),
				"example.crm.dynamics.com",
				test.operation,
				"upload-id",
				test.componentParameters,
				test.skipProductUpdateDependencies)

			require.NoError(t, err)
			require.Equal(t, true, requestBody["PublishWorkflows"])
			require.Equal(t, true, requestBody["OverwriteUnmanagedCustomizations"])
			require.Equal(t, test.skipProductUpdateDependencies, requestBody["SkipProductUpdateDependencies"])
			require.Equal(t, map[string]any{"StageSolutionUploadId": "upload-id"}, requestBody["SolutionParameters"])
			_, hasComponentParameters := requestBody["ComponentParameters"]
			require.Equal(t, test.wantComponentParameters, hasComponentParameters, "empty component parameters must be omitted from the import request")
			require.Equal(t, "22222222-2222-2222-2222-222222222222", importResponse.ImportJobKey)
			require.Equal(t, "11111111-1111-1111-1111-111111111111", importResponse.AsyncOperationId)
			require.Equal(t, 1, httpmock.GetTotalCallCount())
		})
	}
}

func TestUnitStartStagedSolutionImport_DoesNotReplayAmbiguousImportStart(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	attempts := 0
	httpmock.RegisterResponder("POST", "https://example.crm.dynamics.com/api/data/v9.2/ImportSolutionAsync",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusServiceUnavailable, `{"error":"ambiguous"}`), nil
		})

	client := newSolutionStagingTestClient(t)
	_, err := client.StartStagedSolutionImport(
		context.Background(),
		"example.crm.dynamics.com",
		StagedImportOperationInstall,
		"upload-id",
		nil,
		true)

	require.Error(t, err)
	require.Equal(t, 1, attempts, "an ambiguous ImportSolutionAsync response must not replay the import")
}

func TestUnitAwaitStagedSolutionImport_RetriesTransientStatusesAndPollsUntilCompleted(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	responses := []*http.Response{
		httpmock.NewStringResponse(http.StatusServiceUnavailable, `{"error":"transient"}`),
		httpmock.NewStringResponse(http.StatusOK, `{"completedon":""}`),
		httpmock.NewStringResponse(http.StatusOK, `{"completedon":"2026-07-13T00:00:00Z"}`),
	}
	calls := 0
	httpmock.RegisterResponder("GET", "https://example.crm.dynamics.com/api/data/v9.2/asyncoperations%2811111111-1111-1111-1111-111111111111%29",
		func(req *http.Request) (*http.Response, error) {
			response := responses[calls]
			calls++
			return response, nil
		})

	client := newSolutionStagingTestClient(t)
	err := client.AwaitStagedSolutionImport(context.Background(), "example.crm.dynamics.com", "11111111-1111-1111-1111-111111111111")

	require.NoError(t, err)
	require.Equal(t, 3, calls, "the completion poll must retry a transient status and keep polling until completedon is set")
}

func TestUnitValidateStagedSolutionImportResult_PassedResultSucceeds(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://example.crm.dynamics.com/api/data/v9.0/RetrieveSolutionImportResult%28ImportJobId=22222222-2222-2222-2222-222222222222%29",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, `{"SolutionOperationResult":{"Status":"Passed","ErrorMessages":[]}}`), nil
		})

	client := newSolutionStagingTestClient(t)
	err := client.ValidateStagedSolutionImportResult(context.Background(), "example.crm.dynamics.com", "22222222-2222-2222-2222-222222222222")

	require.NoError(t, err)
	require.Equal(t, 1, httpmock.GetTotalCallCount())
}

func TestUnitValidateStagedSolutionImportResult_JoinsErrorMessages(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://example.crm.dynamics.com/api/data/v9.0/RetrieveSolutionImportResult%28ImportJobId=22222222-2222-2222-2222-222222222222%29",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, `{"SolutionOperationResult":{"Status":"Failed","ErrorMessages":["first failure",{"Message":"second failure"}]}}`), nil
		})

	client := newSolutionStagingTestClient(t)
	err := client.ValidateStagedSolutionImportResult(context.Background(), "example.crm.dynamics.com", "22222222-2222-2222-2222-222222222222")

	require.EqualError(t, err, fmt.Sprintf("solution import failed: first failure; %v", map[string]any{"Message": "second failure"}))
}

func TestUnitPublishAllCustomizations_UsesDataversePublishAllXml(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	called := 0
	httpmock.RegisterResponder("POST", "https://example.crm.dynamics.com/api/data/v9.2/PublishAllXml",
		func(req *http.Request) (*http.Response, error) {
			called++
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	client := newSolutionStagingTestClient(t)
	err := client.PublishAllCustomizations(context.Background(), "example.crm.dynamics.com")

	require.NoError(t, err)
	require.Equal(t, 1, called)
}
