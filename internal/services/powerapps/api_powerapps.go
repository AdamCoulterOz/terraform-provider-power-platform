// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerapps

import (
	"context"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/powerapps"
)

func newPowerAppssClient(apiClient *api.Client) client {
	return client{
		Client:            powerapps.NewClient(apiClient),
		environmentClient: bapi.NewClient(apiClient),
	}
}

type client struct {
	*powerapps.Client
	environmentClient *bapi.Client
}

// GetPowerApps lists the Power Apps of every environment in the tenant. The
// environment list comes from BAPI and the apps of each environment from the
// PowerApps API, so the composition lives here in the service.
func (client *client) GetPowerApps(ctx context.Context) ([]powerapps.PowerAppBapiDto, error) {
	envs, err := client.environmentClient.GetEnvironments(ctx)
	if err != nil {
		return nil, err
	}
	apps := make([]powerapps.PowerAppBapiDto, 0)
	for _, env := range envs {
		envApps, err := client.GetEnvironmentPowerApps(ctx, env.Name)
		if err != nil {
			return nil, err
		}
		apps = append(apps, envApps...)
	}
	return apps, nil
}
