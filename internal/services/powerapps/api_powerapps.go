// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerapps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

func newPowerAppssClient(apiClient *api.Client) client {
	return client{
		Api:               apiClient,
		environmentClient: bapi.NewClient(apiClient),
	}
}

type client struct {
	Api               *api.Client
	environmentClient *bapi.Client
}

func (client *client) GetPowerApps(ctx context.Context) ([]powerAppBapiDto, error) {
	envs, err := client.environmentClient.GetEnvironments(ctx)
	if err != nil {
		return nil, err
	}
	apps := make([]powerAppBapiDto, 0)
	for _, env := range envs {
		apiUrl := &url.URL{
			Scheme: constants.HTTPS,
			Host:   client.Api.GetConfig().Urls.PowerAppsUrl,
			Path:   fmt.Sprintf("/providers/Microsoft.PowerApps/scopes/admin/environments/%s/apps", env.Name),
		}
		values := url.Values{}
		values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
		apiUrl.RawQuery = values.Encode()

		appsArray := powerAppArrayDto{}
		_, err := client.Api.Execute(ctx, nil, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &appsArray)
		if err != nil {
			return nil, err
		}
		apps = append(apps, appsArray.Value...)
	}
	return apps, nil
}
