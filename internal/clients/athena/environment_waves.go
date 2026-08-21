// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package athena

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetOrganizations lists the organizations of the tenant behind the current
// credentials.
func (client *Client) GetOrganizations(ctx context.Context) (OrganizationsArrayDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/api/tenants/mytenant/organizations",
	}

	organizations := OrganizationsArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &organizations)
	if err != nil {
		return nil, fmt.Errorf("failed to execute API call for organizations: %w", err)
	}

	return organizations, nil
}

// EnableFeature starts enablement of a wave feature on an organization.
func (client *Client) EnableFeature(ctx context.Context, organizationId, geo, featureName string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/api/environments/%s/features/%s/enable", organizationId, featureName),
	}

	values := url.Values{}
	values.Add("geo", geo)
	apiUrl.RawQuery = values.Encode()

	urlString := apiUrl.String()

	_, err := client.Api.Execute(ctx, client.scopes(), "POST", urlString, nil, nil, []int{http.StatusOK}, nil)
	if err != nil {
		return fmt.Errorf("failed to update feature %s: %w", featureName, err)
	}
	return nil
}

// GetFeature reads a single wave feature of an organization by name. It
// returns nil when the feature does not exist.
func (client *Client) GetFeature(ctx context.Context, organizationId, geo, featureName string) (*FeatureDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/api/environments/%s/features", organizationId),
	}

	values := url.Values{}
	values.Add("geo", geo)
	apiUrl.RawQuery = values.Encode()

	features := FeaturesArrayDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &features)
	if err != nil {
		return nil, err
	}

	for _, feature := range features.Values {
		if feature.FeatureName == featureName {
			return &feature, nil
		}
	}

	return nil, nil
}
