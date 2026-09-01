// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment_wave

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/admin"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

type environmentWaveClient struct {
	*admin.Client
	environmentClient *bapi.Client
}

func newEnvironmentWaveClient(apiClient *api.Client) *environmentWaveClient {
	return &environmentWaveClient{
		Client:            admin.NewClient(apiClient),
		environmentClient: bapi.NewClient(apiClient),
	}
}

func (client *environmentWaveClient) GetOrgEnvironmentId(ctx context.Context, environmentId string) (*admin.OrganizationDto, error) {
	organizations, err := client.GetOrganizations(ctx)
	if err != nil {
		return nil, err
	}

	env, err := client.environmentClient.GetEnvironment(ctx, environmentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment %s: %w", environmentId, err)
	}

	for _, org := range organizations {
		if org.Id == env.Properties.LinkedEnvironmentMetadata.ResourceId {
			return &org, nil
		}
	}
	return nil, fmt.Errorf("geo for environment with ID %s not found", environmentId)
}

func (client *environmentWaveClient) UpdateFeature(ctx context.Context, environmentId string, featureName string) (*admin.FeatureDto, error) {
	org, err := client.GetOrgEnvironmentId(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	if err := client.EnableFeature(ctx, org.Id, org.CrmGeo, featureName); err != nil {
		return nil, err
	}

	retryAfter := api.DefaultRetryAfter()
	for {
		// Check if context has been cancelled or deadline exceeded
		if err := helpers.CheckContextTimeout(ctx, fmt.Sprintf("waiting for feature %s to be enabled in environment %s", featureName, environmentId)); err != nil {
			return nil, err
		}

		feature, err := client.GetFeature(ctx, environmentId, featureName)
		if err != nil {
			return nil, err
		}

		if feature != nil && feature.AppsUpgradeState != "Upgrading" {
			tflog.Info(ctx, fmt.Sprintf("Feature %s  with state: %s", featureName, feature.AppsUpgradeState))
			return feature, nil
		}

		err = client.Api.SleepWithContext(ctx, retryAfter)
		if err != nil {
			return nil, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Feature %s not yet enabled, polling...", featureName))
	}
}

func (client *environmentWaveClient) GetFeature(ctx context.Context, environmentId string, featureName string) (*admin.FeatureDto, error) {
	org, err := client.GetOrgEnvironmentId(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	feature, err := client.Client.GetFeature(ctx, org.Id, org.CrmGeo, featureName)
	if err != nil {
		return nil, fmt.Errorf("failed to get features for environment %s: %w", environmentId, err)
	}

	return feature, nil
}
