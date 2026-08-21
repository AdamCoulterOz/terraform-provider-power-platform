// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package advisor

import (
	"context"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetSolutionCheckerRules reads the solution checker ruleset from an
// environment's PowerApps Advisor endpoint. The advisor host is resolved by
// callers from the environment's runtime endpoints via the bapi client.
func (client *Client) GetSolutionCheckerRules(ctx context.Context, advisorHost string) ([]RuleDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   advisorHost,
		Path:   "/api/rule",
	}
	values := url.Values{}
	values.Add("api-version", "2.0")
	// Currently, the ruleset is always the same for all regions
	values.Add("ruleset", constants.SOLUTION_CHECKER_RULESET_ID)
	apiUrl.RawQuery = values.Encode()

	var rules []RuleDto
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &rules)
	if err != nil {
		return nil, err
	}

	return rules, nil
}
