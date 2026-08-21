// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

// GetPolicies lists all DLP policies, reading each policy's full definition.
func (client *Client) GetPolicies(ctx context.Context) ([]DlpPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "providers/PowerPlatform.Governance/v2/policies",
	}
	policiesArray := DlpPolicyDefinitionArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &policiesArray)
	if err != nil {
		return nil, err
	}

	policies := make([]DlpPolicyDto, 0)
	for _, policy := range policiesArray.Value {
		apiUrl := &url.URL{
			Scheme: constants.HTTPS,
			Host:   client.host(),
			Path:   fmt.Sprintf("providers/PowerPlatform.Governance/v2/policies/%s", policy.PolicyDefinition.Name),
		}
		policy := DlpPolicyDto{}
		_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &policy)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

// GetPolicy reads a single DLP policy by name.
func (client *Client) GetPolicy(ctx context.Context, name string) (*DlpPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("providers/PowerPlatform.Governance/v2/policies/%s", name),
	}
	policy := DlpPolicyDto{}
	resp, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &policy)
	if err != nil {
		if resp != nil && resp.HttpResponse.StatusCode == http.StatusNotFound {
			return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("DLP Policy '%s' not found", name))
		}
		return nil, err
	}
	return &policy, nil
}

// DeletePolicy deletes a DLP policy by name.
func (client *Client) DeletePolicy(ctx context.Context, name string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("providers/PowerPlatform.Governance/v2/policies/%s", name),
	}
	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK}, nil)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePolicy updates the DLP policy named in the policy definition.
func (client *Client) UpdatePolicy(ctx context.Context, policy DlpPolicyDto) (*DlpPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("providers/PowerPlatform.Governance/v2/policies/%s", policy.PolicyDefinition.Name),
	}
	createdPolicy := DlpPolicyDto{}

	_, err := client.Api.Execute(ctx, client.scopes(), "PATCH", apiUrl.String(), nil, policy, []int{http.StatusOK}, &createdPolicy)
	if err != nil {
		return nil, err
	}

	return &createdPolicy, nil
}

// CreatePolicy creates a DLP policy.
func (client *Client) CreatePolicy(ctx context.Context, policy DlpPolicyDto) (*DlpPolicyDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/providers/PowerPlatform.Governance/v2/policies",
	}

	createdPolicy := DlpPolicyDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, policy, []int{http.StatusCreated}, &createdPolicy)
	if err != nil {
		return nil, err
	}
	return &createdPolicy, nil
}
