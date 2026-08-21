// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package copilot

// CopilotStudioAppInsightsDto is the Application Insights configuration of a
// Copilot Studio bot as sent to and returned by the bot management API.
type CopilotStudioAppInsightsDto struct {
	EnvironmentId               string `json:"environmentId"`
	BotId                       string `json:"botId"`
	AppInsightsConnectionString string `json:"appInsightsConnectionString"`
	IncludeSensitiveInformation bool   `json:"includeSensitiveInformation"`
	IncludeActivities           bool   `json:"includeActivities"`
	IncludeActions              bool   `json:"includeActions"`
	Errors                      string `json:"errors"`
	NetworkIsolation            string `json:"networkIsolation"`
}
