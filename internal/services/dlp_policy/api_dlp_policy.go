// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dlp_policy

import (
	"context"
	"sort"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
)

func newDlpPolicyClient(apiClient *api.Client) client {
	return client{
		Client: bapi.NewClient(apiClient),
	}
}

// client is the DLP policy service client. BAPI DLP policy operations are
// promoted from the embedded bapi.Client; the wrappers below convert between
// the wire DTOs and the service's intermediate policy model.
type client struct {
	*bapi.Client
}

func (client *client) GetPolicies(ctx context.Context) ([]dlpPolicyModelDto, error) {
	policiesArray, err := client.Client.GetPolicies(ctx)
	if err != nil {
		return nil, err
	}

	policies := make([]dlpPolicyModelDto, 0)
	for _, policy := range policiesArray {
		v, err := covertDlpPolicyToPolicyModelDto(policy)
		if err != nil {
			return nil, err
		}
		policies = append(policies, *v)
	}
	return policies, nil
}

func (client *client) GetPolicy(ctx context.Context, name string) (*dlpPolicyModelDto, error) {
	policy, err := client.Client.GetPolicy(ctx, name)
	if err != nil {
		return nil, err
	}
	return covertDlpPolicyToPolicyModel(*policy)
}

func (client *client) UpdatePolicy(ctx context.Context, policy dlpPolicyModelDto) (*dlpPolicyModelDto, error) {
	policyToUpdate := convertPolicyModelToDlpPolicy(policy)

	updatedPolicy, err := client.Client.UpdatePolicy(ctx, policyToUpdate)
	if err != nil {
		return nil, err
	}

	return covertDlpPolicyToPolicyModel(*updatedPolicy)
}

func (client *client) CreatePolicy(ctx context.Context, policy dlpPolicyModelDto) (*dlpPolicyModelDto, error) {
	policyToCreate := convertPolicyModelToDlpPolicy(policy)

	createdPolicy, err := client.Client.CreatePolicy(ctx, policyToCreate)
	if err != nil {
		return nil, err
	}
	return covertDlpPolicyToPolicyModel(*createdPolicy)
}

func convertPolicyModelToDlpPolicy(policy dlpPolicyModelDto) bapi.DlpPolicyDto {
	policyToCreate := bapi.DlpPolicyDto{
		PolicyDefinition: bapi.DlpPolicyDefinitionDto{
			Name:                            policy.Name,
			DisplayName:                     policy.DisplayName,
			DefaultConnectorsClassification: policy.DefaultConnectorsClassification,
			EnvironmentType:                 policy.EnvironmentType,
			Environments:                    policy.Environments,
			ConnectorGroups:                 []bapi.DlpConnectorGroupsDto{},
		},
		ConnectorConfigurationsDefinition:    &policy.ConnectorConfigurationsDefinition,
		CustomConnectorUrlPatternsDefinition: bapi.DlpConnectorUrlPatternsDefinitionDto{},
	}

	for _, policy := range policy.CustomConnectorUrlPatternsDefinition {
		policyToCreate.CustomConnectorUrlPatternsDefinition.Rules = append(policyToCreate.CustomConnectorUrlPatternsDefinition.Rules, bapi.DlpConnectorUrlPatternsRuleDto{
			Order:                       policy.Rules[0].Order,
			ConnectorRuleClassification: policy.Rules[0].ConnectorRuleClassification,
			Pattern:                     policy.Rules[0].Pattern,
		})
	}

	for _, connGroups := range policy.ConnectorGroups {
		conG := bapi.DlpConnectorGroupsDto{
			Classification: connGroups.Classification,
			Connectors:     []bapi.DlpConnectorDto{},
		}

		for _, connector := range connGroups.Connectors {
			nameSplit := strings.Split(connector.Id, "/")
			con := bapi.DlpConnectorDto{
				Id:   connector.Id,
				Name: nameSplit[len(nameSplit)-1],
				Type: connector.Type,
			}
			conG.Connectors = append(conG.Connectors, con)
		}
		policyToCreate.PolicyDefinition.ConnectorGroups = append(policyToCreate.PolicyDefinition.ConnectorGroups, conG)
	}

	connectorActionConfigurationsDto := []bapi.DlpConnectorActionConfigurationsDto{}
	endpointConfigurationsDto := []bapi.DlpEndpointConfigurationsDto{}

	for _, connGroups := range policy.ConnectorGroups {
		for _, connector := range connGroups.Connectors {
			if len(connector.ActionRules) > 0 {
				connectorActionConfigurationsDto = append(connectorActionConfigurationsDto, bapi.DlpConnectorActionConfigurationsDto{
					ConnectorId:                        connector.Id,
					DefaultConnectorActionRuleBehavior: connector.DefaultActionRuleBehavior,
					ActionRules:                        connector.ActionRules,
				})
			}
			if len(connector.EndpointRules) > 0 {
				endpointConfigurationsDto = append(endpointConfigurationsDto, bapi.DlpEndpointConfigurationsDto{
					ConnectorId:   connector.Id,
					EndpointRules: connector.EndpointRules,
				})
			}
		}
	}

	if len(connectorActionConfigurationsDto) > 0 || len(endpointConfigurationsDto) > 0 {
		policyToCreate.ConnectorConfigurationsDefinition = &bapi.DlpConnectorConfigurationsDefinitionDto{}

		if len(connectorActionConfigurationsDto) > 0 {
			policyToCreate.ConnectorConfigurationsDefinition.ConnectorActionConfigurations = connectorActionConfigurationsDto
		}
		if len(endpointConfigurationsDto) > 0 {
			policyToCreate.ConnectorConfigurationsDefinition.EndpointConfigurations = endpointConfigurationsDto
		}
	} else {
		policyToCreate.ConnectorConfigurationsDefinition = nil
	}
	return policyToCreate
}

