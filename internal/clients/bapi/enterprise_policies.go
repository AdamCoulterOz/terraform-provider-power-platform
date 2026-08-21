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

// buildEnterprisePolicyURL builds the URL for enterprise policy operations.
func (client *Client) buildEnterprisePolicyURL(environmentId, environmentType, action string) string {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/environments/%s/enterprisePolicies/%s/%s", environmentId, environmentType, action),
	}

	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.ENTERPRISE_POLICY_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	return apiUrl.String()
}

// executePolicyOperation executes a policy operation (link/unlink) with common retry logic.
func (client *Client) executePolicyOperation(ctx context.Context, environmentId, environmentType, systemId, action string) error {
	apiUrl := client.buildEnterprisePolicyURL(environmentId, environmentType, action)

	linkEnterprosePolicyDto := LinkEnterprisePolicyDto{
		SystemId: systemId,
	}

	apiResponse, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl, nil, linkEnterprosePolicyDto, []int{http.StatusAccepted, http.StatusConflict}, nil)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, fmt.Sprintf("Policy %s Operation HTTP Status: '%s'", action, apiResponse.HttpResponse.Status))
	tflog.Debug(ctx, "Waiting for operation to complete")

	lifecycleResponse, err := client.Api.DoWaitForLifecycleOperationStatus(ctx, apiResponse)
	if err != nil {
		return err
	}
	if lifecycleResponse != nil && lifecycleResponse.State.Id == "Failed" {
		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return err
		}
		tflog.Info(ctx, fmt.Sprintf("Policy %s Operation failed. Retrying...", action))
		return client.executePolicyOperation(ctx, environmentId, environmentType, systemId, action)
	}
	return nil
}

// LinkEnterprisePolicy links an enterprise policy to an environment and waits
// for the link lifecycle operation to finish.
func (client *Client) LinkEnterprisePolicy(ctx context.Context, environmentId, environmentType, systemId string) error {
	return client.executePolicyOperation(ctx, environmentId, environmentType, systemId, "link")
}

// UnLinkEnterprisePolicy unlinks an enterprise policy from an environment and
// waits for the unlink lifecycle operation to finish.
func (client *Client) UnLinkEnterprisePolicy(ctx context.Context, environmentId, environmentType, systemId string) error {
	return client.executePolicyOperation(ctx, environmentId, environmentType, systemId, "unlink")
}
