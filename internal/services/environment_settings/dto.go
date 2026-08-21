// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment_settings

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

const (
	ON                                = "On"
	OFF                               = "Off"
	DEFAULT                           = "Default"
	AUTO                              = "Auto"
	ALL_USERS                         = "AllUsers"
	USER_AS_FEATURE_BECOMES_AVAILABLE = "UserAsFeatureBecomesAvailable"
	NO_ONE                            = "NoOne"
)

const (
	EnableCopilotAnswerControl              = "EnableCopilotAnswerControl"
	AppCopilotEnabled                       = "appcopilotenabled"
	FormPredictEnabled                      = "FormPredictEnabled"
	FormPredictSmartPasteEnabledOnByDefault = "FormPredictSmartPasteEnabledOnByDefault"
	FormFillBarUXEnabled                    = "FormFillBarUXEnabled"
	NLGridSearchSetting                     = "NLGridSearchSetting"
	NLChartDataVisualizationSetting         = "NLChartDataVisualizationSetting"
)

type environmentSettings struct {
	BackendSettings *dataverse.EnvironmentBackendSettingsValueDto
	OrgSettings     *dataverse.EnvironmentOrgSettingsDto
}
