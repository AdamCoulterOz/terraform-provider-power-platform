// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package analytics

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnitGetAnalyticsUrl_PinsRegionMap pins the full gateway-cluster-region
// to regional-analytics-host map. The map is part of the wire contract: the
// API has no discovery endpoint for these hosts.
func TestUnitGetAnalyticsUrl_PinsRegionMap(t *testing.T) {
	expected := map[string]string{
		"US":   "https://na.csanalytics.powerplatform.microsoft.com/",
		"CAN":  "https://can.csanalytics.powerplatform.microsoft.com/",
		"SAM":  "https://sam.csanalytics.powerplatform.microsoft.com/",
		"EMEA": "https://emea.csanalytics.powerplatform.microsoft.com/",
		"OCE":  "https://oce.csanalytics.powerplatform.microsoft.com/",
		"PAC":  "https://apac.csanalytics.powerplatform.microsoft.com/",
		"JPN":  "https://jpn.csanalytics.powerplatform.microsoft.com/",
		"CHE":  "https://che.csanalytics.powerplatform.microsoft.com/",
		"CH":   "https://che.csanalytics.powerplatform.microsoft.com/",
		"FRA":  "https://fra.csanalytics.powerplatform.microsoft.com/",
		"UAE":  "https://uae.csanalytics.powerplatform.microsoft.com/",
		"GER":  "https://ger.csanalytics.powerplatform.microsoft.com/",
		"GBR":  "https://gbr.csanalytics.powerplatform.microsoft.com/",
		"IND":  "https://ind.csanalytics.powerplatform.microsoft.com/",
		"KOR":  "https://kor.csanalytics.powerplatform.microsoft.com/",
		"NOR":  "https://nor.csanalytics.powerplatform.microsoft.com/",
		"ZAF":  "https://zaf.csanalytics.powerplatform.microsoft.com/",
		"SGP":  "https://sgp.csanalytics.powerplatform.microsoft.com/",
		"SWE":  "https://swe.csanalytics.powerplatform.microsoft.com/",
		"GOV":  "https://gcc.csanalytics.powerplatform.microsoft.us/",
		"HIGH": "https://high.csanalytics.powerplatform.microsoft.us/",
		"DOD":  "https://dod.csanalytics.csanalytics.appsplatform.us/",
	}

	assert.Equal(t, expected, getAnalyticsUrlMap())

	for region, expectedUrl := range expected {
		gotUrl, err := getAnalyticsUrl(region)
		require.NoError(t, err)
		assert.Equal(t, expectedUrl, gotUrl, "region %s", region)
	}
}

// TestUnitGetAnalyticsUrl_IsCaseInsensitive pins that region lookup
// uppercases its input before consulting the map.
func TestUnitGetAnalyticsUrl_IsCaseInsensitive(t *testing.T) {
	gotUrl, err := getAnalyticsUrl("emea")

	require.NoError(t, err)
	assert.Equal(t, "https://emea.csanalytics.powerplatform.microsoft.com/", gotUrl)
}

