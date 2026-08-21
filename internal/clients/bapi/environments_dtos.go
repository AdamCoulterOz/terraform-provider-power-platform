// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

type EnvironmentDto struct {
	Id         string                   `json:"id,omitempty"`
	Type       string                   `json:"type,omitempty"`
	Location   string                   `json:"location,omitempty"`
	Name       string                   `json:"name,omitempty"`
	Properties *EnviromentPropertiesDto `json:"properties"`
}

type EnviromentPropertiesDto struct {
	AzureRegion               string                            `json:"azureRegion,omitempty"`
	DatabaseType              string                            `json:"databaseType,omitempty"`
	DisplayName               string                            `json:"displayName,omitempty"`
	EnvironmentSku            string                            `json:"environmentSku,omitempty"`
	LinkedAppMetadata         *LinkedAppMetadataDto             `json:"linkedAppMetadata,omitempty"`
	RuntimeEndpoints          *RuntimeEndpointsDto              `json:"runtimeEndpoints,omitempty"`
	LinkedEnvironmentMetadata *LinkedEnvironmentMetadataDto     `json:"linkedEnvironmentMetadata,omitempty"`
	States                    *StatesEnvironmentDto             `json:"states,omitempty"`
	TenantId                  string                            `json:"tenantId,omitempty"`
	GovernanceConfiguration   *GovernanceConfigurationDto       `json:"governanceConfiguration,omitempty"`
	BillingPolicy             *BillingPolicyDto                 `json:"billingPolicy,omitempty"`
	ProvisioningState         string                            `json:"provisioningState,omitempty"`
	Description               string                            `json:"description"`
	UpdateCadence             *UpdateCadenceDto                 `json:"updateCadence,omitempty"`
	ParentEnvironmentGroup    *ParentEnvironmentGroupDto        `json:"parentEnvironmentGroup,omitempty"`
	EnterprisePolicies        *EnvironmentEnterprisePoliciesDto `json:"enterprisePolicies,omitempty"`
	Cluster                   *ClusterDto                       `json:"cluster,omitempty"`
	UsedBy                    *UsedByDto                        `json:"usedBy,omitempty"`
	BingChatEnabled           bool                              `json:"bingChatEnabled,omitempty"`
	M365Enabled               bool                              `json:"m365Enabled,omitempty"`
	CopilotPolicies           *CopilotPoliciesDto               `json:"copilotPolicies,omitempty"`
}

type GenerativeAiFeaturesDto struct {
	Properties GenerativeAiFeaturesPropertiesDto `json:"properties"`
}

type GenerativeAiFeaturesPropertiesDto struct {
	CopilotPolicies *CopilotPoliciesDto `json:"copilotPolicies,omitempty"`
	BingChatEnabled *bool               `json:"bingChatEnabled,omitempty"`
	M365Enabled     *bool               `json:"m365Enabled,omitempty"`
}

type CopilotPoliciesDto struct {
	CrossGeoCopilotDataMovementEnabled      *bool `json:"crossGeoCopilotDataMovementEnabled,omitempty"`
	CrossBoundaryCopilotDataMovementEnabled *bool `json:"crossBoundaryCopilotDataMovementEnabled,omitempty"`
}

type ClusterDto struct {
	Catergory string `json:"category"`
}

type UsedByDto struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	TenantId string `json:"tenantId"`
}

type EnvironmentEnterprisePoliciesDto struct {
	Vnets               *EnterprisePolicyDto `json:"vnets,omitempty"`
	CustomerManagedKeys *EnterprisePolicyDto `json:"customerManagedKeys,omitempty"`
	Identity            *EnterprisePolicyDto `json:"identity,omitempty"`
}

type EnterprisePolicyDto struct {
	PolicyId   string `json:"policyId"`
	Location   string `json:"location"`
	Id         string `json:"id"`
	SystemId   string `json:"systemId"`
	LinkStatus string `json:"linkStatus"`
}

type ParentEnvironmentGroupDto struct {
	Id string `json:"id"`
}

type UpdateCadenceDto struct {
	Id string `json:"id"`
}

type BillingPolicyDto struct {
	Id string `json:"id"`
}

type GovernanceConfigurationDto struct {
	ProtectionLevel string       `json:"protectionLevel,omitempty"`
	Settings        *SettingsDto `json:"settings,omitempty"`
}

type SettingsDto struct {
	ExtendedSettings ExtendedSettingsDto `json:"extendedSettings"`
}

type ExtendedSettingsDto struct {
	ExcludeEnvironmentFromAnalysis string `json:"excludeEnvironmentFromAnalysis"`
	IsGroupSharingDisabled         string `json:"isGroupSharingDisabled"`
	MaxLimitUserSharing            string `json:"maxLimitUserSharing"`
	DisableAiGeneratedDescriptions string `json:"disableAiGeneratedDescriptions"`
	IncludeOnHomepageInsights      string `json:"includeOnHomepageInsights"`
	LimitSharingMode               string `json:"limitSharingMode"`
	SolutionCheckerMode            string `json:"solutionCheckerMode"`
	SuppressValidationEmails       string `json:"suppressValidationEmails"`
	SolutionCheckerRuleOverrides   string `json:"solutionCheckerRuleOverrides"`

	SolutionCloudFlowsLimitSharingMode *string `json:"solutionCloudFlows-limitSharingMode"`
	// not exposed in PPAC UI yet, always set to -1
	SolutionCloudFlowsMaxLimitUserSharing *string `json:"solutionCloudFlows-maxLimitUserSharing"`
	BotAuthoringSharingDisabled           *string `json:"bot-authoringSharingDisabled"`
	BotLimitSharingMode                   *string `json:"bot-limitSharingMode"`
	BotMaxLimitUserSharing                *string `json:"bot-maxLimitUserSharing"`
}

