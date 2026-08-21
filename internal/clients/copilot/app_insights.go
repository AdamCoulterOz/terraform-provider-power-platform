// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package copilot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetAppInsightsConfiguration reads the Application Insights configuration of
// a Copilot Studio bot through the environment's PowerVirtualAgents endpoint.
// The endpoint host and tenant id are resolved by callers from the
// environment via the bapi client.
func (client *Client) GetAppInsightsConfiguration(ctx context.Context, endpointHost, tenantId, environmentId, botId string) (*CopilotStudioAppInsightsDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   endpointHost,
		Path:   fmt.Sprintf("/api/botmanagement/2022-01-15/environments/%s/bots/%s/applicationinsightsconfiguration", environmentId, botId),
	}
	values := url.Values{}
	apiUrl.RawQuery = values.Encode()

	copilotStudioAppInsights := CopilotStudioAppInsightsDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), http.Header{"x-cci-tenantid": {tenantId}}, nil, []int{http.StatusOK}, &copilotStudioAppInsights)
	if err != nil {
		return nil, fmt.Errorf("failed to get Copilot Studio application insights configuration: %w", err)
	}
	copilotStudioAppInsights.EnvironmentId = environmentId
	copilotStudioAppInsights.BotId = botId
	return &copilotStudioAppInsights, nil
}

// UpdateAppInsightsConfiguration updates the Application Insights
// configuration of a Copilot Studio bot through the environment's
// PowerVirtualAgents endpoint. The endpoint host and tenant id are resolved
// by callers from the environment via the bapi client.
func (client *Client) UpdateAppInsightsConfiguration(ctx context.Context, endpointHost, tenantId string, copilotStudioAppInsightsConfig CopilotStudioAppInsightsDto, botId string) (*CopilotStudioAppInsightsDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   endpointHost,
		Path:   fmt.Sprintf("/api/botmanagement/2022-01-15/environments/%s/bots/%s/applicationinsightsconfiguration", copilotStudioAppInsightsConfig.EnvironmentId, botId),
	}
	values := url.Values{}
	apiUrl.RawQuery = values.Encode()

	updatedCopilotStudioAppInsightsConfiguration := CopilotStudioAppInsightsDto{}

	resp, err := client.Api.Execute(ctx, client.scopes(), "PUT", apiUrl.String(), http.Header{"x-cci-tenantid": {tenantId}}, copilotStudioAppInsightsConfig, []int{http.StatusOK, http.StatusInternalServerError}, &updatedCopilotStudioAppInsightsConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to update Copilot Studio application insights configuration: %w", err)
	}
	if resp.HttpResponse.StatusCode == http.StatusInternalServerError {
		return nil, fmt.Errorf("error updating Application Insights configuration: %s", string(resp.BodyAsBytes))
	}

	if len(updatedCopilotStudioAppInsightsConfiguration.Errors) > 0 {
		return nil, fmt.Errorf("error updating Application Insights configuration: %s", updatedCopilotStudioAppInsightsConfiguration.Errors)
	}
	updatedCopilotStudioAppInsightsConfiguration.EnvironmentId = copilotStudioAppInsightsConfig.EnvironmentId
	updatedCopilotStudioAppInsightsConfiguration.BotId = botId
	return &updatedCopilotStudioAppInsightsConfiguration, nil
}
