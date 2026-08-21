// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// EnableDisasterRecovery enables cross-region disaster recovery for an
// environment and waits for the lifecycle operation to finish.
func (client *Client) EnableDisasterRecovery(ctx context.Context, environmentId string) error {
	return client.setDisasterRecoveryWithRetry(ctx, environmentId, "Enabled", 0)
}

// DisableDisasterRecovery disables cross-region disaster recovery for an
// environment and waits for the lifecycle operation to finish.
func (client *Client) DisableDisasterRecovery(ctx context.Context, environmentId string) error {
	return client.setDisasterRecoveryWithRetry(ctx, environmentId, "Disabled", 0)
}

func (client *Client) setDisasterRecoveryWithRetry(ctx context.Context, environmentId, state string, retryCount int) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	body := DisasterRecoveryPatchDto{
		Properties: DisasterRecoveryPatchPropertiesDto{
			States: DisasterRecoveryPatchStatesDto{
				DisasterRecovery: DisasterRecoveryStateDto{
					Id: state,
				},
			},
		},
	}

	tflog.Debug(ctx, fmt.Sprintf("Setting Disaster Recovery to '%s' for environment '%s'", state, environmentId))

	apiResponse, err := client.Api.Execute(ctx, client.scopes(), "PATCH", apiUrl.String(), nil, body, []int{http.StatusAccepted, http.StatusConflict}, nil)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, fmt.Sprintf("Disaster Recovery PATCH HTTP Status: '%s'", apiResponse.HttpResponse.Status))

	tflog.Debug(ctx, "Waiting for Disaster Recovery lifecycle operation to complete")
	lifecycleResponse, err := client.Api.DoWaitForLifecycleOperationStatus(ctx, apiResponse)
	if err != nil {
		return err
	}

	if lifecycleResponse != nil && lifecycleResponse.State.Id == "Failed" {
		if retryCount >= constants.MAX_RETRY_COUNT {
			return fmt.Errorf("maximum retries (%d) reached for setting disaster recovery to '%s'", constants.MAX_RETRY_COUNT, state)
		}
		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return err
		}
		tflog.Info(ctx, fmt.Sprintf("Disaster Recovery operation failed. Retrying (%d/%d)...", retryCount+1, constants.MAX_RETRY_COUNT))
		return client.setDisasterRecoveryWithRetry(ctx, environmentId, state, retryCount+1)
	}

	return nil
}
