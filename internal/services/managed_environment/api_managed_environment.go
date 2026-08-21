// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package managed_environment

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/advisor"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
)

func newManagedEnvironmentClient(apiClient *api.Client) client {
	return client{
		Client:        bapi.NewClient(apiClient),
		advisorClient: advisor.NewClient(apiClient),
	}
}

type client struct {
	*bapi.Client
	advisorClient *advisor.Client
}

// FetchSolutionCheckerRules resolves the environment's PowerApps Advisor
// runtime endpoint via BAPI and then reads the solution checker ruleset from
// it. It spans two API boundaries, so it stays in the service layer.
func (client *client) FetchSolutionCheckerRules(ctx context.Context, environmentId string) ([]string, error) {
	if client.Client == nil {
		return nil, errors.New("environmentClient is not initialized")
	}

	env, err := client.GetEnvironment(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	if env.Properties.RuntimeEndpoints.PowerAppsAdvisor == "" {
		return nil, errors.New("PowerAppsAdvisor URL is empty")
	}

	powerAppsAdvisorUrl, err := url.Parse(env.Properties.RuntimeEndpoints.PowerAppsAdvisor)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PowerAppsAdvisor URL: %w", err)
	}

	rules, err := client.advisorClient.GetSolutionCheckerRules(ctx, powerAppsAdvisorUrl.Host)
	if err != nil {
		return nil, err
	}

	var codes []string
	for _, rule := range rules {
		codes = append(codes, rule.Code)
	}

	return codes, nil
}