func covertDlpPolicyToPolicyModel(policy bapi.DlpPolicyDto) (*dlpPolicyModelDto, error) {
	policyModel := dlpPolicyModelDto{
		Name:                                 policy.PolicyDefinition.Name,
		DisplayName:                          policy.PolicyDefinition.DisplayName,
		EnvironmentType:                      policy.PolicyDefinition.EnvironmentType,
		Environments:                         policy.PolicyDefinition.Environments,
		ETag:                                 policy.PolicyDefinition.ETag,
		CreatedBy:                            policy.PolicyDefinition.CreatedBy.DisplayName,
		CreatedTime:                          policy.PolicyDefinition.CreatedTime,
		LastModifiedBy:                       policy.PolicyDefinition.LastModifiedBy.DisplayName,
		LastModifiedTime:                     policy.PolicyDefinition.LastModifiedTime,
		DefaultConnectorsClassification:      policy.PolicyDefinition.DefaultConnectorsClassification,
		ConnectorConfigurationsDefinition:    bapi.DlpConnectorConfigurationsDefinitionDto{},
		CustomConnectorUrlPatternsDefinition: []bapi.DlpConnectorUrlPatternsDefinitionDto{},
		ConnectorGroups:                      []dlpConnectorGroupsModelDto{},
	}

	for _, connGroup := range policy.PolicyDefinition.ConnectorGroups {
		connGroupModel := dlpConnectorGroupsModelDto{
			Classification: connGroup.Classification,
			Connectors:     []dlpConnectorModelDto{},
		}
		for _, connector := range connGroup.Connectors {
			nameSplit := strings.Split(connector.Id, "/")
			m := dlpConnectorModelDto{
				Id:   connector.Id,
				Name: nameSplit[len(nameSplit)-1],
				Type: connector.Type,
			}
			if policy.ConnectorConfigurationsDefinition != nil {
				for _, connectorActionConfigurations := range policy.ConnectorConfigurationsDefinition.ConnectorActionConfigurations {
					if connectorActionConfigurations.ConnectorId == connector.Id {
						m.DefaultActionRuleBehavior = connectorActionConfigurations.DefaultConnectorActionRuleBehavior
						m.ActionRules = connectorActionConfigurations.ActionRules
					}
				}
				for _, endpointConfigurations := range policy.ConnectorConfigurationsDefinition.EndpointConfigurations {
					if endpointConfigurations.ConnectorId == connector.Id {
						m.EndpointRules = endpointConfigurations.EndpointRules
					}
				}
			}
			connGroupModel.Connectors = append(connGroupModel.Connectors, m)
		}
		policyModel.ConnectorGroups = append(policyModel.ConnectorGroups, connGroupModel)
	}

	for _, rule := range policy.CustomConnectorUrlPatternsDefinition.Rules {
		policyModel.CustomConnectorUrlPatternsDefinition = append(policyModel.CustomConnectorUrlPatternsDefinition, bapi.DlpConnectorUrlPatternsDefinitionDto{
			Rules: append([]bapi.DlpConnectorUrlPatternsRuleDto{}, rule),
		})
	}

	sort.Slice(policyModel.CustomConnectorUrlPatternsDefinition, func(i, j int) bool {
		// Safely compare the Order of the first rule in each definition, handling empty Rules slices
		rulesI := policyModel.CustomConnectorUrlPatternsDefinition[i].Rules
		rulesJ := policyModel.CustomConnectorUrlPatternsDefinition[j].Rules
		if len(rulesI) == 0 && len(rulesJ) == 0 {
			return i < j // stable sort for empty slices
		}
		if len(rulesI) == 0 {
			return true // empty comes before non-empty
		}
		if len(rulesJ) == 0 {
			return false // non-empty comes after empty
		}
		return rulesI[0].Order < rulesJ[0].Order
	})

	return &policyModel, nil
}
