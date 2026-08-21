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

const testBillingPolicyId = "60000000-0000-0000-0000-00000000000f"

// licensingQuery pins the api-version every licensing operation must send.
func licensingQuery() url.Values {
	return url.Values{"api-version": []string{"2022-03-01-preview"}}
}

func TestUnitGetBillingPolicies_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies", licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyArrayDto{Value: []BillingPolicyDto{
				{Id: "pol-1", Status: "Enabled"},
				{Id: "pol-2", Status: "Disabled"},
			}})
		})

	policies, err := newTestClient(t).GetBillingPolicies(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, []BillingPolicyDto{{Id: "pol-1", Status: "Enabled"}, {Id: "pol-2", Status: "Disabled"}}, policies)
}

func TestUnitGetBillingPolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyDto{Id: testBillingPolicyId, Status: "Enabled"})
		})

	policy, err := newTestClient(t).GetBillingPolicy(context.Background(), testBillingPolicyId)

	assert.NoError(t, err)
	assert.Equal(t, testBillingPolicyId, policy.Id)
	assert.Equal(t, "Enabled", policy.Status)
}

func TestUnitGetBillingPolicy_NotFoundTranslation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"NotFound"}}`))

	policy, err := newTestClient(t).GetBillingPolicy(context.Background(), testBillingPolicyId)

	assert.Nil(t, policy)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, "Billing Policy with ID '"+testBillingPolicyId+"' not found")
}

func TestUnitCreateBillingPolicy_TerminalStatusSkipsPolling(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	policyToCreate := BillingPolicyCreateDto{
		Location: "europe",
		Name:     "billingpolicy1",
		Status:   "Enabled",
		BillingInstrument: BillingInstrumentDto{
			ResourceGroup:  "rg-1",
			SubscriptionId: "70000000-0000-0000-0000-000000000010",
		},
	}

	var gotBody BillingPolicyCreateDto
	// The create path uses a capitalized "BillingPolicies" segment.
	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/licensing/BillingPolicies", licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusCreated, BillingPolicyDto{Id: testBillingPolicyId, Status: "Enabled"})
		})

	policy, err := newTestClient(t).CreateBillingPolicy(context.Background(), policyToCreate)

	assert.NoError(t, err)
	assert.Equal(t, policyToCreate, gotBody)
	assert.Equal(t, "Enabled", policy.Status)
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "a terminal create response must not trigger status polling")
}

func TestUnitCreateBillingPolicy_WaitsForFinalStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/licensing/BillingPolicies", licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusCreated, BillingPolicyDto{Id: testBillingPolicyId, Status: "Provisioning"})
		})

	polls := 0
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			polls++
			status := "Provisioning"
			if polls >= 2 {
				status = "Enabled"
			}
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyDto{Id: testBillingPolicyId, Status: status})
		})

	policy, err := newTestClient(t).CreateBillingPolicy(context.Background(), BillingPolicyCreateDto{Name: "billingpolicy1"})

	assert.NoError(t, err)
	assert.Equal(t, 2, polls, "polling must continue until the policy reaches a terminal status")
	assert.Equal(t, "Enabled", policy.Status)
}

func TestUnitUpdateBillingPolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	policyToUpdate := BillingPolicyUpdateDto{Name: "renamed", Status: "Disabled"}

	var gotBody BillingPolicyUpdateDto
	httpmock.RegisterResponderWithQuery("PUT", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyDto{Id: testBillingPolicyId, Status: "Disabled"})
		})

	policy, err := newTestClient(t).UpdateBillingPolicy(context.Background(), testBillingPolicyId, policyToUpdate)

	assert.NoError(t, err)
	assert.Equal(t, policyToUpdate, gotBody)
	assert.Equal(t, "Disabled", policy.Status)
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "a terminal update response must not trigger status polling")
}

func TestUnitUpdateBillingPolicy_WaitsForFinalStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("PUT", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyDto{Id: testBillingPolicyId, Status: "Updating"})
		})

	polls := 0
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			polls++
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyDto{Id: testBillingPolicyId, Status: "Enabled"})
		})

	policy, err := newTestClient(t).UpdateBillingPolicy(context.Background(), testBillingPolicyId, BillingPolicyUpdateDto{Name: "renamed", Status: "Enabled"})

	assert.NoError(t, err)
	assert.Equal(t, 1, polls, "a non-terminal update response must trigger status polling")
	assert.Equal(t, "Enabled", policy.Status)
}

func TestUnitDeleteBillingPolicy_Statuses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"204 is accepted", http.StatusNoContent, false},
		{"200 is not an accepted status", http.StatusOK, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			// The delete path uses a capitalized "BillingPolicies" segment.
			httpmock.RegisterResponderWithQuery("DELETE", "https://api.powerplatform.com/licensing/BillingPolicies/"+testBillingPolicyId, licensingQuery(),
				httpmock.NewStringResponder(tc.status, ""))

			err := newTestClient(t).DeleteBillingPolicy(context.Background(), testBillingPolicyId)

			if tc.wantErr {
				var httpErr customerrors.UnexpectedHttpStatusCodeError
				assert.ErrorAs(t, err, &httpErr)
				assert.Equal(t, tc.status, httpErr.StatusCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnitGetEnvironmentsForBillingPolicy_LowercasesEnvironmentIds(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId+"/environments", licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyEnvironmentsArrayResponseDto{Value: []BillingPolicyEnvironmentsDto{
				{BillingPolicyId: testBillingPolicyId, EnvironmentId: "00000000-0000-0000-0000-0000000000AA"},
				{BillingPolicyId: testBillingPolicyId, EnvironmentId: "00000000-0000-0000-0000-0000000000BB"},
			}})
		})

	environments, err := newTestClient(t).GetEnvironmentsForBillingPolicy(context.Background(), testBillingPolicyId)

	assert.NoError(t, err)
	assert.Equal(t, []string{"00000000-0000-0000-0000-0000000000aa", "00000000-0000-0000-0000-0000000000bb"}, environments)
}

func TestUnitGetEnvironmentsForBillingPolicy_NotFoundTranslation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId+"/environments", licensingQuery(),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"NotFound"}}`))

	environments, err := newTestClient(t).GetEnvironmentsForBillingPolicy(context.Background(), testBillingPolicyId)

	assert.Nil(t, environments)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, "Billing Policy with ID '"+testBillingPolicyId+"' not found")
}

func TestUnitAddEnvironmentsToBillingPolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var gotBody BillingPolicyEnvironmentsArrayDto
	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId+"/environments/add", licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := newTestClient(t).AddEnvironmentsToBillingPolicy(context.Background(), testBillingPolicyId, []string{"00000000-0000-0000-0000-0000000000AA", "00000000-0000-0000-0000-0000000000bb"})

	assert.NoError(t, err)
	assert.Equal(t, BillingPolicyEnvironmentsArrayDto{EnvironmentIds: []string{"00000000-0000-0000-0000-0000000000aa", "00000000-0000-0000-0000-0000000000bb"}}, gotBody, "environment ids must be lowercased in the request body")
}

func TestUnitAddEnvironmentsToBillingPolicy_EmptyListSkipsRequest(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	err := newTestClient(t).AddEnvironmentsToBillingPolicy(context.Background(), testBillingPolicyId, []string{})

	assert.NoError(t, err)
	assert.Equal(t, 0, httpmock.GetTotalCallCount())
}

func TestUnitRemoveEnvironmentsToBillingPolicy_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var gotBody BillingPolicyEnvironmentsArrayDto
	httpmock.RegisterResponderWithQuery("POST", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId+"/environments/remove", licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				return nil, err
			}
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := newTestClient(t).RemoveEnvironmentsToBillingPolicy(context.Background(), testBillingPolicyId, []string{"00000000-0000-0000-0000-0000000000CC"})

	assert.NoError(t, err)
	assert.Equal(t, BillingPolicyEnvironmentsArrayDto{EnvironmentIds: []string{"00000000-0000-0000-0000-0000000000cc"}}, gotBody, "environment ids must be lowercased in the request body")
}

func TestUnitRemoveEnvironmentsToBillingPolicy_EmptyListSkipsRequest(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	err := newTestClient(t).RemoveEnvironmentsToBillingPolicy(context.Background(), testBillingPolicyId, nil)

	assert.NoError(t, err)
	assert.Equal(t, 0, httpmock.GetTotalCallCount())
}

func TestUnitDoWaitForFinalStatus_PollsUntilTerminal(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	polls := 0
	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		func(req *http.Request) (*http.Response, error) {
			polls++
			status := "Provisioning"
			if polls >= 3 {
				status = "Disabled"
			}
			return httpmock.NewJsonResponse(http.StatusOK, BillingPolicyDto{Id: testBillingPolicyId, Status: status})
		})

	policy, err := newTestClient(t).DoWaitForFinalStatus(context.Background(), &BillingPolicyDto{Id: testBillingPolicyId, Status: "Provisioning"})

	assert.NoError(t, err)
	assert.Equal(t, 3, polls, "TestMode sleeps are no-ops, so each poll issues one GET until a terminal status arrives")
	assert.Equal(t, "Disabled", policy.Status)
}

func TestUnitDoWaitForFinalStatus_PropagatesReadError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("GET", "https://api.powerplatform.com/licensing/billingPolicies/"+testBillingPolicyId, licensingQuery(),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"NotFound"}}`))

	policy, err := newTestClient(t).DoWaitForFinalStatus(context.Background(), &BillingPolicyDto{Id: testBillingPolicyId, Status: "Provisioning"})

	assert.Nil(t, policy)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
}
