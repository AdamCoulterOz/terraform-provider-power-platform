// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package solution_checker_rules

import (
	"context"
	"fmt"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/advisor"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
)

type Client struct {
	*advisor.Client
	environmentClient *bapi.Client
}

func NewSolutionCheckerRulesClient(apiClient *api.Client) *Client {
	return &Client{
		Client:            advisor.NewClient(apiClient),
		environmentClient: bapi.NewClient(apiClient),
	}
}

// GetSolutionCheckerRules resolves the environment's PowerApps Advisor
// runtime endpoint via BAPI and then reads the solution checker ruleset from
// it. It spans two API boundaries, so it stays in the service layer.
func (c *Client) GetSolutionCheckerRules(ctx context.Context, environmentId string) ([]advisor.RuleDto, error) {
	env, err := c.environmentClient.GetEnvironment(ctx, environmentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment details for %s: %w", environmentId, err)
	}

	if env.Properties.RuntimeEndpoints.PowerAppsAdvisor == "" {
		return nil, fmt.Errorf("could not find PowerAppsAdvisor endpoint for environment %s", environmentId)
	}

	// Parse and get host from PowerAppsAdvisor URL
	advisorURL, err := url.Parse(env.Properties.RuntimeEndpoints.PowerAppsAdvisor)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PowerAppsAdvisor URL: %w", err)
	}

	rules, err := c.Client.GetSolutionCheckerRules(ctx, advisorURL.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to get solution checker rules: %w", err)
	}

	return rules, nil
}
