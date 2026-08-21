// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package advisor

import (
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/stretchr/testify/assert"
)

// newTestClient returns an Advisor client over a TestMode transport: auth
// returns a mock token and retry sleeps are no-ops, so httpmock responders
// see exactly the requests the operations build.
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

// TestUnitAdvisorClient_Scopes pins the Advisor boundary to the advisor scope
// configured for the cloud. The boundary has no host() method: operations
// take the environment's advisor endpoint host as a parameter, dataverse
// style.
func TestUnitAdvisorClient_Scopes(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, []string{"https://advisor.powerapps.com/.default"}, client.scopes())
}
