// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

// GetBillingPolicies lists all billing policies in the tenant.
func (client *Client) GetBillingPolicies(ctx context.Context) ([]BillingPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "licensing/billingPolicies",
	}

	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	policies := BillingPolicyArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &policies)

	return policies.Value, err
}

// GetBillingPolicy reads a single billing policy by id.
func (client *Client) GetBillingPolicy(ctx context.Context, billingId string) (*BillingPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/licensing/billingPolicies/%s", billingId),
	}

	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	policy := BillingPolicyDto{}
	resp, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &policy)

	if err != nil {
		if resp != nil && resp.HttpResponse.StatusCode == http.StatusNotFound {
			return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("Billing Policy with ID '%s' not found", billingId))
		}
		return nil, err
	}
	return &policy, nil
}

// CreateBillingPolicy creates a billing policy and waits until it reaches a
// terminal (Enabled/Disabled) status.
func (client *Client) CreateBillingPolicy(ctx context.Context, policyToCreate BillingPolicyCreateDto) (*BillingPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/licensing/BillingPolicies",
	}

	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	policy := &BillingPolicyDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, policyToCreate, []int{http.StatusCreated}, policy)
	if err != nil {
		return nil, err
	}

	// If billing policy status is not Enabled or Disabled, wait for it to reach a terminal state
	if policy.Status != "Enabled" && policy.Status != "Disabled" {
		policy, err = client.DoWaitForFinalStatus(ctx, policy)

		if err != nil {
			return nil, err
		}
	}

	return policy, err
}

// UpdateBillingPolicy updates a billing policy and waits until it reaches a
// terminal (Enabled/Disabled) status.
func (client *Client) UpdateBillingPolicy(ctx context.Context, billingId string, policyToUpdate BillingPolicyUpdateDto) (*BillingPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/licensing/billingPolicies/%s", billingId),
	}

	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	policy := &BillingPolicyDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "PUT", apiUrl.String(), nil, policyToUpdate, []int{http.StatusOK}, policy)

	// If billing policy status is not Enabled or Disabled, wait for it to reach a terminal state
	if policy.Status != "Enabled" && policy.Status != "Disabled" {
		policy, err = client.DoWaitForFinalStatus(ctx, policy)

		if err != nil {
			return nil, err
		}
	}

	return policy, err
}

// DeleteBillingPolicy deletes a billing policy by id.
func (client *Client) DeleteBillingPolicy(ctx context.Context, billingId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/licensing/BillingPolicies/%s", billingId),
	}

	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusNoContent}, nil)

	return err
}

// GetEnvironmentsForBillingPolicy lists the ids (lowercased) of the
// environments linked to a billing policy.
func (client *Client) GetEnvironmentsForBillingPolicy(ctx context.Context, billingId string) ([]string, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("licensing/billingPolicies/%s/environments", billingId),
	}

	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	billingPolicyEnvironments := BillingPolicyEnvironmentsArrayResponseDto{}
	resp, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &billingPolicyEnvironments)
	if err != nil {
		if resp != nil && resp.HttpResponse.StatusCode == http.StatusNotFound {
			return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("Billing Policy with ID '%s' not found", billingId))
		}
		return nil, err
	}

	environments := []string{}
	for _, billingPolicyEnvironment := range billingPolicyEnvironments.Value {
		environments = append(environments, strings.ToLower(billingPolicyEnvironment.EnvironmentId))
	}
	return environments, err
}

// AddEnvironmentsToBillingPolicy links environments to a billing policy.
func (client *Client) AddEnvironmentsToBillingPolicy(ctx context.Context, billingId string, environmentIds []string) error {
	if len(environmentIds) == 0 {
		return nil
	}
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("licensing/billingPolicies/%s/environments/add", billingId),
	}

	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")
	apiUrl.RawQuery = values.Encode()

	normalizedIds := make([]string, len(environmentIds))
	for i, id := range environmentIds {
		normalizedIds[i] = strings.ToLower(id)
	}

	environments := BillingPolicyEnvironmentsArrayDto{
		EnvironmentIds: normalizedIds,
	}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, environments, []int{http.StatusOK}, nil)

	return err
}

// RemoveEnvironmentsToBillingPolicy unlinks environments from a billing policy.
func (client *Client) RemoveEnvironmentsToBillingPolicy(ctx context.Context, billingId string, environmentIds []string) error {
	if len(environmentIds) == 0 {
		return nil
	}
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("licensing/billingPolicies/%s/environments/remove", billingId),
	}

	values := url.Values{}
	values.Add("api-version", "2022-03-01-preview")

	normalizedIds := make([]string, len(environmentIds))
	for i, id := range environmentIds {
		normalizedIds[i] = strings.ToLower(id)
	}
	apiUrl.RawQuery = values.Encode()

	environments := BillingPolicyEnvironmentsArrayDto{
		EnvironmentIds: normalizedIds,
	}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, environments, []int{http.StatusOK}, nil)

	return err
}

// DoWaitForFinalStatus polls a billing policy until it reaches a terminal
// (Enabled/Disabled) status.
func (client *Client) DoWaitForFinalStatus(ctx context.Context, billingPolicyDto *BillingPolicyDto) (*BillingPolicyDto, error) {
	billingId := billingPolicyDto.Id

	for {
		billingPolicy, err := client.GetBillingPolicy(ctx, billingId)

		if err != nil {
			return nil, err
		}

		if billingPolicy.Status == "Enabled" || billingPolicy.Status == "Disabled" {
			return billingPolicy, nil
		}

		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return nil, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Billing Policy Operation State: '%s'", billingPolicy.Status))
	}
}
