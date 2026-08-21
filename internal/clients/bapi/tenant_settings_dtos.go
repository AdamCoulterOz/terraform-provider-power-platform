// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

type PowerPlatformSettingsDto struct {
	Search                 *SearchSettingsDto               `json:"search,omitempty"`
	TeamsIntegration       *TeamsIntegrationSettingsDto     `json:"teamsIntegration,omitempty"`
	PowerApps              *PowerAppsSettingsDto            `json:"powerApps,omitempty"`
	PowerAutomate          *PowerAutomateSettingsDto        `json:"powerAutomate,omitempty"`
	Environments           *EnvironmentSettingsDto          `json:"environments,omitempty"`
	Governance             *GovernanceSettingsDto           `json:"governance,omitempty"`
	Licensing              *LicenseSettingsDto              `json:"licensing,omitempty"`
	PowerPages             *PowerPagesSettingsDto           `json:"powerPages,omitempty"`
	Champions              *ChampionSettingsDto             `json:"champions,omitempty"`
	Intelligence           *IntelligenceSettingsDto         `json:"intelligence,omitempty"`
	ModelExperimentation   *ModelExperimentationSettingsDto `json:"modelExperimentation,omitempty"`
	CatalogSettings        *CatalogSettingsDto              `json:"catalogSettings,omitempty"`
	UserManagementSettings *UserManagementSettingsDto       `json:"userManagementSettings,omitempty"`
}

type TeamsIntegrationSettingsDto struct {
	ShareWithColleaguesUserLimit *int64 `json:"shareWithColleaguesUserLimit,omitempty"`
}

type PowerAutomateSettingsDto struct {
	DisableCopilot *bool `json:"disableCopilot,omitempty"`
}

type PowerAppsSettingsDto struct {
	DisableShareWithEveryone             *bool `json:"disableShareWithEveryone,omitempty"`
	EnableGuestsToMake                   *bool `json:"enableGuestsToMake,omitempty"`
	DisableMakerMatch                    *bool `json:"disableMakerMatch,omitempty"`
	DisableUnusedLicenseAssignment       *bool `json:"disableUnusedLicenseAssignment,omitempty"`
	DisableCreateFromImage               *bool `json:"disableCreateFromImage,omitempty"`
	DisableCreateFromFigma               *bool `json:"disableCreateFromFigma,omitempty"`
	DisableConnectionSharingWithEveryone *bool `json:"disableConnectionSharingWithEveryone,omitempty"`
}

type EnvironmentSettingsDto struct {
	DisablePreferredDataLocationForTeamsEnvironment *bool `json:"disablePreferredDataLocationForTeamsEnvironment,omitempty"`
}

type SearchSettingsDto struct {
	DisableDocsSearch      *bool `json:"disableDocsSearch,omitempty"`
	DisableCommunitySearch *bool `json:"disableCommunitySearch,omitempty"`
	DisableBingVideoSearch *bool `json:"disableBingVideoSearch,omitempty"`
}

type GovernanceSettingsDto struct {
	DisableAdminDigest                                 *bool              `json:"disableAdminDigest,omitempty"`
	DisableDeveloperEnvironmentCreationByNonAdminUsers *bool              `json:"disableDeveloperEnvironmentCreationByNonAdminUsers,omitempty"`
	EnableDefaultEnvironmentRouting                    *bool              `json:"enableDefaultEnvironmentRouting,omitempty"`
	EnvironmentRoutingAllMakers                        *bool              `json:"environmentRoutingAllMakers,omitempty"`
	EnvironmentRoutingTargetEnvironmentGroupId         *string            `json:"environmentRoutingTargetEnvironmentGroupId,omitempty"`
	EnvironmentRoutingTargetSecurityGroupId            *string            `json:"environmentRoutingTargetSecurityGroupId,omitempty"`
	Policy                                             *PolicySettingsDto `json:"policy,omitempty"`
}

type PolicySettingsDto struct {
	EnableDesktopFlowDataPolicyManagement *bool `json:"enableDesktopFlowDataPolicyManagement,omitempty"`
}

type LicenseSettingsDto struct {
	DisableBillingPolicyCreationByNonAdminUsers     *bool  `json:"disableBillingPolicyCreationByNonAdminUsers,omitempty"`
	EnableTenantCapacityReportForEnvironmentAdmins  *bool  `json:"enableTenantCapacityReportForEnvironmentAdmins,omitempty"`
	StorageCapacityConsumptionWarningThreshold      *int64 `json:"storageCapacityConsumptionWarningThreshold,omitempty"`
	EnableTenantLicensingReportForEnvironmentAdmins *bool  `json:"enableTenantLicensingReportForEnvironmentAdmins,omitempty"`
	DisableUseOfUnassignedAIBuilderCredits          *bool  `json:"disableUseOfUnassignedAIBuilderCredits,omitempty"`
}

type PowerPagesSettingsDto struct {
}

type ChampionSettingsDto struct {
	DisableChampionsInvitationReachout   *bool `json:"disableChampionsInvitationReachout,omitempty"`
	DisableSkillsMatchInvitationReachout *bool `json:"disableSkillsMatchInvitationReachout,omitempty"`
}

type IntelligenceSettingsDto struct {
	DisableCopilot            *bool `json:"disableCopilot,omitempty"`
	EnableOpenAiBotPublishing *bool `json:"enableOpenAiBotPublishing,omitempty"`
}

type ModelExperimentationSettingsDto struct {
	EnableModelDataSharing *bool `json:"enableModelDataSharing,omitempty"`
	DisableDataLogging     *bool `json:"disableDataLogging,omitempty"`
}

type CatalogSettingsDto struct {
	PowerCatalogAudienceSetting *string `json:"powerCatalogAudienceSetting,omitempty"`
}

type UserManagementSettingsDto struct {
	EnableDeleteDisabledUserinAllEnvironments *bool `json:"enableDeleteDisabledUserinAllEnvironments,omitempty"`
}

type TenantSettingsDto struct {
	WalkMeOptOut                                   *bool                     `json:"walkMeOptOut,omitempty"`
	DisableNPSCommentsReachout                     *bool                     `json:"disableNPSCommentsReachout,omitempty"`
	DisableNewsletterSendout                       *bool                     `json:"disableNewsletterSendout,omitempty"`
	DisableEnvironmentCreationByNonAdminUsers      *bool                     `json:"disableEnvironmentCreationByNonAdminUsers,omitempty"`
	DisablePortalsCreationByNonAdminUsers          *bool                     `json:"disablePortalsCreationByNonAdminUsers,omitempty"`
	DisableSurveyFeedback                          *bool                     `json:"disableSurveyFeedback,omitempty"`
	DisableTrialEnvironmentCreationByNonAdminUsers *bool                     `json:"disableTrialEnvironmentCreationByNonAdminUsers,omitempty"`
	DisableCapacityAllocationByEnvironmentAdmins   *bool                     `json:"disableCapacityAllocationByEnvironmentAdmins,omitempty"`
	DisableSupportTicketsVisibleByAllUsers         *bool                     `json:"disableSupportTicketsVisibleByAllUsers,omitempty"`
	PowerPlatform                                  *PowerPlatformSettingsDto `json:"powerPlatform,omitempty"`
}
