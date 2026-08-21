// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

// GetEnvironmentGroupRuleSet reads the legacy rule set of an environment group
// from the tenant-scoped host.
func (client *Client) GetEnvironmentGroupRuleSet(ctx context.Context, tenantId, environmentGroupId string) (*EnvironmentGroupRuleSetValueSetDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.tenantHost(tenantId),
		Path:   fmt.Sprintf("/governance/environmentGroups/%s/ruleSets", environmentGroupId),
	}

	values := url.Values{}
	values.Add("api-version", "2021-10-01-preview")
	apiUrl.RawQuery = values.Encode()

	environmentGroupRuleSet := EnvironmentGroupRuleSetDto{}
	resp, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNoContent}, &environmentGroupRuleSet)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment group rule set: %w", err)
	}

	if resp.HttpResponse.StatusCode == http.StatusNoContent {
		return nil, customerrors.WrapIntoProviderError(
			fmt.Errorf("rule set '%s' not found", environmentGroupId),
			constants.ERROR_OBJECT_NOT_FOUND,
			fmt.Sprintf("rule set '%s' not found", environmentGroupId),
		)
	}

	if len(environmentGroupRuleSet.Value) == 0 {
		return nil, fmt.Errorf("no environment group ruleset found for environment group id %s", environmentGroupId)
	}

	return &environmentGroupRuleSet.Value[0], nil
}

// CreateEnvironmentGroupRuleSet creates the legacy rule set of an environment
// group on the tenant-scoped host.
func (client *Client) CreateEnvironmentGroupRuleSet(ctx context.Context, tenantId, environmentGroupId string, newEnvironmentGroupRuleSet EnvironmentGroupRuleSetValueSetDto) (*EnvironmentGroupRuleSetValueSetDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.tenantHost(tenantId),
		Path:   fmt.Sprintf("/governance/environmentGroups/%s/ruleSets", environmentGroupId),
	}

	values := url.Values{}
	values.Add("api-version", "2021-10-01-preview")
	apiUrl.RawQuery = values.Encode()

	environmentGroupRuleSet := EnvironmentGroupRuleSetValueSetDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, newEnvironmentGroupRuleSet, []int{http.StatusCreated}, &environmentGroupRuleSet)

	if err != nil {
		return nil, fmt.Errorf("failed to create environment group rule set: %w", err)
	}

	// If empty Parameters is a valid API response, remove the following check.
	// Otherwise, consider clarifying its necessity and error messaging.
	if len(environmentGroupRuleSet.Parameters) == 0 {
		return nil, fmt.Errorf("no environment group ruleset parameters found for environment group id %s", environmentGroupId)
	}

	return &environmentGroupRuleSet, nil
}

// UpdateEnvironmentGroupRuleSet updates a legacy rule set on the tenant-scoped host.
func (client *Client) UpdateEnvironmentGroupRuleSet(ctx context.Context, tenantId, environmentGroupId string, newEnvironmentGroupRuleSet EnvironmentGroupRuleSetValueSetDto) (*EnvironmentGroupRuleSetValueSetDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.tenantHost(tenantId),
		Path:   fmt.Sprintf("/governance/ruleSets/%s", environmentGroupId),
	}

	values := url.Values{}
	values.Add("api-version", "2021-10-01-preview")
	apiUrl.RawQuery = values.Encode()

	environmentGroupRuleSet := EnvironmentGroupRuleSetValueSetDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "PUT", apiUrl.String(), nil, newEnvironmentGroupRuleSet, []int{http.StatusOK}, &environmentGroupRuleSet)

	if err != nil {
		return nil, fmt.Errorf("failed to update environment group rule set: %w", err)
	}

	return &environmentGroupRuleSet, nil
}

// DeleteEnvironmentGroupRuleSet deletes a legacy rule set on the tenant-scoped host.
func (client *Client) DeleteEnvironmentGroupRuleSet(ctx context.Context, tenantId, ruleSetId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.tenantHost(tenantId),
		Path:   fmt.Sprintf("/governance/ruleSets/%s", ruleSetId),
	}

	values := url.Values{}
	values.Add("api-version", "2021-10-01-preview")
	apiUrl.RawQuery = values.Encode()

	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK}, nil)
	if err != nil {
		return fmt.Errorf("failed to delete environment group rule set: %w", err)
	}

	return nil
}

// policyUrl builds a rule-based policies URL. These endpoints live on the flat Power Platform
// host rather than the tenant-specific host used by the legacy rule sets endpoints.
func (client *Client) policyUrl(path string, extraQuery url.Values) string {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   path,
	}

	values := url.Values{}
	values.Add("api-version", POLICY_API_VERSION)
	for k, vs := range extraQuery {
		for _, v := range vs {
			values.Add(k, v)
		}
	}
	apiUrl.RawQuery = values.Encode()

	return apiUrl.String()
}

