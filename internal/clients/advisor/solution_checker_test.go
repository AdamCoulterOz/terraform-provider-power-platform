// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package advisor

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnitGetSolutionCheckerRules_RequestShape pins the ruleset read: GET
// /api/rule on the caller-provided per-environment advisor host, with
// api-version 2.0 and the fixed solution checker ruleset id as query
// parameters, decoding a top-level JSON array of rules.
func TestUnitGetSolutionCheckerRules_RequestShape(t *testing.T) {
	tests := []struct {
		name        string
		advisorHost string
		expectedUrl string
	}{
		{
			name:        "unitedstates advisor endpoint",
			advisorHost: "unitedstates.api.advisor.powerapps.com",
			expectedUrl: "https://unitedstates.api.advisor.powerapps.com/api/rule?api-version=2.0&ruleset=0ad12346-e108-40b8-a956-9a8f95ea18c9",
		},
		{
			name:        "europe advisor endpoint",
			advisorHost: "europe.api.advisor.powerapps.com",
			expectedUrl: "https://europe.api.advisor.powerapps.com/api/rule?api-version=2.0&ruleset=0ad12346-e108-40b8-a956-9a8f95ea18c9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			client := newTestClient(t)

			gotUrl := ""
			httpmock.RegisterResponder(http.MethodGet, tt.expectedUrl,
				func(req *http.Request) (*http.Response, error) {
					gotUrl = req.URL.String()
					return httpmock.NewStringResponse(http.StatusOK, `[
						{
							"code": "meta-remove-dup-reg",
							"description": "Checks for duplicate plug-in registrations",
							"summary": "Duplicate plug-in registration",
							"howToFix": "Remove the duplicate registration",
							"guidanceUrl": "https://learn.microsoft.com/power-apps/meta-remove-dup-reg",
							"componentType": 1,
							"primaryCategory": 2,
							"include": true,
							"severity": 3
						}
					]`), nil
				})

			rules, err := client.GetSolutionCheckerRules(context.Background(), tt.advisorHost)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedUrl, gotUrl)
			require.Len(t, rules, 1)
			assert.Equal(t, RuleDto{
				Code:            "meta-remove-dup-reg",
				Description:     "Checks for duplicate plug-in registrations",
				Summary:         "Duplicate plug-in registration",
				HowToFix:        "Remove the duplicate registration",
				GuidanceUrl:     "https://learn.microsoft.com/power-apps/meta-remove-dup-reg",
				ComponentType:   1,
				PrimaryCategory: 2,
				Include:         true,
				Severity:        3,
			}, rules[0])
		})
	}
}

// TestUnitGetSolutionCheckerRules_UnexpectedStatus pins that a status outside
// the accepted [200] surfaces the customerrors unexpected-status error
// without additional context.
func TestUnitGetSolutionCheckerRules_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://unitedstates.api.advisor.powerapps.com/api/rule",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusNotFound, "no rules here"), nil
		})

	rules, err := client.GetSolutionCheckerRules(context.Background(), "unitedstates.api.advisor.powerapps.com")

	require.Error(t, err)
	assert.Nil(t, rules)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
	assert.Equal(t, statusErr.Error(), err.Error(), "the operation must not add context around the transport error")
	assert.Equal(t, 1, attempts, "a non-retryable status must not be replayed")
}

// TestUnitGetSolutionCheckerRules_RetriesTransientStatus pins that the
// operation goes through Execute's transient-status retry (instant in
// TestMode).
func TestUnitGetSolutionCheckerRules_RetriesTransientStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://unitedstates.api.advisor.powerapps.com/api/rule",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusTooManyRequests, "slow down"), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, `[]`), nil
		})

	rules, err := client.GetSolutionCheckerRules(context.Background(), "unitedstates.api.advisor.powerapps.com")

	require.NoError(t, err)
	assert.Empty(t, rules)
	assert.Equal(t, 2, attempts, "a transient 429 must be retried")
}