type LinkedEnvironmentMetadataDto struct {
	BackgroundOperationsState string                     `json:"backgroundOperationsState,omitempty"`
	DomainName                string                     `json:"domainName,omitempty"`
	InstanceURL               string                     `json:"instanceUrl,omitempty"`
	BaseLanguage              int                        `json:"baseLanguage,omitempty"`
	SecurityGroupId           string                     `json:"securityGroupId,omitempty"`
	ResourceId                string                     `json:"resourceId,omitempty"`
	Version                   string                     `json:"version,omitempty"`
	Templates                 []string                   `json:"template,omitempty"`
	TemplateMetadata          *CreateTemplateMetadataDto `json:"templateMetadata,omitempty"`
	UniqueName                string                     `json:"uniqueName,omitempty"`
}

type RuntimeEndpointsDto struct {
	BusinessAppPlatform string `json:"microsoft.BusinessAppPlatform"`
	CommonDataModel     string `json:"microsoft.CommonDataModel"`
	PowerApps           string `json:"microsoft.PowerApps"`
	PowerAppsAdvisor    string `json:"microsoft.PowerAppsAdvisor"`
	PowerVirtualAgents  string `json:"microsoft.PowerVirtualAgents"`
	ApiManagement       string `json:"microsoft.ApiManagement"`
	Flow                string `json:"microsoft.Flow"`
}

type LinkedAppMetadataDto struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Url  string `json:"url"`
}

type StatesEnvironmentDto struct {
	Management       *StatesManagementEnvironmentDto `json:"management,omitempty"`
	Runtime          *RuntimeEnvironmentDto          `json:"runtime,omitempty"`
	DisasterRecovery *DisasterRecoveryStateDto       `json:"disasterRecovery,omitempty"`
}

type DisasterRecoveryStateDto struct {
	Id string `json:"id,omitempty"`
}

type RuntimeEnvironmentDto struct {
	Id string `json:"id,omitempty"`
}

type StatesManagementEnvironmentDto struct {
	Id string `json:"id,omitempty"`
}

type EnvironmentArrayDto struct {
	Value []EnvironmentDto `json:"value"`
}

type EnvironmentCreateDto struct {
	Location   string                         `json:"location"`
	Properties EnvironmentCreatePropertiesDto `json:"properties"`
}

type ModifySkuDto struct {
	EnvironmentSku string `json:"environmentSku,omitempty"`
}

type EnvironmentCreatePropertiesDto struct {
	AzureRegion               string                            `json:"azureRegion,omitempty"`
	BillingPolicy             BillingPolicyDto                  `json:"billingPolicy,omitempty"`
	DataBaseType              string                            `json:"databaseType,omitempty"`
	DisplayName               string                            `json:"displayName,omitempty"`
	Description               string                            `json:"description,omitempty"`
	UpdateCadence             *UpdateCadenceDto                 `json:"updateCadence,omitempty"`
	EnvironmentSku            string                            `json:"environmentSku,omitempty"`
	LinkedEnvironmentMetadata *CreateLinkEnvironmentMetadataDto `json:"linkedEnvironmentMetadata,omitempty"`
	ParentEnvironmentGroup    *ParentEnvironmentGroupDto        `json:"parentEnvironmentGroup,omitempty"`
	BingChatEnabled           bool                              `json:"bingChatEnabled,omitempty"`
	M365Enabled               bool                              `json:"m365Enabled,omitempty"`
	CopilotPolicies           *CopilotPoliciesDto               `json:"copilotPolicies,omitempty"`
	UsedBy                    *UsedByDto                        `json:"usedBy,omitempty"`
	Cluster                   *ClusterDto                       `json:"cluster,omitempty"`
}

type CreateLinkEnvironmentMetadataDto struct {
	BaseLanguage     int                        `json:"baseLanguage,omitempty"`
	DomainName       string                     `json:"domainName,omitempty"`
	Currency         *CreateCurrencyDto         `json:"currency,omitempty"`
	SecurityGroupId  string                     `json:"securityGroupId,omitempty"`
	Templates        []string                   `json:"templates,omitempty"`
	TemplateMetadata *CreateTemplateMetadataDto `json:"templateMetadata,omitempty"`
}
type CreateCurrencyDto struct {
	Code string `json:"code,omitempty"`
}

type CreateTemplateMetadataDto struct {
	PostProvisioningPackages []CreatePostProvisioningPackagesDto `json:"PostProvisioningPackages,omitempty"`
}

type CreatePostProvisioningPackagesDto struct {
	ApplicationUniqueName string `json:"applicationUniqueName,omitempty"`
	Parameters            string `json:"parameters,omitempty"`
}

type EnvironmentDeleteDto struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LifecycleCreatedDto struct {
	Name       string                        `json:"name"`
	Properties LifecycleCreatedPropertiesDto `json:"properties"`
}

type LifecycleCreatedPropertiesDto struct {
	ProvisioningState string `json:"provisioningState"`
}

type ValidateCreateEnvironmentDetailsDto struct {
	DomainName          string `json:"domainName"`
	EnvironmentLocation string `json:"environmentLocation"`
}

type ValidateUpdateEnvironmentDetailsDto struct {
	DomainName      string `json:"domainName"`
	EnvironmentName string `json:"environmentName"`
}
