// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSolutionId   = "abcdefab-0000-0000-0000-000000000001"
	testSolutionBody = `{"solutionid":"abcdefab-0000-0000-0000-000000000001","uniquename":"MySolution","friendlyname":"My Solution","ismanaged":true,"version":"1.0.0.0"}`
	testDefinitionId = "eeeeeeee-1111-0000-0000-000000000001"
	testValueId      = "eeeeeeee-2222-0000-0000-000000000002"
)

func TestUnitGetSolutionUniqueName_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + testSolutionBody + `]}`})

	solution, err := client.GetSolutionUniqueName(context.Background(), testEnvironmentHost, "MySolution")

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/solutions?%24expand=publisherid&%24filter=uniquename+eq+%27MySolution%27"},
	})
	assert.Equal(t, testSolutionId, solution.Id)
	assert.Equal(t, "MySolution", solution.Name)
}

func TestUnitGetSolutionUniqueName_EmptyResultIsProviderError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

	_, err := client.GetSolutionUniqueName(context.Background(), testEnvironmentHost, "Missing")

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
	assert.Contains(t, err.Error(), "solution with unique name 'Missing' not found")
}

func TestUnitGetSolutionById_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + testSolutionBody + `]}`})

	solution, err := client.GetSolutionById(context.Background(), testEnvironmentHost, testSolutionId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/solutions?%24expand=publisherid&%24filter=solutionid+eq+" + testSolutionId},
	})
	assert.Equal(t, testSolutionId, solution.Id)
}

func TestUnitGetSolutionById_EmptyResultIsProviderError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

	_, err := client.GetSolutionById(context.Background(), testEnvironmentHost, testSolutionId)

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
	assert.Contains(t, err.Error(), "solution with id '"+testSolutionId+"' not found")
}

func TestUnitGetSolutions_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + testSolutionBody + `]}`})

	solutions, err := client.GetSolutions(context.Background(), testEnvironmentHost)

	require.NoError(t, err)
	// Visible solutions only, newest first.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/solutions?%24expand=publisherid&%24filter=%28isvisible+eq+true%29&%24orderby=createdon+desc"},
	})
	require.Len(t, solutions, 1)
}

func TestUnitGetInstalledSolutions_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + testSolutionBody + `]}`})

	solutions, err := client.GetInstalledSolutions(context.Background(), testEnvironmentHost)

	require.NoError(t, err)
	// No visibility filter: every installed solution is listed.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/solutions?%24expand=publisherid&%24orderby=createdon+desc"},
	})
	require.Len(t, solutions, 1)
}

func TestUnitDeleteSolution_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusNoContent})

	err := client.DeleteSolution(context.Background(), testEnvironmentHost, testSolutionId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/solutions%28" + testSolutionId + "%29"},
	})
}

// TestUnitCreateSolution_RequestShapes pins the wire shape of each call the
// import workflow makes once: stage, async import, completion poll, import
// result validation, and the final read-back. TestMode makes the poll sleeps
// instant.
func TestUnitCreateSolution_RequestShapes(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusOK, body: `{"StageSolutionResults":{"StageSolutionUploadId":"eeeeeeee-0000-0000-0000-000000000001","StageSolutionStatus":"Passed","SolutionDetails":{"SolutionUniqueName":"MySolution"}}}`},
		step{status: http.StatusOK, body: `{"ImportJobKey":"ffffffff-0000-0000-0000-000000000002","AsyncOperationId":"ffffffff-0000-0000-0000-000000000001"}`},
		step{status: http.StatusOK, body: `{"completedon":""}`},
		step{status: http.StatusOK, body: `{"completedon":"2026-08-21T00:00:00Z"}`},
		step{status: http.StatusOK, body: `{"SolutionOperationResult":{"Status":"Passed"}}`},
		step{status: http.StatusOK, body: `{"value":[` + testSolutionBody + `]}`},
	)

	settings := []byte(`{
		"connectionreferences": [{"logicalname": "cr_logical", "connectionid": "conn1", "connectorid": "shared_sql"}],
		"environmentvariables": [{"schemaname": "var_one", "value": "v1"}, {"schemaname": "var_skipped", "value": ""}]
	}`)
	solution, err := client.CreateSolution(context.Background(), testEnvironmentHost, []byte("test"), settings)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{
			method: "POST",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/StageSolution",
			body:   `{"CustomizationFile":"dGVzdA=="}`,
		},
		{
			method: "POST",
			url:    "https://env.crm.dynamics.com/api/data/v9.2/ImportSolutionAsync",
			body: `{
				"PublishWorkflows": true,
				"OverwriteUnmanagedCustomizations": false,
				"ComponentParameters": [
					{"@odata.type": "Microsoft.Dynamics.CRM.connectionreference", "connectionreferencedisplayname": "", "connectionreferencelogicalname": "cr_logical", "description": "", "connectorid": "shared_sql", "connectionid": "conn1"},
					{"@odata.type": "Microsoft.Dynamics.CRM.environmentvariablevalue", "schemaname": "var_one", "value": "v1"}
				],
				"SolutionParameters": {"StageSolutionUploadId": "eeeeeeee-0000-0000-0000-000000000001"}
			}`,
		},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/asyncoperations%28ffffffff-0000-0000-0000-000000000001%29"},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/asyncoperations%28ffffffff-0000-0000-0000-000000000001%29"},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.0/RetrieveSolutionImportResult%28ImportJobId=ffffffff-0000-0000-0000-000000000002%29"},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/solutions?%24expand=publisherid&%24filter=uniquename+eq+%27MySolution%27"},
	})
	assert.Equal(t, testSolutionId, solution.Id)
}

