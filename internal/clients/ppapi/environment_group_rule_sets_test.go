// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
)

const (
	testRuleSetId = "80000000-0000-0000-0000-000000000011"
	testPolicyId  = "90000000-0000-0000-0000-000000000012"
)

// legacyRuleSetsQuery pins the api-version of the legacy (tenant-scoped host)
// rule sets endpoints.
func legacyRuleSetsQuery() url.Values {
	return url.Values{"api-version": []string{"2021-10-01-preview"}}
}

// policiesQuery pins the api-version of the rule-based policies endpoints,
// which are served from the flat Power Platform host.
func policiesQuery() url.Values {
	return url.Values{"api-version": []string{"2024-10-01"}}
}

func testRuleSet() EnvironmentGroupRuleSetValueSetDto {
	return EnvironmentGroupRuleSetValueSetDto{
		Parameters: []*EnvironmentGroupRuleSetParameterDto{
			{
				Type:         SHARING,
				ResourceType: APP,
				Value:        []EnvironmentGroupRuleSetValueDto{{Id: IS_GROUP_SHARING_DISABLED, Value: "true"}},
			},
		},
	}
}

func TestUnitGetEnvironmentGroupRuleSet_UsesTenantHost(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	ruleSetId := testRuleSetId
	httpmock.RegisterResponderWithQuery("GET", "https://"+testTenantHost+"/governance/environmentGroups/"+testEnvironmentGroupId+"/ruleSets", legacyRuleSetsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, EnvironmentGroupRuleSetDto{Value: []EnvironmentGroupRuleSetValueSetDto{
				{Id: &ruleSetId, Parameters: testRuleSet().Parameters},
			}})
		})

	ruleSet, err := newTestClient(t).GetEnvironmentGroupRuleSet(context.Background(), testTenantId, testEnvironmentGroupId)

	assert.NoError(t, err)
	assert.Equal(t, testRuleSetId, *ruleSet.Id)
	assert.Len(t, ruleSet.Parameters, 1)
}

func TestUnitGetEnvironmentGroupRuleSet_NoContentMeansNotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://"+testTenantHost+"/governance/environmentGroups/"+testEnvironmentGroupId+"/ruleSets", legacyRuleSetsQuery(),
		httpmock.NewStringResponder(http.StatusNoContent, ""))

	ruleSet, err := newTestClient(t).GetEnvironmentGroupRuleSet(context.Background(), testTenantId, testEnvironmentGroupId)

	assert.Nil(t, ruleSet)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, "rule set '"+testEnvironmentGroupId+"' not found")
}

func TestUnitGetEnvironmentGroupRuleSet_EmptyValueIsAnError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://"+testTenantHost+"/governance/environmentGroups/"+testEnvironmentGroupId+"/ruleSets", legacyRuleSetsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, EnvironmentGroupRuleSetDto{Value: []EnvironmentGroupRuleSetValueSetDto{}})
		})

	ruleSet, err := newTestClient(t).GetEnvironmentGroupRuleSet(context.Background(), testTenantId, testEnvironmentGroupId)

	assert.Nil(t, ruleSet)
	assert.ErrorContains(t, err, "no environment group ruleset found for environment group id "+testEnvironmentGroupId)
}

func TestUnitCreateEnvironmentGroupRuleSet_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	newRuleSet := testRuleSet()
	var gotBody EnvironmentGroupRuleSetValueSetDto
	httpmock.RegisterResponderWithQuery("POST", "https://"+testTenantHost+"/governance/environmentGroups/"+testEnvironmentGroupId+"/ruleSets", legacyRuleSetsQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			ruleSetId := testRuleSetId
			created := testRuleSet()
			created.Id = &ruleSetId
			return httpmock.NewJsonResponse(http.StatusCreated, created)
		})

	ruleSet, err := newTestClient(t).CreateEnvironmentGroupRuleSet(context.Background(), testTenantId, testEnvironmentGroupId, newRuleSet)

	assert.NoError(t, err)
	assert.Equal(t, newRuleSet, gotBody)
	assert.Equal(t, testRuleSetId, *ruleSet.Id)
}

