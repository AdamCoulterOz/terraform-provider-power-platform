// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package athena

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnitGetOrganizations_RequestShape pins the tenant organizations
// listing: GET on the admin center host without query parameters, decoding a
// top-level JSON array.
func TestUnitGetOrganizations_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	expectedUrl := "https://api.admin.powerplatform.microsoft.com/api/tenants/mytenant/organizations"
	gotUrl := ""
	httpmock.RegisterResponder(http.MethodGet, expectedUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, `[
				{
					"tenantId": "10000000-0000-0000-0000-000000000000",
					"name": "Contoso Org",
					"id": "20000000-0000-0000-0000-000000000000",
					"crmGeo": "NA",
					"relationType": "Owned",
					"organizationType": 1,
					"url": "https://contoso.crm.dynamics.com"
				}
			]`), nil
		})

	organizations, err := client.GetOrganizations(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, gotUrl)
	require.Len(t, organizations, 1)
	assert.Equal(t, "10000000-0000-0000-0000-000000000000", organizations[0].TenantId)
	assert.Equal(t, "Contoso Org", organizations[0].Name)
	assert.Equal(t, "20000000-0000-0000-0000-000000000000", organizations[0].Id)
	assert.Equal(t, "NA", organizations[0].CrmGeo)
	assert.Equal(t, "Owned", organizations[0].RelationType)
	assert.Equal(t, 1, organizations[0].OrganizationType)
	assert.Equal(t, "https://contoso.crm.dynamics.com", organizations[0].Url)
}

// TestUnitGetOrganizations_UnexpectedStatus pins that a status outside the
// accepted [200] surfaces the customerrors unexpected-status error wrapped in
// the operation's context.
func TestUnitGetOrganizations_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "https://api.admin.powerplatform.microsoft.com/api/tenants/mytenant/organizations",
		httpmock.NewStringResponder(http.StatusForbidden, "no access"))

	organizations, err := client.GetOrganizations(context.Background())

	require.Error(t, err)
	assert.Nil(t, organizations)
	assert.ErrorContains(t, err, "failed to execute API call for organizations:")
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusForbidden, statusErr.StatusCode)
}

// TestUnitGetOrganizations_RetriesTransientStatus pins that the operation
// goes through Execute's transient-status retry (instant in TestMode).
func TestUnitGetOrganizations_RetriesTransientStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, "https://api.admin.powerplatform.microsoft.com/api/tenants/mytenant/organizations",
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusInternalServerError, "oops"), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, `[]`), nil
		})

	organizations, err := client.GetOrganizations(context.Background())

	require.NoError(t, err)
	assert.Empty(t, organizations)
	assert.Equal(t, 2, attempts, "a transient 500 must be retried")
}

// TestUnitEnableFeature_RequestShape pins the feature enablement call: POST
// on the admin center host with the geo as query parameter and no request
// body.
func TestUnitEnableFeature_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	expectedUrl := "https://api.admin.powerplatform.microsoft.com/api/environments/20000000-0000-0000-0000-000000000000/features/October2025Update/enable?geo=NA"
	gotUrl := ""
	var gotBody []byte
	httpmock.RegisterResponder(http.MethodPost, expectedUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			if req.Body != nil {
				body, readErr := io.ReadAll(req.Body)
				if readErr != nil {
					return nil, readErr
				}
				gotBody = body
			}
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.EnableFeature(context.Background(), "20000000-0000-0000-0000-000000000000", "NA", "October2025Update")

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, gotUrl)
	assert.Empty(t, gotBody, "feature enablement must not send a request body")
}

// TestUnitEnableFeature_UnexpectedStatus pins that a status outside the
// accepted [200] surfaces the customerrors unexpected-status error wrapped in
// the operation's feature-update context.
func TestUnitEnableFeature_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodPost, "https://api.admin.powerplatform.microsoft.com/api/environments/org-1/features/October2025Update/enable",
		httpmock.NewStringResponder(http.StatusBadRequest, "bad geo"))

	err := client.EnableFeature(context.Background(), "org-1", "XX", "October2025Update")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to update feature October2025Update:")
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
}

// TestUnitGetFeature_ReturnsMatchingFeature pins the feature read: GET on the
// admin center host with the geo as query parameter, returning the feature
// whose featureName matches.
func TestUnitGetFeature_ReturnsMatchingFeature(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	expectedUrl := "https://api.admin.powerplatform.microsoft.com/api/environments/20000000-0000-0000-0000-000000000000/features?geo=NA"
	gotUrl := ""
	httpmock.RegisterResponder(http.MethodGet, expectedUrl,
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, `{
				"values": [
					{
						"featureName": "April2025Update",
						"displayName": "2025 release wave 1",
						"canBeReset": false,
						"enabled": true,
						"isAllowed": true,
						"state": "on",
						"appsUpgradeState": "Completed"
					},
					{
						"featureName": "October2025Update",
						"displayName": "2025 release wave 2",
						"canBeReset": true,
						"enabled": false,
						"isAllowed": true,
						"notBefore": "2025-10-01T00:00:00Z",
						"notAfter": "2026-03-31T00:00:00Z",
						"minVersion": "9.2.0.0",
						"maxVersion": "9.2.99.99",
						"state": "off",
						"appsUpgradeState": "NotStarted"
					}
				]
			}`), nil
		})

	feature, err := client.GetFeature(context.Background(), "20000000-0000-0000-0000-000000000000", "NA", "October2025Update")

	require.NoError(t, err)
	assert.Equal(t, expectedUrl, gotUrl)
	require.NotNil(t, feature)
	assert.Equal(t, FeatureDto{
		FeatureName:      "October2025Update",
		DisplayName:      "2025 release wave 2",
		CanBeReset:       true,
		Enabled:          false,
		IsAllowed:        true,
		NotBefore:        "2025-10-01T00:00:00Z",
		NotAfter:         "2026-03-31T00:00:00Z",
		MinVersion:       "9.2.0.0",
		MaxVersion:       "9.2.99.99",
		State:            "off",
		AppsUpgradeState: "NotStarted",
	}, *feature)
}

// TestUnitGetFeature_ReturnsNilWhenAbsent pins that an unknown feature name
// yields neither a feature nor an error.
func TestUnitGetFeature_ReturnsNilWhenAbsent(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "https://api.admin.powerplatform.microsoft.com/api/environments/org-1/features?geo=NA",
		httpmock.NewStringResponder(http.StatusOK, `{"values": [{"featureName": "April2025Update"}]}`))

	feature, err := client.GetFeature(context.Background(), "org-1", "NA", "October2025Update")

	require.NoError(t, err)
	assert.Nil(t, feature)
}

// TestUnitGetFeature_UnexpectedStatus pins that a status outside the accepted
// [200] surfaces the customerrors unexpected-status error without additional
// context.
func TestUnitGetFeature_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := newTestClient(t)

	httpmock.RegisterResponder(http.MethodGet, "https://api.admin.powerplatform.microsoft.com/api/environments/org-1/features?geo=NA",
		httpmock.NewStringResponder(http.StatusNotFound, "no such organization"))

	feature, err := client.GetFeature(context.Background(), "org-1", "NA", "October2025Update")

	require.Error(t, err)
	assert.Nil(t, feature)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
	assert.Equal(t, statusErr.Error(), err.Error(), "the operation must not add context around the transport error")
}
