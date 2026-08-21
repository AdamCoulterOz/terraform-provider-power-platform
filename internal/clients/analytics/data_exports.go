// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package analytics

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetAnalyticsDataExport lists the analytics data export connections of the
// tenant in the given gateway cluster region.
func (client *Client) GetAnalyticsDataExport(ctx context.Context, region string) ([]AnalyticsDataDto, error) {
	// Get the analytics URL based on the region
	analyticsUrl, err := getAnalyticsUrl(region)
	if err != nil {
		return nil, err
	}

	apiUrl, err := url.Parse(analyticsUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analytics URL: %w", err)
	}
	apiUrl.Path = "api/v2/connections"

	var adr AnalyticsDataResponse
	_, err = client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &adr)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics data export: %w", err)
	}

	return adr.Value, nil
}

// getAnalyticsUrlMap maps gateway cluster geo names to the regional admin
// analytics hosts. The map is part of the wire contract: the API has no
// discovery endpoint for these hosts.
func getAnalyticsUrlMap() map[string]string {
	return map[string]string{
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
}

func getAnalyticsUrl(region string) (string, error) {
	urlMap := getAnalyticsUrlMap()
	analyticDataUrl, exists := urlMap[strings.ToUpper(region)]
	if !exists {
		return "", fmt.Errorf("invalid region: %s", region)
	}
	return analyticDataUrl, nil
}