func TestUnitCreateSolution_NilContentFails(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t)

	_, err := client.CreateSolution(context.Background(), testEnvironmentHost, nil, nil)

	require.Error(t, err)
	assert.EqualError(t, err, "solution content is nil")
	require.Empty(t, *recorded, "nil content must fail before any request")
}

func TestUnitCreateSolution_StageFailureReportsDetails(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{
		"StageSolutionResults": {
			"StageSolutionStatus": "Failed",
			"MissingDependencies": [{"RequiredComponentSchemaName": "dep_schema"}],
			"SolutionValidationResults": [{"Message": "component is invalid"}]
		}
	}`})

	_, err := client.CreateSolution(context.Background(), testEnvironmentHost, []byte("test"), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "solution failed with status: 'Failed'")
	assert.Contains(t, err.Error(), "missing dependency: 'dep_schema'")
	assert.Contains(t, err.Error(), "solution validation failed: component is invalid")
}

func TestUnitGetEnvironmentVariableDefinitionBySchemaName_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"environmentvariabledefinitionid":"` + testDefinitionId + `","schemaname":"var_o'brien","type":100000000}]}`})

	definition, err := client.GetEnvironmentVariableDefinitionBySchemaName(context.Background(), testEnvironmentHost, "var_o'brien")

	require.NoError(t, err)
	// The quote in the schema name is doubled per OData literal rules.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/environmentvariabledefinitions?%24filter=schemaname+eq+%27var_o%27%27brien%27&%24select=environmentvariabledefinitionid%2Cschemaname%2Cdisplayname%2Cdescription%2Cdefaultvalue%2Ctype%2Cvalueschema%2Csecretstore"},
	})
	assert.Equal(t, testDefinitionId, definition.EnvironmentVariableDefinitionId)
}

func TestUnitGetEnvironmentVariableDefinitionBySchemaName_ResultCardinality(t *testing.T) {
	t.Run("no match is OBJECT_NOT_FOUND", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

		_, err := client.GetEnvironmentVariableDefinitionBySchemaName(context.Background(), testEnvironmentHost, "var_missing")

		require.Error(t, err)
		assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
		assert.Contains(t, err.Error(), "environment variable definition with schema name 'var_missing' not found")
	})
	t.Run("multiple matches fail", func(t *testing.T) {
		client := newTestClient(t)
		activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"environmentvariabledefinitionid":"a"},{"environmentvariabledefinitionid":"b"}]}`})

		_, err := client.GetEnvironmentVariableDefinitionBySchemaName(context.Background(), testEnvironmentHost, "var_dup")

		require.Error(t, err)
		assert.EqualError(t, err, "multiple environment variable definitions found for schema name 'var_dup'")
	})
}

func TestUnitGetEnvironmentVariableValueByDefinitionId_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"environmentvariablevalueid":"` + testValueId + `","schemaname":"var_one","value":"v1"}]}`})

	value, err := client.GetEnvironmentVariableValueByDefinitionId(context.Background(), testEnvironmentHost, testDefinitionId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/environmentvariablevalues?%24filter=_environmentvariabledefinitionid_value+eq+" + testDefinitionId + "&%24select=environmentvariablevalueid%2Cschemaname%2Cvalue"},
	})
	require.NotNil(t, value)
	assert.Equal(t, "v1", value.Value)
}

