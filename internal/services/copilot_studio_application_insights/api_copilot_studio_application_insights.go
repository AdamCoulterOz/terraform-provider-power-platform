// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package copilot_studio_application_insights

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/copilot"
)

func newCopilotStudioClient(apiClient *api.Client) client {
	return client{
		Client:            copilot.NewClient(apiClient),
		EnvironmentClient: bapi.NewClient(apiClient),
	}
}

type client struct {
	*copilot.Client
	EnvironmentClient *bapi.Client
}

func (client *client) getCopilotStudioEndpoint(ctx context.Context, environmentId string) (string, error) {
	env, err := client.EnvironmentClient.GetEnvironment(ctx, environmentId)
	if err != nil {
		return "", fmt.Errorf("failed to get environment %s: %w", environmentId, err)
	}
	if env == nil || env.Properties == nil || env.Properties.RuntimeEndpoints == nil || env.Properties.RuntimeEndpoints.PowerVirtualAgents == "" {
		return "", errors.New("power virtual agents runtime endpoint is not available in the environment")
	}

	u, err := url.Parse(env.Properties.RuntimeEndpoints.PowerVirtualAgents)
	if err != nil {
		return "", fmt.Errorf("failed to parse Power Virtual Agents endpoint: %w", err)
	}

	return u.Host, nil
}

func parseImportId(importId string) (envId string, botId string, err error) {
	parts := strings.Split(importId, "_")
	if len(parts) != 2 {
		return "", "", errors.New("invalid import id format")
	}
	return parts[0], parts[1], nil
}

// getCopilotStudioAppInsightsConfiguration resolves the environment's
// PowerVirtualAgents runtime endpoint and tenant id via BAPI and then reads
// the bot's Application Insights configuration from it. It spans two API
// boundaries, so it stays in the service layer.
func (client *client) getCopilotStudioAppInsightsConfiguration(ctx context.Context, importId string) (*copilot.CopilotStudioAppInsightsDto, error) {
	environmentId, botId, err := parseImportId(importId)
	if err != nil {
		return nil, fmt.Errorf("failed to parse import ID: %w", err)
	}
	copilotStudioEndpoint, err := client.getCopilotStudioEndpoint(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	env, err := client.EnvironmentClient.GetEnvironment(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	if env == nil || env.Properties == nil || env.Properties.TenantId == "" {
		return nil, errors.New("TenantId value is not available in the environment properties")
	}

	return client.GetAppInsightsConfiguration(ctx, copilotStudioEndpoint, env.Properties.TenantId, environmentId, botId)
}

// updateCopilotStudioAppInsightsConfiguration resolves the environment's
// PowerVirtualAgents runtime endpoint and tenant id via BAPI and then updates
// the bot's Application Insights configuration through it. It spans two API
// boundaries, so it stays in the service layer.
func (client *client) updateCopilotStudioAppInsightsConfiguration(ctx context.Context, copilotStudioAppInsightsConfig copilot.CopilotStudioAppInsightsDto, botId string) (*copilot.CopilotStudioAppInsightsDto, error) {
	copilotStudioEndpoint, err := client.getCopilotStudioEndpoint(ctx, copilotStudioAppInsightsConfig.EnvironmentId)
	if err != nil {
		return nil, err
	}

	env, err := client.EnvironmentClient.GetEnvironment(ctx, copilotStudioAppInsightsConfig.EnvironmentId)
	if err != nil {
		return nil, err
	}

	if env == nil || env.Properties == nil || env.Properties.TenantId == "" {
		return nil, errors.New("TenantId value is not available in the environment properties")
	}

	return client.UpdateAppInsightsConfiguration(ctx, copilotStudioEndpoint, env.Properties.TenantId, copilotStudioAppInsightsConfig, botId)
}
