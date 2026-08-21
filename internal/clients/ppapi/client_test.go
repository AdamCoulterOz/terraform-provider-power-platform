// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/stretchr/testify/assert"
)

// Shared fixtures for the ppapi package tests. The environment and tenant
// hosts are the literal values helpers.BuildEnvironmentHostUri and
// helpers.BuildTenantHostUri produce for the ids below against the public
// cloud Power Platform API host; they are spelled out so a change in host
// construction fails these tests.
const (
	testEnvironmentId   = "00000000-0000-0000-0000-000000000001"
	testTenantId        = "00000000-0000-0000-0000-000000000002"
	testEnvironmentHost = "000000000000000000000000000000.01.environment.api.powerplatform.com"
	testTenantHost      = "000000000000000000000000000000.02.tenant.api.powerplatform.com"
)

// newTestClient returns a Client backed by the public cloud URL fixture with
// TestMode enabled, so token acquisition returns a mock token and
// SleepWithContext is a no-op.
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

func TestUnitNewClient_WiresSharedTransport(t *testing.T) {
	cfg := &config.ProviderConfig{TestMode: true}
	apiClient := api.NewApiClientBase(cfg, api.NewAuthBase(cfg))

	client := NewClient(apiClient)

	assert.Same(t, apiClient, client.Api)
}

func TestUnitClient_Scopes(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, []string{"https://api.powerplatform.com/.default"}, client.scopes())
}

func TestUnitClient_Host(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, "api.powerplatform.com", client.host())
}

func TestUnitClient_TenantHost(t *testing.T) {
	client := newTestClient(t)

	assert.Equal(t, testTenantHost, client.tenantHost(testTenantId))
}
