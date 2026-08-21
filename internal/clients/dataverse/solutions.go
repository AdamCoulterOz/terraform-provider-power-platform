// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

// GetSolutionUniqueName reads a solution by its unique name.
func (client *Client) GetSolutionUniqueName(ctx context.Context, environmentHost, name string) (*SolutionDto, error) {
	values := url.Values{}
	values.Add("$expand", "publisherid")
	values.Add("$filter", fmt.Sprintf("uniquename eq '%s'", name))

	solutions := SolutionArrayDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", helpers.BuildApiUrl(environmentHost, "/api/data/v9.2/solutions", values), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &solutions)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	if len(solutions.Value) == 0 {
		baseErr := fmt.Errorf("solution with unique name '%s' not found", name)
		return nil, customerrors.WrapIntoProviderError(baseErr, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), baseErr.Error())
	}

	return &solutions.Value[0], nil
}

// GetSolutionById reads a solution by its solution id.
func (client *Client) GetSolutionById(ctx context.Context, environmentHost, solutionId string) (*SolutionDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/solutions",
	}
	values := url.Values{}
	values.Add("$expand", "publisherid")
	values.Add("$filter", fmt.Sprintf("solutionid eq %s", solutionId))
	apiUrl.RawQuery = values.Encode()

	solutions := SolutionArrayDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &solutions)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	if len(solutions.Value) == 0 {
		baseErr := fmt.Errorf("solution with id '%s' not found", solutionId)
		return nil, customerrors.WrapIntoProviderError(baseErr, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), baseErr.Error())
	}

	return &solutions.Value[0], nil
}

// GetSolutions lists the visible solutions of an environment.
func (client *Client) GetSolutions(ctx context.Context, environmentHost string) ([]SolutionDto, error) {
	return client.listSolutions(ctx, environmentHost, "(isvisible eq true)")
}

// GetInstalledSolutions lists every solution installed in an environment.
func (client *Client) GetInstalledSolutions(ctx context.Context, environmentHost string) ([]SolutionDto, error) {
	return client.listSolutions(ctx, environmentHost, "")
}

func (client *Client) listSolutions(ctx context.Context, environmentHost, filter string) ([]SolutionDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/solutions",
	}
	values := url.Values{}
	values.Add("$expand", "publisherid")
	if filter != "" {
		values.Add("$filter", filter)
	}
	values.Add("$orderby", "createdon desc")
	apiUrl.RawQuery = values.Encode()

	solutionArray := SolutionArrayDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &solutionArray)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	solutions := make([]SolutionDto, 0)
	solutions = append(solutions, solutionArray.Value...)

	return solutions, nil
}

// CreateSolution stages and imports a solution package, waits for the import
// to complete, and reads back the imported solution.
func (client *Client) CreateSolution(ctx context.Context, environmentHost string, content []byte, settings []byte) (*SolutionDto, error) {
	if content == nil {
		return nil, errors.New("solution content is nil")
	}

	stageSolutionRequestBody := StageSolutionImportDto{
		CustomizationFile: base64.StdEncoding.EncodeToString(content),
	}

	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/StageSolution",
	}

	stageSolutionResponse := StageSolutionImportResponseDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", apiUrl.String(), nil, stageSolutionRequestBody, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &stageSolutionResponse)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	if stageSolutionResponse.StageSolutionResults.StageSolutionStatus != "Passed" {
		e := fmt.Errorf("solution failed with status: '%s'", stageSolutionResponse.StageSolutionResults.StageSolutionStatus)

		for _, missingDependency := range stageSolutionResponse.StageSolutionResults.MissingDependencies {
			e = errors.Join(fmt.Errorf("missing dependency: '%s'", missingDependency.RequiredComponentSchemaName), e)
		}
		for _, validation := range stageSolutionResponse.StageSolutionResults.SolutionValidationResults {
			e = errors.Join(fmt.Errorf("solution validation failed: %s", validation.Message), e)
		}
		return nil, e
	}

	solutionComponents, err := createSolutionComponentParameters(settings)
	if err != nil {
		return nil, err
	}

	importSolutionRequestBody := ImportSolutionDto{
		PublishWorkflows:                 true,
		OverwriteUnmanagedCustomizations: false,
		ComponentParameters:              solutionComponents,
		SolutionParameters: ImportSolutionSolutionParametersDto{
			StageSolutionUploadId: stageSolutionResponse.StageSolutionResults.StageSolutionUploadId,
		},
	}

	apiUrl = &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/ImportSolutionAsync",
	}
	importSolutionResponse := ImportSolutionResponseDto{}
	_, err = client.Api.Execute(ctx, scopes(environmentHost), "POST", apiUrl.String(), nil, importSolutionRequestBody, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &importSolutionResponse)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	// pull for solution import completion.
	if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
		return nil, err
	}

	apiUrl = &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/v9.2/asyncoperations(%s)", importSolutionResponse.AsyncOperationId),
	}
	for {
		asyncSolutionPullResponse := AsyncSolutionPullResponseDto{}
		_, err = client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &asyncSolutionPullResponse)
		if err != nil {
			return nil, err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return nil, err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return nil, err
		}
		if asyncSolutionPullResponse.CompletedOn != "" {
			err = client.validateSolutionImportResult(ctx, environmentHost, importSolutionResponse.ImportJobKey)
			if err != nil {
				return nil, err
			}
			solution, err := client.GetSolutionUniqueName(ctx, environmentHost, stageSolutionResponse.StageSolutionResults.SolutionDetails.SolutionUniqueName)
			if err != nil {
				return nil, err
			}
			return solution, nil
		}
		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return nil, err
		}
	}
}

