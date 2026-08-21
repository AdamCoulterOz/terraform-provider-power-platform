// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"time"
)

// SolutionSettingsDto is the deployment settings file shape consumed by
// CreateSolution: the connection references and environment variable values
// applied as component parameters of a solution import.
type SolutionSettingsDto struct {
	EnvironmentVariables []SettingsEnvironmentVariableDto  `json:"environmentvariables"`
	ConnectionReferences []SettingsConnectionReferencesDto `json:"connectionreferences"`
}

type SettingsEnvironmentVariableDto struct {
	SchemaName string `json:"schemaname"`
	Value      string `json:"value"`
}

type SettingsConnectionReferencesDto struct {
	LogicalName  string `json:"logicalname"`
	ConnectionId string `json:"connectionid"`
	ConnectorId  string `json:"connectorid"`
}

// SolutionDto is a row of the solutions table. EnvironmentId is not part of
// the wire shape; services stamp it after resolving which environment the
// solution was read from.
type SolutionDto struct {
	Id            string `json:"solutionid"`
	EnvironmentId string `json:"environment_id"`
	Name          string `json:"uniquename"`
	DisplayName   string `json:"friendlyname"`
	IsManaged     bool   `json:"ismanaged"`
	CreatedTime   string `json:"createdon"`
	Version       string `json:"version"`
	ModifiedTime  string `json:"modifiedon"`
	InstallTime   string `json:"installedon"`
}

type SolutionArrayDto struct {
	Value []SolutionDto `json:"value"`
}

type StageSolutionImportDto struct {
	CustomizationFile string `json:"CustomizationFile"`
}

type StageSolutionImportResponseDto struct {
	StageSolutionResults StageSolutionImportResultResponseDto `json:"StageSolutionResults"`
}

type StageSolutionImportResultResponseDto struct {
	StageSolutionUploadId     string                          `json:"StageSolutionUploadId"`
	StageSolutionStatus       string                          `json:"StageSolutionStatus"`
	SolutionValidationResults []ValidationResultsDto          `json:"SolutionValidationResults"`
	MissingDependencies       []MissingDependenciesDto        `json:"MissingDependencies"`
	SolutionDetails           StageSolutionSolutionDetailsDto `json:"SolutionDetails"`
}

type ValidationResultsDto struct {
	SolutionValidationResultType string `json:"SolutionValidationResultType"`
	ErrorCode                    int    `json:"ErrorCode"`
	AdditionalInfo               string `json:"AdditionalInfo"`
	Message                      string `json:"Message"`
}

type MissingDependenciesDto struct {
	RequiredComponentSchemaName         string `json:"RequiredComponentSchemaName"`
	RequiredComponentDisplayName        string `json:"RequiredComponentDisplayName"`
	RequiredComponentParentSchemaName   string `json:"RequiredComponentParentSchemaName"`
	RequiredComponentParentDisplayName  string `json:"RequiredComponentParentDisplayName"`
	RequiredComponentId                 string `json:"RequiredComponentId"`
	RequiredSolutionName                string `json:"RequiredSolutionName"`
	RequiredComponentType               string `json:"RequiredComponentType"`
	DependentComponentSchemaName        string `json:"DependentComponentSchemaName"`
	DependentComponentDisplayName       string `json:"DependentComponentDisplayName"`
	DependentComponentParentSchemaName  string `json:"DependentComponentParentSchemaName"`
	DependentComponentParentDisplayName string `json:"DependentComponentParentDisplayName"`
	DependentComponentType              string `json:"DependentComponentType"`
	DependentComponentId                string `json:"DependentComponentId"`
}

type StageSolutionSolutionDetailsDto struct {
	SolutionUniqueName   string `json:"SolutionUniqueName"`
	SolutionFriendlyName string `json:"SolutionFriendlyName"`
	IsManaged            bool   `json:"IsManaged"`
	SolutionVersion      string `json:"SolutionVersion"`
}