func TestUnitGetEnvironmentVariableValueByDefinitionId_NoValueIsNilNotError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

	value, err := client.GetEnvironmentVariableValueByDefinitionId(context.Background(), testEnvironmentHost, testDefinitionId)

	require.NoError(t, err, "a variable without a current value is not an error")
	assert.Nil(t, value)
}

func TestUnitCreateEnvironmentVariableValue_RequestShape(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
	}{
		{
			name:   "id from OData-EntityId header",
			header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/environmentvariablevalues(" + testValueId + ")"),
		},
		{
			name:   "id falls back to Location header",
			header: http.Header{"Location": []string{"https://env.crm.dynamics.com/api/data/v9.2/environmentvariablevalues(" + testValueId + ")"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			recorded := activateRecorder(t, step{status: http.StatusCreated, header: tt.header})

			valueId, err := client.CreateEnvironmentVariableValue(context.Background(), testEnvironmentHost, testDefinitionId, "var_one", "v1")

			require.NoError(t, err)
			assertRequests(t, recorded, []recordedRequest{
				{
					method: "POST",
					url:    "https://env.crm.dynamics.com/api/data/v9.2/environmentvariablevalues",
					body:   `{"schemaname":"var_one","value":"v1","EnvironmentVariableDefinitionId@odata.bind":"/environmentvariabledefinitions(` + testDefinitionId + `)"}`,
				},
			})
			assert.Equal(t, testValueId, valueId)
		})
	}
}

func TestUnitCreateEnvironmentVariableValue_MissingIdHeadersFail(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusNoContent})

	_, err := client.CreateEnvironmentVariableValue(context.Background(), testEnvironmentHost, testDefinitionId, "var_one", "v1")

	require.Error(t, err)
	assert.EqualError(t, err, "no environment variable value id returned after creating schema name 'var_one'")
}

func TestUnitUpdateEnvironmentVariableValue_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusNoContent})

	err := client.UpdateEnvironmentVariableValue(context.Background(), testEnvironmentHost, testValueId, "v2")

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "PATCH", url: "https://env.crm.dynamics.com/api/data/v9.2/environmentvariablevalues%28" + testValueId + "%29", body: `{"value":"v2"}`},
	})
}

func TestUnitGetEnvironmentVariableValueById_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"environmentvariablevalueid":"` + testValueId + `","schemaname":"var_one","value":"v1"}`})

	value, err := client.GetEnvironmentVariableValueById(context.Background(), testEnvironmentHost, testValueId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/environmentvariablevalues%28" + testValueId + "%29?%24select=environmentvariablevalueid%2Cschemaname%2Cvalue"},
	})
	assert.Equal(t, "v1", value.Value)
}

func TestUnitDeleteEnvironmentVariableValue_RequestShape(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "deleted", status: http.StatusNoContent},
		{name: "already gone is not an error", status: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			recorded := activateRecorder(t, step{status: tt.status})

			err := client.DeleteEnvironmentVariableValue(context.Background(), testEnvironmentHost, testValueId)

			require.NoError(t, err)
			assertRequests(t, recorded, []recordedRequest{
				{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/environmentvariablevalues%28" + testValueId + "%29"},
			})
		})
	}
}

// TestUnitCreateEnvironmentVariableValue_ODataBindKeyIsNavigationProperty pins
// the @odata.bind key of the environmentvariablevalue create payload.
//
// Dataverse requires the key to be the relationship's
// ReferencingEntityNavigationPropertyName, which for
// environmentvariablevalue -> environmentvariabledefinition is
// "EnvironmentVariableDefinitionId". Binding to the lookup column name
// "environmentvariabledefinitionid" is rejected with 0x80048d19 ("undeclared
// property"), so a regression here breaks every create of a variable that has
// no value yet.
func TestUnitCreateEnvironmentVariableValue_ODataBindKeyIsNavigationProperty(t *testing.T) {
	const ref = "/environmentvariabledefinitions(" + testDefinitionId + ")"

	payload, err := json.Marshal(CreateEnvironmentVariableValueDto{
		SchemaName:                            "var_one",
		Value:                                 "v1",
		EnvironmentVariableDefinitionODataRef: ref,
	})
	require.NoError(t, err)

	decoded := map[string]any{}
	require.NoError(t, json.Unmarshal(payload, &decoded))

	assert.Equal(t, ref, decoded["EnvironmentVariableDefinitionId@odata.bind"])
	assert.NotContains(t, decoded, "environmentvariabledefinitionid@odata.bind",
		"@odata.bind must use the navigation property name, not the lookup column name")
}