func TestUnitCreateEnvironmentGroupRuleSet_EmptyParametersIsAnError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("POST", "https://"+testTenantHost+"/governance/environmentGroups/"+testEnvironmentGroupId+"/ruleSets", legacyRuleSetsQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusCreated, EnvironmentGroupRuleSetValueSetDto{})
		})

	ruleSet, err := newTestClient(t).CreateEnvironmentGroupRuleSet(context.Background(), testTenantId, testEnvironmentGroupId, testRuleSet())

	assert.Nil(t, ruleSet)
	assert.ErrorContains(t, err, "no environment group ruleset parameters found for environment group id "+testEnvironmentGroupId)
}

func TestUnitUpdateEnvironmentGroupRuleSet_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	newRuleSet := testRuleSet()
	var gotBody EnvironmentGroupRuleSetValueSetDto
	// Updates address the rule set id directly under /governance/ruleSets.
	httpmock.RegisterResponderWithQuery("PUT", "https://"+testTenantHost+"/governance/ruleSets/"+testRuleSetId, legacyRuleSetsQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusOK, newRuleSet)
		})

	ruleSet, err := newTestClient(t).UpdateEnvironmentGroupRuleSet(context.Background(), testTenantId, testRuleSetId, newRuleSet)

	assert.NoError(t, err)
	assert.Equal(t, newRuleSet, gotBody)
	assert.Equal(t, newRuleSet, *ruleSet)
}

func TestUnitDeleteEnvironmentGroupRuleSet_Statuses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"200 is accepted", http.StatusOK, false},
		{"204 is not an accepted status", http.StatusNoContent, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponderWithQuery("DELETE", "https://"+testTenantHost+"/governance/ruleSets/"+testRuleSetId, legacyRuleSetsQuery(),
				httpmock.NewStringResponder(tc.status, ""))

			err := newTestClient(t).DeleteEnvironmentGroupRuleSet(context.Background(), testTenantId, testRuleSetId)

			if tc.wantErr {
				assert.ErrorContains(t, err, "failed to delete environment group rule set")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnitCreatePolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	request := RuleBasedPolicyRequestDto{
		Name: "policy-1",
		RuleSets: []PolicyRuleSetDto{
			{Id: ADVANCED_CONNECTOR_POLICIES_ONLY_ID, Version: ADVANCED_CONNECTOR_POLICIES_VERSION, Inputs: map[string]any{ENABLE_ADVANCED_CONNECTOR_POLICIES_KEY: true}},
		},
	}

	var gotBody RuleBasedPolicyRequestDto
	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/governance/ruleBasedPolicies", policiesQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusCreated, RuleBasedPolicyDto{Id: testPolicyId, Name: "policy-1"})
		})

	policy, err := newTestClient(t).CreatePolicy(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, request, gotBody)
	assert.Equal(t, testPolicyId, policy.Id)
}

func TestUnitGetPolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/governance/ruleBasedPolicies/"+testPolicyId, policiesQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, RuleBasedPolicyDto{Id: testPolicyId, Name: "policy-1"})
		})

	policy, err := newTestClient(t).GetPolicy(context.Background(), testPolicyId)

	assert.NoError(t, err)
	assert.Equal(t, testPolicyId, policy.Id)
}

func TestUnitGetPolicy_NotFoundTranslation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// 404 is an accepted status for GetPolicy and is translated into the
	// provider's not-found error instead of an unexpected-status error.
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/governance/ruleBasedPolicies/"+testPolicyId, policiesQuery(),
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	policy, err := newTestClient(t).GetPolicy(context.Background(), testPolicyId)

	assert.Nil(t, policy)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, "rule-based policy '"+testPolicyId+"' not found")
}

func TestUnitUpdatePolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	request := RuleBasedPolicyRequestDto{Name: "policy-1-renamed", RuleSets: []PolicyRuleSetDto{}}

	var gotBody RuleBasedPolicyRequestDto
	httpmock.RegisterResponderWithQuery("PUT", "https://api.powerplatform.com/governance/ruleBasedPolicies/"+testPolicyId, policiesQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusOK, RuleBasedPolicyDto{Id: testPolicyId, Name: "policy-1-renamed"})
		})

	policy, err := newTestClient(t).UpdatePolicy(context.Background(), testPolicyId, request)

	assert.NoError(t, err)
	assert.Equal(t, request, gotBody)
	assert.Equal(t, "policy-1-renamed", policy.Name)
}