func createSolutionComponentParameters(settings []byte) ([]any, error) {
	if len(settings) == 0 {
		return nil, nil
	}

	solutionSettings := SolutionSettingsDto{}
	if settings != nil {
		err := json.Unmarshal(settings, &solutionSettings)
		if err != nil {
			return nil, err
		}
	}

	solutionComponents := make([]any, 0)
	for _, connectionReferenceComponent := range solutionSettings.ConnectionReferences {
		solutionComponents = append(solutionComponents, ImportSolutionConnectionReferencesDto{
			Type:                           "Microsoft.Dynamics.CRM.connectionreference",
			ConnectionId:                   connectionReferenceComponent.ConnectionId,
			ConnectorId:                    connectionReferenceComponent.ConnectorId,
			ConnectionReferenceLogicalName: connectionReferenceComponent.LogicalName,
			ConnectionReferenceDisplayName: "",
			Description:                    "",
		})
	}
	for _, envVariableComponent := range solutionSettings.EnvironmentVariables {
		if envVariableComponent.Value != "" {
			solutionComponents = append(solutionComponents, ImportSolutionEnvironmentVariablesDto{
				Type:       "Microsoft.Dynamics.CRM.environmentvariablevalue",
				SchemaName: envVariableComponent.SchemaName,
				Value:      envVariableComponent.Value,
			})
		}
	}

	if len(solutionComponents) == 0 {
		return nil, nil
	}
	return solutionComponents, nil
}

func (client *Client) validateSolutionImportResult(ctx context.Context, environmentHost, importJobKey string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/v9.0/RetrieveSolutionImportResult(ImportJobId=%s)", importJobKey),
	}

	validateSolutionImportResponse := ValidateSolutionImportResponseDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &validateSolutionImportResponse)
	if err != nil {
		return err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return err
	}
	if validateSolutionImportResponse.SolutionOperationResult.Status != "Passed" {
		return fmt.Errorf("solution import failed: %s", validateSolutionImportResponse.SolutionOperationResult.ErrorMessages...)
	}
	return nil
}

// DeleteSolution deletes a solution by its solution id.
func (client *Client) DeleteSolution(ctx context.Context, environmentHost, solutionId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/v9.2/solutions(%s)", solutionId),
	}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return err
	} else if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	} else if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return err
	}
	return nil
}

// Environment variables are solution components: their definitions ship inside
// solution packages and their values are set through solution deployment
// settings, so their Dataverse Web API operations live with the solution
// domain.

// GetEnvironmentVariableDefinitionBySchemaName reads the single environment
// variable definition with the given schema name.
func (client *Client) GetEnvironmentVariableDefinitionBySchemaName(ctx context.Context, environmentHost, schemaName string) (*EnvironmentVariableDefinitionDto, error) {
	values := url.Values{}
	values.Add("$select", "environmentvariabledefinitionid,schemaname,displayname,description,defaultvalue,type,valueschema,secretstore")
	values.Add("$filter", fmt.Sprintf("schemaname eq '%s'", escapeODataStringLiteral(schemaName)))

	apiURL := helpers.BuildApiUrl(environmentHost, fmt.Sprintf("/api/data/%s/environmentvariabledefinitions", constants.DATAVERSE_API_VERSION), values)

	response := EnvironmentVariableDefinitionArrayDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiURL, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	switch len(response.Value) {
	case 0:
		return nil, customerrors.WrapIntoProviderError(
			fmt.Errorf("environment variable definition with schema name '%s' not found", schemaName),
			customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND),
			fmt.Sprintf("environment variable definition with schema name '%s' not found", schemaName),
		)
	case 1:
		return &response.Value[0], nil
	default:
		return nil, fmt.Errorf("multiple environment variable definitions found for schema name '%s'", schemaName)
	}
}

