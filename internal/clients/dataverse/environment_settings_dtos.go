// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"fmt"
)

// EnvironmentBackendSettingsValueDto is the response of the Dataverse
// RetrieveSettingList() function.
type EnvironmentBackendSettingsValueDto struct {
	ODataContext            string                         `json:"@odata.context"`
	SettingDetailCollection []EnvironmentBackendSettingDto `json:"SettingDetailCollection"`
}

// EnvironmentBackendSettingDto is a single backend setting entry.
type EnvironmentBackendSettingDto struct {
	Name     string `json:"Name"`
	Value    string `json:"Value"`
	DataType int    `json:"DataType"`
}

// GetValue returns the value of the named setting, or an empty string when the
// setting is not present.
func (environmentBackendSettings EnvironmentBackendSettingsValueDto) GetValue(name string) string {
	for _, setting := range environmentBackendSettings.SettingDetailCollection {
		if setting.Name == name {
			return setting.Value
		}
	}
	return ""
}

// SetValue sets the named setting to the given string, bool or int value,
// appending a new entry when the setting is not present yet.
func (environmentBackendSettings *EnvironmentBackendSettingsValueDto) SetValue(name string, value any) error {
	val := ""
	var dataType int
	switch v := value.(type) {
	case string:
		dataType = 1
		val = v
	case bool:
		dataType = 2
		if v {
			val = "true"
		} else {
			val = "false"
		}
	case int:
		dataType = 0
		val = fmt.Sprintf("%d", v)
	default:
		return fmt.Errorf("unsupported value type: %T", v)
	}
	for i, setting := range environmentBackendSettings.SettingDetailCollection {
		if setting.Name == name {
			environmentBackendSettings.SettingDetailCollection[i].Value = val
			return nil
		}
	}

	// value not found in lets add it to the collection
	environmentBackendSettings.SettingDetailCollection = append(environmentBackendSettings.SettingDetailCollection, EnvironmentBackendSettingDto{
		Name:     name,
		Value:    val,
		DataType: dataType,
	})
	return nil
}

// EnvironmentOrgSettingsValueDto is the response of the Dataverse
// organizations collection endpoint.
type EnvironmentOrgSettingsValueDto struct {
	Value []EnvironmentOrgSettingsDto `json:"value"`
}

// EnvironmentOrgSettingsDto is an organization record's settings.
type EnvironmentOrgSettingsDto struct {
	MaxUploadFileSize                                    *int64  `json:"maxuploadfilesize,omitempty"`
	BlockedAttachments                                   *string `json:"blockedattachments,omitempty"`
	PluginTraceLogSetting                                *int64  `json:"plugintracelogsetting,omitempty"`
	IsAuditEnabled                                       *bool   `json:"isauditenabled,omitempty"`
	IsUserAccessAuditEnabled                             *bool   `json:"isuseraccessauditenabled,omitempty"`
	IsReadAuditEnabled                                   *bool   `json:"isreadauditenabled,omitempty"`
	AuditRetentionPeriodV2                               *int32  `json:"auditretentionperiodv2,omitempty"`
	BoundDashboardDefaultCardExpanded                    *bool   `json:"bounddashboarddefaultcardexpanded,omitempty"`
	OrganizationId                                       *string `json:"organizationid,omitempty"`
	PowerAppsComponentFrameworkForCanvasApps             *bool   `json:"iscustomcontrolsincanvasappsenabled,omitempty"`
	PowerAppsMakerBotEnabled                             *bool   `json:"powerappsmakerbotenabled,omitempty"`
	BlockAccessToSessionTranscriptsForCopilotStudio      *bool   `json:"blockaccesstosessiontranscriptsforcopilotstudio,omitempty"`
	BlockTranscriptRecordingForCopilotStudio             *bool   `json:"blocktranscriptrecordingforcopilotstudio,omitempty"`
	EnableCopilotStudioShareDataWithVivaInsights         *bool   `json:"enablecopilotstudiosharedatawithvivainsights,omitempty"`
	EnableCopilotStudioCrossGeoShareDataWithVivaInsights *bool   `json:"enablecopilotstudiocrossgeosharedatawithvivainsights,omitempty"`
	PaiPreviewScenarioEnabled                            *bool   `json:"paipreviewscenarioenabled,omitempty"`
	AiPromptsEnabled                                     *bool   `json:"aipromptsenabled,omitempty"`
	EnableIpBasedCookieBinding                           *bool   `json:"enableipbasedcookiebinding,omitempty"`
	EnableIpBasedFirewallRule                            *bool   `json:"enableipbasedfirewallrule,omitempty"`
	AllowedIpRangeForFirewall                            *string `json:"allowediprangeforfirewall,omitempty"`
	AllowedServiceTagsForFirewall                        *string `json:"allowedservicetagsforfirewall,omitempty"`
	AllowApplicationUserAccess                           *bool   `json:"allowapplicationuseraccess,omitempty"`
	AllowMicrosoftTrustedServiceTags                     *bool   `json:"allowmicrosofttrustedservicetags,omitempty"`
	EnableIpBasedFirewallRuleInAuditMode                 *bool   `json:"enableipbasedfirewallruleinauditmode,omitempty"`
	ReverseProxyIpAddresses                              *string `json:"reverseproxyipaddresses,omitempty"`
}

// SetSettingValueDto is the request body of the Dataverse SaveSettingValue()
// action.
type SetSettingValueDto struct {
	SettingName string `json:"SettingName"`
	Value       string `json:"Value"`
}