func TestUnitDeletePolicy_Statuses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"200 is accepted", http.StatusOK, false},
		{"204 is accepted", http.StatusNoContent, false},
		{"404 is accepted (already gone)", http.StatusNotFound, false},
		{"400 is not an accepted status", http.StatusBadRequest, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponderWithQuery("DELETE", "https://api.powerplatform.com/governance/ruleBasedPolicies/"+testPolicyId, policiesQuery(),
				httpmock.NewStringResponder(tc.status, ""))

			err := newTestClient(t).DeletePolicy(context.Background(), testPolicyId)

			if tc.wantErr {
				assert.ErrorContains(t, err, "failed to delete rule-based policy")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnitRemoveRuleFromPolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var gotBody RemoveRuleRequestDto
	httpmock.RegisterResponderWithQuery("PATCH", "https://api.powerplatform.com/governance/ruleBasedPolicies/"+testPolicyId+"/removeRule", policiesQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	err := newTestClient(t).RemoveRuleFromPolicy(context.Background(), testPolicyId, "policy-1", CONTENT_SECURITY_POLICY_ID, CONTENT_SECURITY_POLICY_VERSION)

	assert.NoError(t, err)
	assert.Equal(t, RemoveRuleRequestDto{
		Name:     "policy-1",
		RuleSets: []RemoveRuleSetDto{{Id: CONTENT_SECURITY_POLICY_ID, Version: CONTENT_SECURITY_POLICY_VERSION}},
	}, gotBody)
}

func TestUnitCreateEnvironmentGroupAssignment_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var gotBody PolicyAssignmentRequestDto
	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/governance/ruleBasedPolicies/"+testPolicyId+"/environmentGroups/"+testEnvironmentGroupId+"/assignments", policiesQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusCreated, RuleAssignmentDto{PolicyId: testPolicyId, ResourceId: testEnvironmentGroupId})
		})

	assignment, err := newTestClient(t).CreateEnvironmentGroupAssignment(context.Background(), testPolicyId, testEnvironmentGroupId)

	assert.NoError(t, err)
	assert.Equal(t, PolicyAssignmentRequestDto{AssignmentOverrides: []any{}}, gotBody)
	assert.Equal(t, testPolicyId, assignment.PolicyId)
	assert.Equal(t, testEnvironmentGroupId, assignment.ResourceId)
}

func TestUnitDeleteEnvironmentGroupAssignment_Statuses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"204 is accepted", http.StatusNoContent, false},
		{"404 is accepted (already gone)", http.StatusNotFound, false},
		{"409 is not an accepted status", http.StatusConflict, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponderWithQuery("DELETE", "https://api.powerplatform.com/governance/ruleBasedPolicies/"+testPolicyId+"/environmentGroups/"+testEnvironmentGroupId+"/assignments", policiesQuery(),
				httpmock.NewStringResponder(tc.status, ""))

			err := newTestClient(t).DeleteEnvironmentGroupAssignment(context.Background(), testPolicyId, testEnvironmentGroupId)

			if tc.wantErr {
				assert.ErrorContains(t, err, "failed to delete environment group assignment")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnitListAssignmentsByEnvironmentGroup_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	query := policiesQuery()
	query.Add("includeRuleSetCounts", "true")
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/governance/ruleBasedPolicies/environmentGroups/"+testEnvironmentGroupId+"/assignments", query,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, RuleAssignmentsResponseDto{Value: []RuleAssignmentDto{
				{PolicyId: testPolicyId, ResourceId: testEnvironmentGroupId, RuleSetCount: 2},
			}})
		})

	assignments, err := newTestClient(t).ListAssignmentsByEnvironmentGroup(context.Background(), testEnvironmentGroupId)

	assert.NoError(t, err)
	assert.Equal(t, []RuleAssignmentDto{{PolicyId: testPolicyId, ResourceId: testEnvironmentGroupId, RuleSetCount: 2}}, assignments)
}
