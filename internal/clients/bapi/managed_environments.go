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

// EnableManagedEnvironment applies a governance configuration to an
// environment, making it a managed environment, and waits for the enablement
// lifecycle operation to finish.
func (client *Client) EnableManagedEnvironment(ctx context.Context, managedEnvSettings GovernanceConfigurationDto, environmentId string) error {
	return client.enableManagedEnvironmentWithRetry(ctx, managedEnvSettings, environmentId, 0)
}

func (client *Client) enableManagedEnvironmentWithRetry(ctx context.Context, managedEnvSettings GovernanceConfigurationDto, environmentId string, retryCount int) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/environments/%s/governanceConfiguration", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	apiResponse, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, managedEnvSettings, []int{http.StatusNoContent, http.StatusAccepted, http.StatusConflict}, nil)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, "Managed Environment Enablement Operation HTTP Status: '"+apiResponse.HttpResponse.Status+"'")

	// A 409 means the request was rejected, not applied. It comes back both when the environment is
	// already managed and when the environment is busy, for example straight after it was created.
	// Treating it as success silently skips the enablement: there is no lifecycle operation to wait
	// for, and the caller then fails reading state back with "doesn't have managed environment
	// feature enabled". Tell the two cases apart by looking at the environment itself, and retry when
	// it is genuinely not managed yet.
	if apiResponse.HttpResponse.StatusCode == http.StatusConflict {
		env, envErr := client.GetEnvironment(ctx, environmentId)
		if envErr != nil {
			return envErr
		}
		if env.Properties.GovernanceConfiguration != nil && env.Properties.GovernanceConfiguration.ProtectionLevel == constants.PROTECTION_LEVEL_STANDARD {
			// Already managed, so there is nothing to enable and retrying cannot help. This also covers
			// an environment that is in an environment group, where the group made it managed and the
			// governance configuration is locked; the caller reports that when it reads the settings.
			tflog.Debug(ctx, "Managed Environment is already enabled, nothing to do")
			return nil
		}
		if retryCount >= constants.MAX_RETRY_COUNT {
			return fmt.Errorf("maximum retries (%d) reached for EnableManagedEnvironment: the environment kept rejecting the request with 409 and is still not managed", constants.MAX_RETRY_COUNT)
		}
		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return err
		}
		tflog.Info(ctx, "Managed Environment Enablement was rejected with 409 and the environment is not managed yet. Retrying...")
		return client.enableManagedEnvironmentWithRetry(ctx, managedEnvSettings, environmentId, retryCount+1)
	}

	tflog.Debug(ctx, "Waiting for Managed Environment Enablement Operation to complete")
	lifecycleResponse, err := client.Api.DoWaitForLifecycleOperationStatus(ctx, apiResponse)
	if err != nil {
		return err
	}
	if lifecycleResponse != nil && lifecycleResponse.State.Id == "Failed" {
		if retryCount >= constants.MAX_RETRY_COUNT {
			return fmt.Errorf("maximum retries (%d) reached for EnableManagedEnvironment on lifecycle failure", constants.MAX_RETRY_COUNT)
		}
		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return err
		}
		tflog.Info(ctx, "Managed Environment Enablement Operation failed. Retrying...")
		return client.enableManagedEnvironmentWithRetry(ctx, managedEnvSettings, environmentId, retryCount+1)
	}
	return nil
}

// DisableManagedEnvironment resets an environment's governance configuration
// to basic protection, disabling the managed environment feature, and waits
// for the disablement lifecycle operation to finish.
func (client *Client) DisableManagedEnvironment(ctx context.Context, environmentId string) error {
	return client.disableManagedEnvironmentWithRetry(ctx, environmentId, 0)
}

func (client *Client) disableManagedEnvironmentWithRetry(ctx context.Context, environmentId string, retryCount int) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/environments/%s/governanceConfiguration", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	managedEnv := GovernanceConfigurationDto{
		ProtectionLevel: "Basic",
	}

	apiResponse, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, managedEnv, []int{http.StatusAccepted, http.StatusConflict}, nil)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, "Managed Environment Disablement Operation HTTP Status: '"+apiResponse.HttpResponse.Status+"'")
	tflog.Debug(ctx, "Waiting for Managed Environment Disablement Operation to complete")

	lifecycleResponse, err := client.Api.DoWaitForLifecycleOperationStatus(ctx, apiResponse)
	if err != nil {
		return err
	}
	if lifecycleResponse != nil && lifecycleResponse.State.Id == "Failed" {
		if retryCount >= constants.MAX_RETRY_COUNT {
			return fmt.Errorf("maximum retries (%d) reached for DisableManagedEnvironment on lifecycle failure", constants.MAX_RETRY_COUNT)
		}
		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return err
		}
		tflog.Info(ctx, "Managed Environment Disablement Operation failed. Retrying...")
		return client.disableManagedEnvironmentWithRetry(ctx, environmentId, retryCount+1)
	}
	return nil
}