// TestUnitGetAnalyticsDataExport_RequestShape pins the connections listing:
// GET api/v2/connections on the regional analytics host resolved from the
// region (case-insensitively), decoding the value array.
func TestUnitGetAnalyticsDataExport_RequestShape(t *testing.T) {
	tests := []struct {
		name        string
		region      string
		expectedUrl string
	}{
		{
			name:        "US region",
			region:      "US",
			expectedUrl: "https://na.csanalytics.powerplatform.microsoft.com/api/v2/connections",
		},
		{
			name:        "lowercase region resolves the same host",
			region:      "gov",
			expectedUrl: "https://gcc.csanalytics.powerplatform.microsoft.us/api/v2/connections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			client := newTestClient(t)

			gotUrl := ""
			httpmock.RegisterResponder(http.MethodGet, tt.expectedUrl,
				func(req *http.Request) (*http.Response, error) {
					gotUrl = req.URL.String()
					return httpmock.NewStringResponse(http.StatusOK, `{
						"value": [
							{
								"id": "30000000-0000-0000-0000-000000000000",
								"source": "app",
								"environments": [
									{"environmentId": "00000000-0000-0000-0000-000000000001", "organizationId": "20000000-0000-0000-0000-000000000000"}
								],
								"status": [
									{"name": "app", "state": "Enabled", "lastRunOn": "2024-05-01T00:00:00Z", "message": null}
								],
								"sink": {
									"id": "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Insights/components/appins-1",
									"type": "AppInsights",
									"subscriptionId": "sub-1",
									"resourceGroupName": "rg-1",
									"resourceName": "appins-1",
									"key": "key-1"
								},
								"packageName": "pkg",
								"scenarios": ["app"],
								"resourceProvider": "Microsoft.Insights",
								"aiType": "AppInsights"
							}
						]
					}`), nil
				})

			exports, err := client.GetAnalyticsDataExport(context.Background(), tt.region)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedUrl, gotUrl)
			require.Len(t, exports, 1)
			assert.Equal(t, "30000000-0000-0000-0000-000000000000", exports[0].ID)
			assert.Equal(t, "app", exports[0].Source)
			require.Len(t, exports[0].Environments, 1)
			assert.Equal(t, "00000000-0000-0000-0000-000000000001", exports[0].Environments[0].EnvironmentId)
			assert.Equal(t, "20000000-0000-0000-0000-000000000000", exports[0].Environments[0].OrganizationId)
			require.Len(t, exports[0].Status, 1)
			assert.Equal(t, "Enabled", exports[0].Status[0].State)
			assert.Nil(t, exports[0].Status[0].Message)
			assert.Equal(t, "AppInsights", exports[0].Sink.Type)
			assert.Equal(t, "appins-1", exports[0].Sink.ResourceName)
			assert.Equal(t, "pkg", exports[0].PackageName)
			assert.Equal(t, []string{"app"}, exports[0].Scenarios)
			assert.Equal(t, "Microsoft.Insights", exports[0].ResourceProvider)
			assert.Equal(t, "AppInsights", exports[0].AiType)
		})
	}
}

// TestUnitGetAnalyticsDataExport_UnknownRegion pins that an unmapped region
// fails with the invalid-region error, preserving the caller's casing, before
// any request is made.
func TestUnitGetAnalyticsDataExport_UnknownRegion(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	exports, err := client.GetAnalyticsDataExport(context.Background(), "mars")

	require.Error(t, err)
	assert.Nil(t, exports)
	assert.Equal(t, "invalid region: mars", err.Error())
	assert.Zero(t, httpmock.GetTotalCallCount(), "an unknown region must fail without calling the API")
}

// TestUnitGetAnalyticsDataExport_UnexpectedStatus pins that a status outside
// the accepted [200] surfaces the customerrors unexpected-status error
// wrapped in the operation's context.
func TestUnitGetAnalyticsDataExport_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "https://na.csanalytics.powerplatform.microsoft.com/api/v2/connections",
		httpmock.NewStringResponder(http.StatusForbidden, "no access"))

	exports, err := client.GetAnalyticsDataExport(context.Background(), "US")

	require.Error(t, err)
	assert.Nil(t, exports)
	assert.ErrorContains(t, err, "failed to get analytics data export:")
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusForbidden, statusErr.StatusCode)
}

// TestUnitGetAnalyticsDataExport_RetriesTransientStatus pins that the
// operation goes through Execute's transient-status retry (instant in
// TestMode).
func TestUnitGetAnalyticsDataExport_RetriesTransientStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://na.csanalytics.powerplatform.microsoft.com/api/v2/connections",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusBadGateway, "bad gateway"), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, `{"value": []}`), nil
		})

	exports, err := client.GetAnalyticsDataExport(context.Background(), "US")

	require.NoError(t, err)
	assert.Empty(t, exports)
	assert.Equal(t, 2, attempts, "a transient 502 must be retried")
}
