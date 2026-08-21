// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package analytics

import (
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/stretchr/testify/assert"
)

// newTestClient returns an admin analytics client over a TestMode transport:
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

// TestUnitAnalyticsClient_Scopes pins the admin analytics boundary to the
// analytics scope configured for the cloud. The boundary has no host()
// method: hosts are derived per request from the gateway cluster region.
func TestUnitAnalyticsClient_Scopes(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, []string{"https://adminanalytics.powerplatform.microsoft.com/.default"}, client.scopes())
}
