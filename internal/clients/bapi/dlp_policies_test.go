// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dlpPoliciesUrl  = "https://api.bap.microsoft.com/providers/PowerPlatform.Governance/v2/policies"
	testDlpPolicyId = "55555555-5555-5555-5555-555555555555"
)

func dlpPolicyUrl(name string) string {
	return dlpPoliciesUrl + "/" + name
}

func dlpPolicyBody(name, displayName string) string {
	return fmt.Sprintf(`{"policyDefinition":{"name":"%s","displayName":"%s","defaultConnectorsClassification":"Blocked","environmentType":"AllEnvironments","environments":[],"connectorGroups":[]},"customConnectorUrlPatternsDefinition":{"rules":[]}}`, name, displayName)
}

func TestUnitGetPolicy_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotRawQuery string
	httpmock.RegisterResponder(http.MethodGet, dlpPolicyUrl(testDlpPolicyId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotRawQuery = req.URL.RawQuery
			return httpmock.NewStringResponse(http.StatusOK, dlpPolicyBody(testDlpPolicyId, "Policy One")), nil
		})

	policy, err := client.GetPolicy(context.Background(), testDlpPolicyId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Empty(t, gotRawQuery, "the DLP governance endpoints are not versioned with an api-version query")
	assert.Equal(t, testDlpPolicyId, policy.PolicyDefinition.Name)
	assert.Equal(t, "Policy One", policy.PolicyDefinition.DisplayName)
}

func TestUnitGetPolicy_NotFoundIsTranslated(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, dlpPolicyUrl(testDlpPolicyId),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"PolicyNotFound"}}`))

	policy, err := client.GetPolicy(context.Background(), testDlpPolicyId)

	assert.Nil(t, policy)
	require.Error(t, err)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, fmt.Sprintf("DLP Policy '%s' not found", testDlpPolicyId))
}

func TestUnitGetPolicies_ReadsEachPolicyDefinition(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	listBody := fmt.Sprintf(`{"value":[%s,%s]}`, dlpPolicyBody("policy-1", "One"), dlpPolicyBody("policy-2", "Two"))
	httpmock.RegisterResponder(http.MethodGet, dlpPoliciesUrl,
		httpmock.NewStringResponder(http.StatusOK, listBody))
	httpmock.RegisterResponder(http.MethodGet, dlpPolicyUrl("policy-1"),
		httpmock.NewStringResponder(http.StatusOK, dlpPolicyBody("policy-1", "One Full")))
	httpmock.RegisterResponder(http.MethodGet, dlpPolicyUrl("policy-2"),
		httpmock.NewStringResponder(http.StatusOK, dlpPolicyBody("policy-2", "Two Full")))

	policies, err := client.GetPolicies(context.Background())

	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, "One Full", policies[0].PolicyDefinition.DisplayName, "each listed policy must be re-read for its full definition")
	assert.Equal(t, "Two Full", policies[1].PolicyDefinition.DisplayName)
	assert.Equal(t, 3, httpmock.GetTotalCallCount(), "one list call plus one read per policy")
}

func TestUnitCreatePolicy_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, dlpPoliciesUrl,
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusCreated, dlpPolicyBody(testDlpPolicyId, "Created")), nil
		})

	created, err := client.CreatePolicy(context.Background(), DlpPolicyDto{
		PolicyDefinition: DlpPolicyDefinitionDto{
			DisplayName:                     "Created",
			DefaultConnectorsClassification: "Blocked",
			EnvironmentType:                 "AllEnvironments",
			Environments:                    []DlpEnvironmentDto{},
			ConnectorGroups:                 []DlpConnectorGroupsDto{},
		},
		CustomConnectorUrlPatternsDefinition: DlpConnectorUrlPatternsDefinitionDto{Rules: []DlpConnectorUrlPatternsRuleDto{}},
	})

	require.NoError(t, err)
	assert.JSONEq(t, `{"policyDefinition":{"displayName":"Created","defaultConnectorsClassification":"Blocked","environmentType":"AllEnvironments","environments":[],"connectorGroups":[],"createdBy":{"displayName":""},"lastModifiedBy":{"displayName":""}},"customConnectorUrlPatternsDefinition":{"rules":[]}}`, gotBody)
	assert.Equal(t, testDlpPolicyId, created.PolicyDefinition.Name)
}

func TestUnitCreatePolicy_OnlyCreatedStatusIsAccepted(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPost, dlpPoliciesUrl,
		httpmock.NewStringResponder(http.StatusOK, dlpPolicyBody(testDlpPolicyId, "Created")))

	created, err := client.CreatePolicy(context.Background(), DlpPolicyDto{})

	assert.Nil(t, created)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusOK, statusErr.StatusCode)
	assert.Equal(t, []int{http.StatusCreated}, statusErr.ExpectedStatusCodes)
}

func TestUnitUpdatePolicy_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod string
	httpmock.RegisterResponder(http.MethodPatch, dlpPolicyUrl(testDlpPolicyId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			return httpmock.NewStringResponse(http.StatusOK, dlpPolicyBody(testDlpPolicyId, "Updated")), nil
		})

	updated, err := client.UpdatePolicy(context.Background(), DlpPolicyDto{
		PolicyDefinition: DlpPolicyDefinitionDto{Name: testDlpPolicyId, DisplayName: "Updated"},
	})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "Updated", updated.PolicyDefinition.DisplayName)
}

func TestUnitDeletePolicy_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod string
	httpmock.RegisterResponder(http.MethodDelete, dlpPolicyUrl(testDlpPolicyId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.DeletePolicy(context.Background(), testDlpPolicyId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
}
