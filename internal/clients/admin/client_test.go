// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package admin

import (
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/stretchr/testify/assert"
)

// newTestClient returns an Athena client over a TestMode transport: auth
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

// TestUnitAthenaClient_Scopes pins the Athena boundary to the fixed PPAC
// scope constant — the admin center API does not take its scope from the
// cloud configuration.
func TestUnitAthenaClient_Scopes(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, []string{"065d9450-1e87-434e-ac2f-69af271549ed"}, client.scopes())
	assert.Equal(t, []string{constants.PPAC_SCOPE}, client.scopes())
}

// TestUnitAthenaClient_Host pins the Athena boundary to the admin center host
// configured for the cloud.
func TestUnitAthenaClient_Host(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, "api.admin.powerplatform.microsoft.com", client.host())
}