// CreatePolicy creates a rule-based policy.
func (client *Client) CreatePolicy(ctx context.Context, request RuleBasedPolicyRequestDto) (*RuleBasedPolicyDto, error) {
	policy := RuleBasedPolicyDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", client.policyUrl("/governance/ruleBasedPolicies", nil), nil, request, []int{http.StatusOK, http.StatusCreated}, &policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule-based policy: %w", err)
	}

	return &policy, nil
}

// GetPolicy reads a rule-based policy by id.
func (client *Client) GetPolicy(ctx context.Context, policyId string) (*RuleBasedPolicyDto, error) {
	policy := RuleBasedPolicyDto{}
	resp, err := client.Api.Execute(ctx, client.scopes(), "GET", client.policyUrl(fmt.Sprintf("/governance/ruleBasedPolicies/%s", policyId), nil), nil, nil, []int{http.StatusOK, http.StatusNotFound}, &policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule-based policy: %w", err)
	}

	if resp.HttpResponse.StatusCode == http.StatusNotFound {
		return nil, customerrors.WrapIntoProviderError(
			fmt.Errorf("rule-based policy '%s' not found", policyId),
			constants.ERROR_OBJECT_NOT_FOUND,
			fmt.Sprintf("rule-based policy '%s' not found", policyId),
		)
	}

	return &policy, nil
}

// UpdatePolicy replaces a rule-based policy.
func (client *Client) UpdatePolicy(ctx context.Context, policyId string, request RuleBasedPolicyRequestDto) (*RuleBasedPolicyDto, error) {
	policy := RuleBasedPolicyDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "PUT", client.policyUrl(fmt.Sprintf("/governance/ruleBasedPolicies/%s", policyId), nil), nil, request, []int{http.StatusOK}, &policy)
	if err != nil {
		return nil, fmt.Errorf("failed to update rule-based policy: %w", err)
	}

	return &policy, nil
}

// DeletePolicy deletes a rule-based policy.
func (client *Client) DeletePolicy(ctx context.Context, policyId string) error {
	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", client.policyUrl(fmt.Sprintf("/governance/ruleBasedPolicies/%s", policyId), nil), nil, nil, []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound}, nil)
	if err != nil {
		return fmt.Errorf("failed to delete rule-based policy: %w", err)
	}

	return nil
}

// DeleteEnvironmentGroupAssignment removes the assignment of a rule-based
// policy to an environment group.
func (client *Client) DeleteEnvironmentGroupAssignment(ctx context.Context, policyId, groupId string) error {
	path := fmt.Sprintf("/governance/ruleBasedPolicies/%s/environmentGroups/%s/assignments", policyId, groupId)
	_, err := client.Api.Execute(ctx, client.scopes(), "DELETE", client.policyUrl(path, nil), nil, nil, []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound}, nil)
	if err != nil {
		return fmt.Errorf("failed to delete environment group assignment: %w", err)
	}

	return nil
}

// RemoveRuleFromPolicy removes a single rule set from a rule-based policy.
func (client *Client) RemoveRuleFromPolicy(ctx context.Context, policyId, policyName, ruleSetId, version string) error {
	request := RemoveRuleRequestDto{
		Name:     policyName,
		RuleSets: []RemoveRuleSetDto{{Id: ruleSetId, Version: version}},
	}

	_, err := client.Api.Execute(ctx, client.scopes(), "PATCH", client.policyUrl(fmt.Sprintf("/governance/ruleBasedPolicies/%s/removeRule", policyId), nil), nil, request, []int{http.StatusOK, http.StatusNoContent}, nil)
	if err != nil {
		return fmt.Errorf("failed to remove rule %s from policy %s: %w", ruleSetId, policyId, err)
	}

	return nil
}

// CreateEnvironmentGroupAssignment assigns a rule-based policy to an
// environment group.
func (client *Client) CreateEnvironmentGroupAssignment(ctx context.Context, policyId, groupId string) (*RuleAssignmentDto, error) {
	assignment := RuleAssignmentDto{}
	path := fmt.Sprintf("/governance/ruleBasedPolicies/%s/environmentGroups/%s/assignments", policyId, groupId)
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", client.policyUrl(path, nil), nil, PolicyAssignmentRequestDto{AssignmentOverrides: []any{}}, []int{http.StatusOK, http.StatusCreated}, &assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment group assignment: %w", err)
	}

	return &assignment, nil
}

// ListAssignmentsByEnvironmentGroup lists the rule-based policy assignments of
// an environment group.
func (client *Client) ListAssignmentsByEnvironmentGroup(ctx context.Context, groupId string) ([]RuleAssignmentDto, error) {
	response := RuleAssignmentsResponseDto{}
	path := fmt.Sprintf("/governance/ruleBasedPolicies/environmentGroups/%s/assignments", groupId)
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", client.policyUrl(path, url.Values{"includeRuleSetCounts": []string{"true"}}), nil, nil, []int{http.StatusOK}, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments by environment group: %w", err)
	}

	return response.Value, nil
}
