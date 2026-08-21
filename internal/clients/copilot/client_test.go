// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package copilot

import (
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/stretchr/testify/assert"
)

// newTestClient returns a Copilot Studio client over a TestMode transport:
// auth returns a mock token and retry sleeps are no-ops, so httpmock
// responders see exactly the requests the operations build.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	cfg := &config.ProviderConfig{
		TestMode: true,
		Urls: config.ProviderConfigUrls{
			BapiUrl:               "api.bap.microsoft.com",
			PowerAppsUrl:          "api.powerapps.com",
			PowerAppsScope:        "https://service.powerapps.com/.default",
			PowerPlatformUrl:      "api.powerplatform.com",
			PowerPlatformScope:    "https://api.powerplatform.com/.default",
			AdminPowerPlatformUrl: "api.admin.powerplatform.microsoft.com",
			PowerAppsAdvisor:      "api.advisor.powerapps.com",
			PowerAppsAdvisorScope: "https://advisor.powerapps.com/.default",
			AnalyticsScope:        "https://adminanalytics.powerplatform.microsoft.com/.default",
		},
	}
	return NewClient(api.NewApiClientBase(cfg, api.NewAuthBase(cfg)))
}

// TestUnitCopilotClient_Scopes pins the Copilot Studio boundary to the fixed
// Copilot scope constant — the bot management API does not take its scope
// from the cloud configuration, and the boundary has no host() method:
// operations take the environment's PowerVirtualAgents endpoint host as a
// parameter, dataverse style.
func TestUnitCopilotClient_Scopes(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, []string{"96ff4394-9197-43aa-b393-6a41652e21f8"}, client.scopes())
	assert.Equal(t, []string{constants.COPILOT_SCOPE}, client.scopes())
}
