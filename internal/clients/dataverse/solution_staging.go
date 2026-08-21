// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// This file is the staged managed-solution import machinery: stage a solution
// package, start the import, wait for the async operation, validate the import
// job result, and publish customizations. It is deliberately not unified with
// CreateSolution in solutions.go: starting an import here is non-idempotent
// and must never be replayed (ExecuteWithoutRetry), the import request carries
// SkipProductUpdateDependencies and overwrites unmanaged customizations, omits
// empty component parameters, and supports the StageAndUpgradeAsync operation.

// StagedImportOperation selects the Dataverse import API used to realize a
// staged managed solution: initial installation or in-place staged upgrade of
// an existing managed install.
type StagedImportOperation string

const (
	StagedImportOperationInstall         StagedImportOperation = "ImportSolutionAsync"
	StagedImportOperationStageAndUpgrade StagedImportOperation = "StageAndUpgradeAsync"
)

// StageSolution stages a solution package for a later staged import and
// returns the staging results. A staging result that did not pass is returned
// as an error joining every missing dependency and validation failure.
func (client *Client) StageSolution(ctx context.Context, environmentHost string, content []byte) (*StageSolutionImportResponseDto, error) {
	if content == nil {
		return nil, errors.New("solution content is nil")
	}

	stageRequestBody := StageSolutionRequestDto{
		CustomizationFile: base64.StdEncoding.EncodeToString(content),
	}

	stageURL := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/StageSolution",
	}

	stageResponse := StageSolutionImportResponseDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", stageURL.String(), nil, stageRequestBody, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &stageResponse)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	if stageResponse.StageSolutionResults.StageSolutionStatus != "Passed" {
		e := fmt.Errorf("solution failed with status: '%s'", stageResponse.StageSolutionResults.StageSolutionStatus)
		for _, missingDependency := range stageResponse.StageSolutionResults.MissingDependencies {
			e = errors.Join(fmt.Errorf("missing dependency: '%s'", missingDependency.RequiredComponentSchemaName), e)
		}
		for _, validation := range stageResponse.StageSolutionResults.SolutionValidationResults {
			e = errors.Join(fmt.Errorf("solution validation failed: %s", validation.Message), e)
		}
		return nil, e
	}

	return &stageResponse, nil
}

// StartStagedSolutionImport starts the asynchronous import of a previously
// staged solution package. Starting an import is not idempotent, so an
// ambiguous transport failure must never replay the request: the operation
// executes exactly one attempt.
func (client *Client) StartStagedSolutionImport(ctx context.Context, environmentHost string, operation StagedImportOperation, stageSolutionUploadId string, componentParameters []any, skipProductUpdateDependencies bool) (*ImportSolutionResponseDto, error) {
	importRequestBody := StagedImportSolutionDto{
		PublishWorkflows:                 true,
		OverwriteUnmanagedCustomizations: true,
		SkipProductUpdateDependencies:    skipProductUpdateDependencies,
		ComponentParameters:              componentParameters,
		SolutionParameters: StagedImportSolutionParametersDto{
			StageSolutionUploadId: stageSolutionUploadId,
		},
	}

	importURL := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/" + string(operation),
	}
	importResponse := ImportSolutionResponseDto{}
	resp, err := client.Api.ExecuteWithoutRetry(ctx, scopes(environmentHost), "POST", importURL.String(), nil, importRequestBody, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &importResponse)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	return &importResponse, nil
}

// AwaitStagedSolutionImport polls the async operation of a started staged
// solution import until it has completed.
func (client *Client) AwaitStagedSolutionImport(ctx context.Context, environmentHost, asyncOperationId string) error {
	if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
		return err
	}

	pollURL := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/v9.2/asyncoperations(%s)", asyncOperationId),
	}

	for {
		asyncResponse := AsyncSolutionPullResponseDto{}
		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", pollURL.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &asyncResponse)
		if err != nil {
			return err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return err
		}

		if asyncResponse.CompletedOn != "" {
			return nil
		}

		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return err
		}
	}
}

// ValidateStagedSolutionImportResult reads the import job result of a staged
// solution import and returns an error joining the job's error messages unless
// the import passed.
func (client *Client) ValidateStagedSolutionImportResult(ctx context.Context, environmentHost, importJobKey string) error {
	apiURL := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/v9.0/RetrieveSolutionImportResult(ImportJobId=%s)", importJobKey),
	}

	response := ValidateSolutionImportResponseDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiURL.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return err
	}
	if response.SolutionOperationResult.Status != "Passed" {
		messages := make([]string, 0, len(response.SolutionOperationResult.ErrorMessages))
		for _, message := range response.SolutionOperationResult.ErrorMessages {
			messages = append(messages, fmt.Sprint(message))
		}
		return fmt.Errorf("solution import failed: %s", strings.Join(messages, "; "))
	}
	return nil
}

// PublishAllCustomizations publishes all customizations of an environment.
func (client *Client) PublishAllCustomizations(ctx context.Context, environmentHost string) error {
	publishURL := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   "/api/data/v9.2/PublishAllXml",
	}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", publishURL.String(), nil, struct{}{}, []int{http.StatusOK, http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	}
	return client.Api.HandleNotFoundResponse(resp)
}
