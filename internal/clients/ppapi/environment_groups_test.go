// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUnitRemoveEnvironmentFromEnvironmentGroup_RequestShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// The remove operation is a POST to the tenant-scoped host and only 202 is accepted.
	httpmock.RegisterResponderWithQuery("POST",
		"https://"+testTenantHost+"/environmentmanagement/environmentGroups/"+testEnvironmentGroupId+"/removeEnvironment/"+testEnvironmentId,
		url.Values{"api-version": []string{"1"}},
		httpmock.NewStringResponder(http.StatusAccepted, ""))

	err := newTestClient(t).RemoveEnvironmentFromEnvironmentGroup(context.Background(), testTenantId, testEnvironmentGroupId, testEnvironmentId)

	assert.NoError(t, err)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

func TestUnitRemoveEnvironmentFromEnvironmentGroup_UnexpectedStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponderWithQuery("POST",
		"https://"+testTenantHost+"/environmentmanagement/environmentGroups/"+testEnvironmentGroupId+"/removeEnvironment/"+testEnvironmentId,
		url.Values{"api-version": []string{"1"}},
		httpmock.NewStringResponder(http.StatusOK, ""))

	err := newTestClient(t).RemoveEnvironmentFromEnvironmentGroup(context.Background(), testTenantId, testEnvironmentGroupId, testEnvironmentId)

	assert.ErrorContains(t, err, "failed to remove environment from environment group")
}