type ImportSolutionDto struct {
	PublishWorkflows                 bool                                `json:"PublishWorkflows"`
	OverwriteUnmanagedCustomizations bool                                `json:"OverwriteUnmanagedCustomizations"`
	ComponentParameters              []any                               `json:"ComponentParameters"`
	SolutionParameters               ImportSolutionSolutionParametersDto `json:"SolutionParameters"`
}

type ImportSolutionResponseDto struct {
	ImportJobKey     string `json:"ImportJobKey"`
	AsyncOperationId string `json:"AsyncOperationId"`
}

type ImportSolutionSolutionParametersDto struct {
	StageSolutionUploadId string `json:"StageSolutionUploadId"`
}

type ImportSolutionConnectionReferencesDto struct {
	Type                           string `json:"@odata.type"`
	ConnectionReferenceDisplayName string `json:"connectionreferencedisplayname"`
	ConnectionReferenceLogicalName string `json:"connectionreferencelogicalname"`
	Description                    string `json:"description"`
	ConnectorId                    string `json:"connectorid"`
	ConnectionId                   string `json:"connectionid"`
}

type ImportSolutionEnvironmentVariablesDto struct {
	Type       string `json:"@odata.type"`
	SchemaName string `json:"schemaname"`
	Value      string `json:"value"`
}

type AsyncSolutionPullResponseDto struct {
	AsyncOperationId string `json:"AsyncOperationId"`
	CreatedOn        string `json:"createdon"`
	CompletedOn      string `json:"completedon"`
}

type ValidateSolutionImportResponseDto struct {
	SolutionOperationResult ValidateSolutionImportResponseSolutionOperationResultDto `json:"SolutionOperationResult"`
}

type ValidateSolutionImportResponseSolutionOperationResultDto struct {
	Status          string `json:"Status"`
	WarningMessages []any  `json:"WarningMessages"`
	ErrorMessages   []any  `json:"ErrorMessages"`
}

type EnvironmentVariableDefinitionArrayDto struct {
	Value []EnvironmentVariableDefinitionDto `json:"value"`
}

type EnvironmentVariableDefinitionDto struct {
	EnvironmentVariableDefinitionId string `json:"environmentvariabledefinitionid"`
	SchemaName                      string `json:"schemaname"`
	DisplayName                     string `json:"displayname"`
	Description                     string `json:"description"`
	DefaultValue                    string `json:"defaultvalue"`
	Type                            int64  `json:"type"`
	ValueSchema                     string `json:"valueschema"`
	SecretStore                     int64  `json:"secretstore"`
}

type EnvironmentVariableValueArrayDto struct {
	Value []EnvironmentVariableValueDto `json:"value"`
}

type EnvironmentVariableValueDto struct {
	EnvironmentVariableValueId string `json:"environmentvariablevalueid"`
	SchemaName                 string `json:"schemaname"`
	Value                      string `json:"value"`
}

type CreateEnvironmentVariableValueDto struct {
	SchemaName                            string `json:"schemaname"`
	Value                                 string `json:"value"`
	EnvironmentVariableDefinitionODataRef string `json:"environmentvariabledefinitionid@odata.bind"`
}

type UpdateEnvironmentVariableValueDto struct {
	Value string `json:"value"`
}

type OrganizationSettingsArrayDto struct {
	Value []OrganizationSettingsDto `json:"value"`
}

type OrganizationSettingsDto struct {
	ODataEtag      string    `json:"@odata.etag"`
	CreatedOn      time.Time `json:"createdon"`
	BaseCurrencyId string    `json:"_basecurrencyid_value"`
}

type TransactionCurrencyDto struct {
	OrganizationValue     string  `json:"_organizationid_value"`
	CurrencyName          string  `json:"currencyname"`
	CurrencySymbol        string  `json:"currencysymbol"`
	IsoCurrencyCode       string  `json:"isocurrencycode"`
	CreatedOn             string  `json:"createdon"`
	CurrencyPrecision     int     `json:"currencyprecision"`
	ExchangeRate          float32 `json:"exchangerate"`
	TransactionCurrencyId string  `json:"transactioncurrencyid"`
}

type TransactionCurrencyArrayDto struct {
	Value []TransactionCurrencyDto `json:"value"`
}