// GetEnvironmentVariableValueByDefinitionId reads the current value record of
// an environment variable definition, or nil when no value is set.
func (client *Client) GetEnvironmentVariableValueByDefinitionId(ctx context.Context, environmentHost, definitionID string) (*EnvironmentVariableValueDto, error) {
	values := url.Values{}
	values.Add("$select", "environmentvariablevalueid,schemaname,value")
	values.Add("$filter", fmt.Sprintf("_environmentvariabledefinitionid_value eq %s", definitionID))

	apiURL := helpers.BuildApiUrl(environmentHost, fmt.Sprintf("/api/data/%s/environmentvariablevalues", constants.DATAVERSE_API_VERSION), values)

	response := EnvironmentVariableValueArrayDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiURL, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	switch len(response.Value) {
	case 0:
		return nil, nil
	case 1:
		return &response.Value[0], nil
	default:
		return nil, fmt.Errorf("multiple current values found for environment variable definition '%s'", definitionID)
	}
}

// CreateEnvironmentVariableValue creates the current value record of an
// environment variable definition and returns the id Dataverse assigned to it.
func (client *Client) CreateEnvironmentVariableValue(ctx context.Context, environmentHost, definitionID, schemaName, value string) (string, error) {
	requestBody := CreateEnvironmentVariableValueDto{
		SchemaName:                            schemaName,
		Value:                                 value,
		EnvironmentVariableDefinitionODataRef: fmt.Sprintf("/environmentvariabledefinitions(%s)", definitionID),
	}

	apiURL := helpers.BuildApiUrl(environmentHost, fmt.Sprintf("/api/data/%s/environmentvariablevalues", constants.DATAVERSE_API_VERSION), nil)
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", apiURL, nil, requestBody, []int{http.StatusCreated, http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return "", err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return "", err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return "", err
	}

	entityID := extractEntityIDFromResponse(resp)
	if entityID == "" {
		return "", fmt.Errorf("no environment variable value id returned after creating schema name '%s'", schemaName)
	}

	return entityID, nil
}

// UpdateEnvironmentVariableValue updates an existing environment variable
// value record.
func (client *Client) UpdateEnvironmentVariableValue(ctx context.Context, environmentHost, valueID, value string) error {
	requestBody := UpdateEnvironmentVariableValueDto{
		Value: value,
	}

	apiURL := helpers.BuildApiUrl(environmentHost, fmt.Sprintf("/api/data/%s/environmentvariablevalues(%s)", constants.DATAVERSE_API_VERSION, valueID), nil)
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "PATCH", apiURL, nil, requestBody, []int{http.StatusNoContent, http.StatusOK, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	}
	return client.Api.HandleNotFoundResponse(resp)
}

// GetEnvironmentVariableValueById reads an environment variable value record
// by its id.
func (client *Client) GetEnvironmentVariableValueById(ctx context.Context, environmentHost, valueID string) (*EnvironmentVariableValueDto, error) {
	values := url.Values{}
	values.Add("$select", "environmentvariablevalueid,schemaname,value")
	apiURL := helpers.BuildApiUrl(environmentHost, fmt.Sprintf("/api/data/%s/environmentvariablevalues(%s)", constants.DATAVERSE_API_VERSION, valueID), values)

	response := EnvironmentVariableValueDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiURL, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	return &response, nil
}

// DeleteEnvironmentVariableValue deletes an environment variable value record.
// A value that is already gone (HTTP 404) is not an error.
func (client *Client) DeleteEnvironmentVariableValue(ctx context.Context, environmentHost, valueID string) error {
	apiURL := helpers.BuildApiUrl(environmentHost, fmt.Sprintf("/api/data/%s/environmentvariablevalues(%s)", constants.DATAVERSE_API_VERSION, valueID), nil)
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiURL, nil, nil, []int{http.StatusNoContent, http.StatusNotFound, http.StatusForbidden}, nil)
	if err != nil {
		return err
	}
	return client.Api.HandleForbiddenResponse(resp)
}

func escapeODataStringLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func extractEntityIDFromResponse(resp *api.Response) string {
	if resp == nil || resp.HttpResponse == nil {
		return ""
	}

	entityIDHeader := resp.HttpResponse.Header.Get(constants.HEADER_ODATA_ENTITY_ID)
	if entityIDHeader == "" {
		entityIDHeader = resp.HttpResponse.Header.Get(constants.HEADER_LOCATION)
	}
	if entityIDHeader == "" {
		return ""
	}

	re := regexp.MustCompile("[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	matches := re.FindAllString(entityIDHeader, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}
