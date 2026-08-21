// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

// Request wire shapes of the staged managed-solution import machinery in
// solution_staging.go. They stay separate from the CreateSolution request
// DTOs in solutions_dtos.go: the staged import body additionally carries
// SkipProductUpdateDependencies and omits empty component parameters, and the
// two machineries must be able to evolve independently. Responses reuse the
// canonical types in solutions_dtos.go (StageSolutionImportResponseDto,
// ImportSolutionResponseDto, AsyncSolutionPullResponseDto and
// ValidateSolutionImportResponseDto).

// StageSolutionRequestDto is the request body of StageSolution.
type StageSolutionRequestDto struct {
	CustomizationFile string `json:"CustomizationFile"`
}

// StagedImportSolutionDto is the request body of StartStagedSolutionImport.
type StagedImportSolutionDto struct {
	PublishWorkflows                 bool                              `json:"PublishWorkflows"`
	OverwriteUnmanagedCustomizations bool                              `json:"OverwriteUnmanagedCustomizations"`
	SkipProductUpdateDependencies    bool                              `json:"SkipProductUpdateDependencies"`
	ComponentParameters              []any                             `json:"ComponentParameters,omitempty"`
	SolutionParameters               StagedImportSolutionParametersDto `json:"SolutionParameters"`
}

type StagedImportSolutionParametersDto struct {
	StageSolutionUploadId string `json:"StageSolutionUploadId"`
}

// StagedImportConnectionReferenceDto is a connection reference entry of the
// staged import's component parameters.
type StagedImportConnectionReferenceDto struct {
	Type                           string `json:"@odata.type"`
	ConnectionReferenceDisplayName string `json:"connectionreferencedisplayname"`
	ConnectionReferenceLogicalName string `json:"connectionreferencelogicalname"`
	Description                    string `json:"description"`
	ConnectorId                    string `json:"connectorid"`
	ConnectionId                   string `json:"connectionid"`
}
