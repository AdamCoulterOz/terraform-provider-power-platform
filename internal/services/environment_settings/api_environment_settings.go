// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment_settings

import (
	"context"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

func newEnvironmentSettingsClient(apiClient *api.Client) client {
	return client{
		Client:    bapi.NewClient(apiClient),
		dataverse: dataverse.NewClient(apiClient),
	}
}

type client struct {
	*bapi.Client
	dataverse *dataverse.Client
}

func (client *client) DataverseExists(ctx context.Context, environmentId string) (bool, error) {
	env, err := client.GetEnvironment(ctx, environmentId)
	if err != nil {
		return false, err
	}
	return env.Properties.LinkedEnvironmentMetadata != nil && env.Properties.LinkedEnvironmentMetadata.InstanceURL != "", nil
}

func (client *client) GetEnvironmentSettings(ctx context.Context, environmentId string) (*environmentSettings, error) {
	backendSettings, err := client.getEnvironmentBackendSettings(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	orgSettings, err := client.getEnvironmentOrgSettings(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return &environmentSettings{
		BackendSettings: backendSettings,
		OrgSettings:     orgSettings,
	}, nil
}

func (client *client) getEnvironmentBackendSettings(ctx context.Context, environmentId string) (*dataverse.EnvironmentBackendSettingsValueDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverse.GetEnvironmentBackendSettings(ctx, environmentHost)
}

func (client *client) getEnvironmentOrgSettings(ctx context.Context, environmentId string) (*dataverse.EnvironmentOrgSettingsDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverse.GetEnvironmentOrgSettings(ctx, environmentHost)
}

func (client *client) UpdateEnvironmentSettings(ctx context.Context, environmentId string, settings environmentSettings) (*environmentSettings, error) {
	updatedOrgSettings, err := client.updateEnvironmentOrgSettings(ctx, environmentId, *settings.OrgSettings)
	if err != nil {
		return nil, err
	}

	updatedBackendSettings, err := client.updateEnvironmentBackendSettings(ctx, environmentId, *settings.BackendSettings)
	if err != nil {
		return nil, err
	}

	updateSettings := &environmentSettings{
		BackendSettings: updatedBackendSettings,
		OrgSettings:     updatedOrgSettings,
	}
	return updateSettings, nil
}

func (client *client) updateEnvironmentOrgSettings(ctx context.Context, environmentId string, orgSettings dataverse.EnvironmentOrgSettingsDto) (*dataverse.EnvironmentOrgSettingsDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverse.UpdateEnvironmentOrgSettings(ctx, environmentHost, orgSettings)
}

func (client *client) updateEnvironmentBackendSettings(ctx context.Context, environmentId string, backendSettings dataverse.EnvironmentBackendSettingsValueDto) (*dataverse.EnvironmentBackendSettingsValueDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverse.UpdateEnvironmentBackendSettings(ctx, environmentHost, backendSettings)
}
