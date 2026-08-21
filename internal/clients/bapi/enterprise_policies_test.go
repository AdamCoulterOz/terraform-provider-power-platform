// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitLinkAndUnLinkEnterprisePolicy_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	tests := []struct {
		name   string
		action string
		call   func(ctx context.Context, environmentId, environmentType, systemId string) error
	}{
		{name: "link", action: "link", call: client.LinkEnterprisePolicy},
		{name: "unlink", action: "unlink", call: client.UnLinkEnterprisePolicy},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/policy-op-" + tc.action
			registerLifecycleSucceeded(lifecycleUrl)

			var gotMethod, gotBody string
			httpmock.RegisterResponder(http.MethodPost,
				fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/environments/%s/enterprisePolicies/NetworkInjection/%s?api-version=2019-10-01", testEnvironmentId, tc.action),
				func(req *http.Request) (*http.Response, error) {
					gotMethod = req.Method
					gotBody = readBody(t, req)
					return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
				})

			err := tc.call(context.Background(), testEnvironmentId, "NetworkInjection", "66666666-6666-6666-6666-666666666666")

			require.NoError(t, err)
			assert.Equal(t, http.MethodPost, gotMethod)
			assert.JSONEq(t, `{"systemId":"66666666-6666-6666-6666-666666666666"}`, gotBody)
		})
	}
}
