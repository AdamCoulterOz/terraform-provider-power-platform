// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package managedsolution

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/stretchr/testify/require"
)

func TestUnitApplyManagedSolution_UsesStageAndUpgradeForUpdates(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("no responder found for %s %s", req.Method, req.URL)
	})
	httpmock.RegisterResponder("GET", "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/00000000-0000-0000-0000-000000000001?%24expand=permissions%2Cproperties.capacity%2Cproperties%2FbillingPolicy&api-version=2023-06-01",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, `{
  "id":"00000000-0000-0000-0000-000000000001",
  "name":"env",
  "properties":{"linkedEnvironmentMetadata":{"instanceURL":"https://example.crm.dynamics.com/"}}
}`), nil
		})
	httpmock.RegisterResponder("POST", "https://example.crm.dynamics.com/api/data/v9.2/StageSolution",
		func(req *http.Request) (*http.Response, error) {
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
	stageAndUpgradeCalled := false
	httpmock.RegisterResponder("POST", "https://example.crm.dynamics.com/api/data/v9.2/StageAndUpgradeAsync",
		func(req *http.Request) (*http.Response, error) {
			stageAndUpgradeCalled = true
			return httpmock.NewStringResponse(http.StatusOK, `{"ImportJobKey":"22222222-2222-2222-2222-222222222222","AsyncOperationId":"11111111-1111-1111-1111-111111111111"}`), nil
		})
	httpmock.RegisterResponder("GET", "https://example.crm.dynamics.com/api/data/v9.2/asyncoperations%2811111111-1111-1111-1111-111111111111%29",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, `{"completedon":"2026-07-13T00:00:00Z"}`), nil
		})
	httpmock.RegisterResponder("GET", "https://example.crm.dynamics.com/api/data/v9.0/RetrieveSolutionImportResult%28ImportJobId=22222222-2222-2222-2222-222222222222%29",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, `{"SolutionOperationResult":{"Status":"Passed","ErrorMessages":[]}}`), nil
		})
	httpmock.RegisterResponder("GET", "https://example.crm.dynamics.com/api/data/v9.2/solutions?%24expand=publisherid&%24filter=uniquename+eq+%27MetaForm%27",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, `{"value":[{"solutionid":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","uniquename":"MetaForm","friendlyname":"Meta Form","ismanaged":true,"version":"2.0.246"}]}`), nil
		})

	cfg := &config.ProviderConfig{
		TestMode: true,
		Urls:     config.ProviderConfigUrls{BapiUrl: "api.bap.microsoft.com"},
	}
	client := NewManagedSolutionClient(api.NewApiClientBase(cfg, api.NewAuthBase(cfg)))
	solution, err := client.ApplyManagedSolution(
		context.Background(),
		"00000000-0000-0000-0000-000000000001",
		[]byte("managed-package"),
		nil,
		true)

	require.NoError(t, err)
	require.True(t, stageAndUpgradeCalled)
	require.Equal(t, "MetaForm", solution.Name)
}
