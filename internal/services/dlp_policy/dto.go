// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dlp_policy

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
)

// dlpPolicyModelDto is the service's intermediate policy model used for
// Terraform conversion. The wire DTOs live in internal/clients/bapi.
type dlpPolicyModelDto struct {
	Name                                 string                                       `json:"name"`
	DisplayName                          string                                       `json:"displayName"`
	DefaultConnectorsClassification      string                                       `json:"defaultConnectorsClassification"`
	EnvironmentType                      string                                       `json:"environmentType"`
	ETag                                 string                                       `json:"etag"`
	CreatedBy                            string                                       `json:"createdBy"`
	CreatedTime                          string                                       `json:"createdTime"`
	LastModifiedBy                       string                                       `json:"lastModifiedBy"`
	LastModifiedTime                     string                                       `json:"lastModifiedTime"`
	Environments                         []bapi.DlpEnvironmentDto                     `json:"environments"`
	ConnectorGroups                      []dlpConnectorGroupsModelDto                 `json:"connectorGroups"`
	CustomConnectorUrlPatternsDefinition []bapi.DlpConnectorUrlPatternsDefinitionDto  `json:"customConnectorUrlPatternsDefinition,omitempty"`
	ConnectorConfigurationsDefinition    bapi.DlpConnectorConfigurationsDefinitionDto `json:"connectorConfigurationsDefinition,omitempty"`
}

type dlpConnectorGroupsModelDto struct {
	Classification string                 `json:"classification"`
	Connectors     []dlpConnectorModelDto `json:"connectors"`
}

type dlpConnectorModelDto struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`

	DefaultActionRuleBehavior string
	ActionRules               []bapi.DlpActionRuleDto
	EndpointRules             []bapi.DlpEndpointRuleDto
}
