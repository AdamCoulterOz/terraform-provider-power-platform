// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEnvironmentId = "00000000-0000-0000-0000-000000000001"

	// getEnvironmentQuery is the canonical query string of a single-environment
	// read: the full $expand list plus the BAP api-version.
	getEnvironmentQuery = "%24expand=permissions%2Cproperties.capacity%2Cproperties%2FbillingPolicy%2Cproperties%2FcopilotPolicies&api-version=2023-06-01"
)

func getEnvironmentUrl(environmentId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?%s", environmentId, getEnvironmentQuery)
}

func deleteEnvironmentUrl(environmentId string) string {
	return fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?api-version=2023-06-01", environmentId)
}

// registerGetEnvironmentResponder serves a minimal, valid environment for the
// canonical single-environment read.
func registerGetEnvironmentResponder(environmentId, propertiesJson string) {
	body := fmt.Sprintf(`{"id":"/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%[1]s","name":"%[1]s","location":"europe","properties":%[2]s}`, environmentId, propertiesJson)
	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(environmentId),
		httpmock.NewStringResponder(http.StatusOK, body))
}

func TestUnitGetEnvironment_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotUrl string
	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{"name":"%s","location":"europe","properties":{"displayName":"Test","environmentSku":"Sandbox"}}`, testEnvironmentId)), nil
		})

	env, err := client.GetEnvironment(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/"+testEnvironmentId+
		"?%24expand=permissions%2Cproperties.capacity%2Cproperties%2FbillingPolicy%2Cproperties%2FcopilotPolicies&api-version=2023-06-01", gotUrl)
	assert.Equal(t, testEnvironmentId, env.Name)
	assert.Equal(t, "Sandbox", env.Properties.EnvironmentSku)
}

func TestUnitGetEnvironment_EmptyIdIsGuarded(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	env, err := client.GetEnvironment(context.Background(), "")

	assert.Nil(t, env)
	assert.EqualError(t, err, "environment id must not be empty")
	assert.Equal(t, 0, httpmock.GetTotalCallCount(), "the empty-id guard must not degrade into a list-environments request")
}

func TestUnitGetEnvironment_NotFoundIsTranslated(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"EnvironmentNotFound"}}`))

	env, err := client.GetEnvironment(context.Background(), testEnvironmentId)

	assert.Nil(t, env)
	require.Error(t, err)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
	assert.ErrorContains(t, err, fmt.Sprintf("environment '%s' not found", testEnvironmentId))
}

func TestUnitGetEnvironment_UnexpectedStatusReturnsStatusError(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusBadRequest, `{"error":{"code":"BadRequest"}}`), nil
		})

	_, err := client.GetEnvironment(context.Background(), testEnvironmentId)

	require.Error(t, err)
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
	assert.Equal(t, []int{http.StatusOK}, statusErr.ExpectedStatusCodes)
	assert.Equal(t, 1, attempts, "a non-retryable status must not be retried")
}

func TestUnitGetEnvironment_RetriesTransientStatuses(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusServiceUnavailable, ""), nil
			}
			return httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{"name":"%s","properties":{"displayName":"Test"}}`, testEnvironmentId)), nil
		})

	env, err := client.GetEnvironment(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "a transient 503 must be retried")
	assert.Equal(t, testEnvironmentId, env.Name)
}

func TestUnitGetEnvironment_MissingPropertiesIsAnError(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusOK, fmt.Sprintf(`{"name":"%s"}`, testEnvironmentId)))

	env, err := client.GetEnvironment(context.Background(), testEnvironmentId)

	assert.Nil(t, env)
	assert.EqualError(t, err, fmt.Sprintf("unexpected response when reading environment '%s': the response contains no environment properties", testEnvironmentId))
}

func TestUnitGetEnvironment_BackfillsZeroUuids(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	t.Run("empty ids are backfilled with the zero uuid", func(t *testing.T) {
		registerGetEnvironmentResponder(testEnvironmentId,
			`{"displayName":"Test","linkedEnvironmentMetadata":{"domainName":"mydomain"},"parentEnvironmentGroup":{}}`)

		env, err := client.GetEnvironment(context.Background(), testEnvironmentId)

		require.NoError(t, err)
		assert.Equal(t, constants.ZERO_UUID, env.Properties.LinkedEnvironmentMetadata.SecurityGroupId)
		assert.Equal(t, constants.ZERO_UUID, env.Properties.ParentEnvironmentGroup.Id)
	})

	t.Run("populated ids are preserved", func(t *testing.T) {
		registerGetEnvironmentResponder(testEnvironmentId,
			`{"displayName":"Test","linkedEnvironmentMetadata":{"securityGroupId":"11111111-1111-1111-1111-111111111111"},"parentEnvironmentGroup":{"id":"22222222-2222-2222-2222-222222222222"}}`)

		env, err := client.GetEnvironment(context.Background(), testEnvironmentId)

		require.NoError(t, err)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", env.Properties.LinkedEnvironmentMetadata.SecurityGroupId)
		assert.Equal(t, "22222222-2222-2222-2222-222222222222", env.Properties.ParentEnvironmentGroup.Id)
	})

	t.Run("absent sections stay nil", func(t *testing.T) {
		registerGetEnvironmentResponder(testEnvironmentId, `{"displayName":"Test"}`)

		env, err := client.GetEnvironment(context.Background(), testEnvironmentId)

		require.NoError(t, err)
		assert.Nil(t, env.Properties.LinkedEnvironmentMetadata)
		assert.Nil(t, env.Properties.ParentEnvironmentGroup)
	})
}

func TestUnitGetEnvironments_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotUrl string
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments?%24expand=properties%2FbillingPolicy%2Cproperties%2FcopilotPolicies&api-version=2023-06-01",
		func(req *http.Request) (*http.Response, error) {
			gotUrl = req.URL.String()
			return httpmock.NewStringResponse(http.StatusOK, `{"value":[{"name":"env-1","properties":{"displayName":"One"}},{"name":"env-2","properties":{"displayName":"Two"}}]}`), nil
		})

	envs, err := client.GetEnvironments(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments?%24expand=properties%2FbillingPolicy%2Cproperties%2FcopilotPolicies&api-version=2023-06-01", gotUrl)
	require.Len(t, envs, 2)
	assert.Equal(t, "env-1", envs[0].Name)
	assert.Equal(t, "env-2", envs[1].Name)
}

func TestUnitGetEnvironmentHostById_ResolvesDataverseHost(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	registerGetEnvironmentResponder(testEnvironmentId,
		`{"displayName":"Test","linkedEnvironmentMetadata":{"instanceUrl":"https://org42.crm.dynamics.com/"}}`)

	host, err := client.GetEnvironmentHostById(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, "org42.crm.dynamics.com", host)
}

func TestUnitGetEnvironmentHostById_WithoutDataverseIsTranslated(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	tests := []struct {
		name           string
		propertiesJson string
	}{
		{name: "no linked environment metadata", propertiesJson: `{"displayName":"Test"}`},
		{name: "empty instance url", propertiesJson: `{"displayName":"Test","linkedEnvironmentMetadata":{"instanceUrl":""}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registerGetEnvironmentResponder(testEnvironmentId, tc.propertiesJson)

			host, err := client.GetEnvironmentHostById(context.Background(), testEnvironmentId)

			assert.Empty(t, host)
			require.Error(t, err)
			assert.ErrorIs(t, err, customerrors.ErrEnvironmentUrlNotFound)
			assert.ErrorContains(t, err, "environment url not found, please check if the environment has dataverse linked")
		})
	}
}

func TestUnitGetEnvironmentHostById_PropagatesNotFound(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"EnvironmentNotFound"}}`))

	_, err := client.GetEnvironmentHostById(context.Background(), testEnvironmentId)

	require.Error(t, err)
	assert.ErrorIs(t, err, customerrors.ErrObjectNotFound)
}

func TestUnitDeleteEnvironment_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotMethod, gotBody string
	attempts := 0
	httpmock.RegisterResponder(http.MethodDelete, deleteEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			gotMethod = req.Method
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	err := client.DeleteEnvironment(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.JSONEq(t, `{"code":"7","message":"Deleted using Power Platform Terraform Provider"}`, gotBody)
	assert.Equal(t, 1, attempts)
}

func TestUnitDeleteEnvironment_NotFoundIsTreatedAsDeleted(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodDelete, deleteEnvironmentUrl(testEnvironmentId),
		httpmock.NewStringResponder(http.StatusNotFound, `{"error":{"code":"EnvironmentNotFound"}}`))

	err := client.DeleteEnvironment(context.Background(), testEnvironmentId)

	assert.NoError(t, err)
}

func TestUnitDeleteEnvironment_RetriesConflictWhenOperationNotStartable(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodDelete, deleteEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusConflict, `{"error":{"code":"OperationNotStartable","message":"Another lifecycle operation is running"}}`), nil
			}
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	err := client.DeleteEnvironment(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "an OperationNotStartable conflict must be retried")
}

func TestUnitDeleteEnvironment_RetriesConflictWithEmptyBody(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodDelete, deleteEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusConflict, ""), nil
			}
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	err := client.DeleteEnvironment(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "a bodyless conflict is transient and must be retried")
}

func TestUnitDeleteEnvironment_ConflictWithOtherBodyFails(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	conflictBody := `{"error":{"code":"CannotDeleteDefaultEnvironment"}}`
	attempts := 0
	httpmock.RegisterResponder(http.MethodDelete, deleteEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusConflict, conflictBody), nil
		})

	err := client.DeleteEnvironment(context.Background(), testEnvironmentId)

	assert.EqualError(t, err, "environment failed with HTTP 409. Body: "+conflictBody)
	assert.Equal(t, 1, attempts, "a non-OperationNotStartable conflict must fail without retrying")
}

func TestUnitDeleteEnvironment_ConflictMaxRetriesExhausted(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodDelete, deleteEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusConflict, `{"error":{"code":"OperationNotStartable"}}`), nil
		})

	err := client.DeleteEnvironment(context.Background(), testEnvironmentId)

	require.Error(t, err)
	assert.ErrorIs(t, err, customerrors.ErrEnvironmentDeletion)
	assert.ErrorContains(t, err, fmt.Sprintf("maximum retries (%d) reached for DeleteEnvironment on conflict", constants.MAX_RETRY_COUNT))
	assert.Equal(t, constants.MAX_RETRY_COUNT+1, attempts)
}

func TestUnitDeleteEnvironment_RetriesWhenLifecycleFails(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/delete-op-1"
	httpmock.RegisterResponder(http.MethodGet, lifecycleUrl,
		httpmock.NewStringResponder(http.StatusOK, `{"state":{"id":"Failed"}}`))

	attempts := 0
	httpmock.RegisterResponder(http.MethodDelete, deleteEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
			}
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})

	err := client.DeleteEnvironment(context.Background(), testEnvironmentId)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "a failed delete lifecycle operation must be retried")
}

const createEnvironmentUrl = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/environments?api-version=2023-06-01"

func newCreateEnvironmentRequest() EnvironmentCreateDto {
	return EnvironmentCreateDto{
		Location: "europe",
		Properties: EnvironmentCreatePropertiesDto{
			DisplayName:    "Terraform Test",
			EnvironmentSku: "Sandbox",
		},
	}
}

func TestUnitCreateEnvironment_CreatedSynchronously(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	createdId := "00000000-0000-0000-0000-0000000000c1"
	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, createEnvironmentUrl,
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusCreated, fmt.Sprintf(`{"name":"%s","properties":{"provisioningState":"Succeeded"}}`, createdId)), nil
		})
	registerGetEnvironmentResponder(createdId, `{"displayName":"Terraform Test","environmentSku":"Sandbox"}`)

	env, err := client.CreateEnvironment(context.Background(), newCreateEnvironmentRequest())

	require.NoError(t, err)
	assert.JSONEq(t, `{"location":"europe","properties":{"billingPolicy":{"id":""},"displayName":"Terraform Test","environmentSku":"Sandbox"}}`, gotBody)
	assert.Equal(t, createdId, env.Name)
}

func TestUnitCreateEnvironment_AcceptedWaitsForLifecycle(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	createdId := "00000000-0000-0000-0000-0000000000c2"
	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/create-op-1"
	httpmock.RegisterResponder(http.MethodPost, createEnvironmentUrl,
		func(req *http.Request) (*http.Response, error) {
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})
	httpmock.RegisterResponder(http.MethodGet, lifecycleUrl,
		httpmock.NewStringResponder(http.StatusOK, fmt.Sprintf(`{"state":{"id":"Succeeded"},"links":{"environment":{"path":"/providers/Microsoft.BusinessAppPlatform/environments/%s"}}}`, createdId)))
	registerGetEnvironmentResponder(createdId, `{"displayName":"Terraform Test"}`)

	env, err := client.CreateEnvironment(context.Background(), newCreateEnvironmentRequest())

	require.NoError(t, err)
	assert.Equal(t, createdId, env.Name)
}

func TestUnitCreateEnvironment_AcceptedWithoutEnvironmentIdIsGuarded(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/create-op-2"
	httpmock.RegisterResponder(http.MethodPost, createEnvironmentUrl,
		func(req *http.Request) (*http.Response, error) {
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})
	httpmock.RegisterResponder(http.MethodGet, lifecycleUrl,
		httpmock.NewStringResponder(http.StatusOK, `{"state":{"id":"Succeeded"}}`))

	env, err := client.CreateEnvironment(context.Background(), newCreateEnvironmentRequest())

	assert.Nil(t, env)
	assert.EqualError(t, err, "environment creation failed. the API did not return an id for the created environment")
}

func TestUnitCreateEnvironment_InternalServerErrorIsTranslated(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	attempts := 0
	httpmock.RegisterResponder(http.MethodPost, createEnvironmentUrl,
		func(req *http.Request) (*http.Response, error) {
			attempts++
			return httpmock.NewStringResponse(http.StatusInternalServerError, `{"error":{"code":"ProvisioningQuotaExceeded"}}`), nil
		})

	env, err := client.CreateEnvironment(context.Background(), newCreateEnvironmentRequest())

	assert.Nil(t, env)
	require.Error(t, err)
	assert.ErrorIs(t, err, customerrors.ErrEnvironmentCreation)
	assert.ErrorContains(t, err, "ProvisioningQuotaExceeded")
	assert.Equal(t, 1, attempts, "a 500 on create is an accepted status and must not be replayed by the transport")
}

func TestUnitCreateEnvironment_RetriesConflictWhenOperationNotStartable(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	createdId := "00000000-0000-0000-0000-0000000000c3"
	attempts := 0
	httpmock.RegisterResponder(http.MethodPost, createEnvironmentUrl,
		func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return httpmock.NewStringResponse(http.StatusConflict, `{"error":{"code":"OperationNotStartable"}}`), nil
			}
			return httpmock.NewStringResponse(http.StatusCreated, fmt.Sprintf(`{"name":"%s","properties":{"provisioningState":"Succeeded"}}`, createdId)), nil
		})
	registerGetEnvironmentResponder(createdId, `{"displayName":"Terraform Test"}`)

	env, err := client.CreateEnvironment(context.Background(), newCreateEnvironmentRequest())

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, createdId, env.Name)
}

func TestUnitModifyEnvironmentType_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/modifySku?api-version=2021-04-01", testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.ModifyEnvironmentType(context.Background(), testEnvironmentId, "Production")

	require.NoError(t, err)
	assert.JSONEq(t, `{"environmentSku":"Production"}`, gotBody)
}

func TestUnitUpdateEnvironment_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/update-op-1"
	registerLifecycleSucceeded(lifecycleUrl)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPatch,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?%%24expand=permissions%%2Cproperties.capacity%%2Cproperties%%2FbillingPolicy&api-version=2021-04-01", testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})
	registerGetEnvironmentResponder(testEnvironmentId, `{"displayName":"Renamed","states":{"management":{"id":"Ready"}}}`)

	env, err := client.UpdateEnvironment(context.Background(), testEnvironmentId, EnvironmentDto{
		Properties: &EnviromentPropertiesDto{DisplayName: "Renamed"},
	})

	require.NoError(t, err)
	assert.JSONEq(t, `{"properties":{"description":"","displayName":"Renamed"}}`, gotBody)
	assert.Equal(t, "Ready", env.Properties.States.Management.Id)
}

func TestUnitUpdateEnvironment_FailsOnUnexpectedManagementState(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/update-op-2"
	registerLifecycleSucceeded(lifecycleUrl)

	httpmock.RegisterResponder(http.MethodPatch,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?%%24expand=permissions%%2Cproperties.capacity%%2Cproperties%%2FbillingPolicy&api-version=2021-04-01", testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})
	registerGetEnvironmentResponder(testEnvironmentId, `{"displayName":"Renamed","states":{"management":{"id":"Deleting"}}}`)

	env, err := client.UpdateEnvironment(context.Background(), testEnvironmentId, EnvironmentDto{
		Properties: &EnviromentPropertiesDto{DisplayName: "Renamed"},
	})

	assert.Nil(t, env)
	assert.EqualError(t, err, "environment update failed. unexpected management state: Deleting")
}

func TestUnitUpdateEnvironmentAiFeatures_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPatch,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?api-version=2021-04-01", testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
		})
	registerGetEnvironmentResponder(testEnvironmentId, `{"displayName":"Test","bingChatEnabled":true}`)

	bingChatEnabled := true
	env, err := client.UpdateEnvironmentAiFeatures(context.Background(), testEnvironmentId, GenerativeAiFeaturesDto{
		Properties: GenerativeAiFeaturesPropertiesDto{BingChatEnabled: &bingChatEnabled},
	})

	require.NoError(t, err)
	assert.JSONEq(t, `{"properties":{"bingChatEnabled":true}}`, gotBody)
	assert.True(t, env.Properties.BingChatEnabled)
}

func TestUnitUpdateEnvironment_PollsWhileRunning(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/update-op-3"
	registerLifecycleSucceeded(lifecycleUrl)
	httpmock.RegisterResponder(http.MethodPatch,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?%%24expand=permissions%%2Cproperties.capacity%%2Cproperties%%2FbillingPolicy&api-version=2021-04-01", testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})

	readAttempts := 0
	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			readAttempts++
			state := "Running"
			if readAttempts > 1 {
				state = "Ready"
			}
			return httpmock.NewStringResponse(http.StatusOK,
				fmt.Sprintf(`{"name":"%s","properties":{"displayName":"Renamed","states":{"management":{"id":"%s"}}}}`, testEnvironmentId, state)), nil
		})

	env, err := client.UpdateEnvironment(context.Background(), testEnvironmentId, EnvironmentDto{
		Properties: &EnviromentPropertiesDto{DisplayName: "Renamed"},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, readAttempts, "the environment must be re-read until it reports Ready")
	assert.Equal(t, "Ready", env.Properties.States.Management.Id)
}

func TestUnitUpdateEnvironmentAiFeatures_WaitsUntilChangeIsVisible(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPatch,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s?api-version=2021-04-01", testEnvironmentId),
		httpmock.NewStringResponder(http.StatusNoContent, ""))

	readAttempts := 0
	httpmock.RegisterResponder(http.MethodGet, getEnvironmentUrl(testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			readAttempts++
			properties := `{"displayName":"Test","bingChatEnabled":true}`
			if readAttempts > 1 {
				properties = `{"displayName":"Test","bingChatEnabled":true,"copilotPolicies":{"crossGeoCopilotDataMovementEnabled":true}}`
			}
			return httpmock.NewStringResponse(http.StatusOK,
				fmt.Sprintf(`{"name":"%s","properties":%s}`, testEnvironmentId, properties)), nil
		})

	env, err := client.UpdateEnvironmentAiFeatures(context.Background(), testEnvironmentId, GenerativeAiFeaturesDto{
		Properties: GenerativeAiFeaturesPropertiesDto{
			BingChatEnabled: boolPtr(true),
			CopilotPolicies: &CopilotPoliciesDto{CrossGeoCopilotDataMovementEnabled: boolPtr(true)},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, readAttempts, "reads must be repeated until the applied values are visible")
	require.NotNil(t, env.Properties.CopilotPolicies)
	require.NotNil(t, env.Properties.CopilotPolicies.CrossGeoCopilotDataMovementEnabled)
	assert.True(t, *env.Properties.CopilotPolicies.CrossGeoCopilotDataMovementEnabled)
}

func TestUnitAddDataverseToEnvironment_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/provision-op-1"
	var gotBody string
	httpmock.RegisterResponder(http.MethodPost,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/environments/%s/provisionInstance?api-version=2021-04-01", testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})
	httpmock.RegisterResponder(http.MethodGet, lifecycleUrl,
		httpmock.NewStringResponder(http.StatusOK,
			fmt.Sprintf(`{"name":"%s","properties":{"provisioningState":"Succeeded","linkedEnvironmentMetadata":{"domainName":"mydomain"}}}`, testEnvironmentId)))

	env, err := client.AddDataverseToEnvironment(context.Background(), testEnvironmentId, CreateLinkEnvironmentMetadataDto{
		BaseLanguage:    1033,
		DomainName:      "mydomain",
		Currency:        &CreateCurrencyDto{Code: "USD"},
		SecurityGroupId: "33333333-3333-3333-3333-333333333333",
	})

	require.NoError(t, err)
	assert.JSONEq(t, `{"baseLanguage":1033,"domainName":"mydomain","currency":{"code":"USD"},"securityGroupId":"33333333-3333-3333-3333-333333333333"}`, gotBody)
	assert.Equal(t, "Succeeded", env.Properties.ProvisioningState)
}

func TestUnitAddDataverseToEnvironment_FailedProvisioningState(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	lifecycleUrl := "https://api.bap.microsoft.com/lifecycleOperations/provision-op-2"
	httpmock.RegisterResponder(http.MethodPost,
		fmt.Sprintf("https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/environments/%s/provisionInstance?api-version=2021-04-01", testEnvironmentId),
		func(req *http.Request) (*http.Response, error) {
			return newResponseWithLocation(http.StatusAccepted, lifecycleUrl), nil
		})
	httpmock.RegisterResponder(http.MethodGet, lifecycleUrl,
		httpmock.NewStringResponder(http.StatusOK,
			fmt.Sprintf(`{"name":"%s","properties":{"provisioningState":"Failed"}}`, testEnvironmentId)))

	_, err := client.AddDataverseToEnvironment(context.Background(), testEnvironmentId, CreateLinkEnvironmentMetadataDto{DomainName: "mydomain"})

	assert.EqualError(t, err, "dataverse creation failed. provisioning state: Failed")
}

func TestUnitWaitForDataverseMetadata_ReturnsImmediatelyWhenPresent(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	env := &EnvironmentDto{
		Name: testEnvironmentId,
		Properties: &EnviromentPropertiesDto{
			LinkedEnvironmentMetadata: &LinkedEnvironmentMetadataDto{DomainName: "mydomain"},
		},
	}

	got, err := client.WaitForDataverseMetadata(context.Background(), testEnvironmentId, env)

	require.NoError(t, err)
	assert.Same(t, env, got)
	assert.Equal(t, 0, httpmock.GetTotalCallCount())
}

func TestUnitWaitForDataverseMetadata_PollsUntilMetadataAppears(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	registerGetEnvironmentResponder(testEnvironmentId,
		`{"displayName":"Test","linkedEnvironmentMetadata":{"domainName":"mydomain"}}`)

	env := &EnvironmentDto{Name: testEnvironmentId, Properties: &EnviromentPropertiesDto{}}
	got, err := client.WaitForDataverseMetadata(context.Background(), testEnvironmentId, env)

	require.NoError(t, err)
	require.NotNil(t, got.Properties.LinkedEnvironmentMetadata)
	assert.Equal(t, "mydomain", got.Properties.LinkedEnvironmentMetadata.DomainName)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

const validateEnvironmentDetailsUrl = "https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/validateEnvironmentDetails?api-version=2021-04-01"

func TestUnitValidateCreateEnvironmentDetails_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, validateEnvironmentDetailsUrl,
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.ValidateCreateEnvironmentDetails(context.Background(), "europe", "mydomain")

	require.NoError(t, err)
	assert.JSONEq(t, `{"domainName":"mydomain","environmentLocation":"europe"}`, gotBody)
}

func TestUnitValidateCreateEnvironmentDetails_UnexpectedStatusFails(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodPost, validateEnvironmentDetailsUrl,
		httpmock.NewStringResponder(http.StatusBadRequest, `{"error":{"code":"DomainNameAlreadyInUse"}}`))

	err := client.ValidateCreateEnvironmentDetails(context.Background(), "europe", "mydomain")

	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
}

func TestUnitValidateUpdateEnvironmentDetails_RequestShape(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	var gotBody string
	httpmock.RegisterResponder(http.MethodPost, validateEnvironmentDetailsUrl,
		func(req *http.Request) (*http.Response, error) {
			gotBody = readBody(t, req)
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		})

	err := client.ValidateUpdateEnvironmentDetails(context.Background(), testEnvironmentId, "mydomain")

	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"domainName":"mydomain","environmentName":"%s"}`, testEnvironmentId), gotBody)
}

// registerLocationsResponder serves the tenant locations used by the
// location, currency and language validators.
func registerLocationsResponder() {
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/locations?api-version=2023-06-01",
		httpmock.NewStringResponder(http.StatusOK,
			`{"value":[{"name":"europe","properties":{"displayName":"Europe","azureRegions":["westeurope","northeurope"]}},{"name":"asia","properties":{"displayName":"Asia","azureRegions":["eastasia"]}}]}`))
}

func TestUnitLocationValidator(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)
	registerLocationsResponder()

	tests := []struct {
		name          string
		location      string
		azureRegion   string
		expectedError string
	}{
		{name: "valid location without region", location: "europe", azureRegion: ""},
		{name: "valid location and region", location: "europe", azureRegion: "westeurope"},
		{name: "unknown location", location: "mars", expectedError: "location 'mars' is not valid. valid locations are: europe, asia"},
		{name: "unknown region", location: "europe", azureRegion: "eastus", expectedError: "region 'eastus' is not valid for location europe. valid regions are: westeurope, northeurope"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := client.LocationValidator(context.Background(), tc.location, tc.azureRegion)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestUnitValidateCurrencyCode(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet,
		"https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/locations/europe/environmentCurrencies?api-version=2023-06-01",
		httpmock.NewStringResponder(http.StatusOK, `{"value":[{"name":"USD"},{"name":"EUR"}]}`))

	assert.NoError(t, client.ValidateCurrencyCode(context.Background(), "europe", "EUR"))
	assert.EqualError(t, client.ValidateCurrencyCode(context.Background(), "europe", "ABC"),
		"currency Code ABC is not valid. valid currency codes are: USD, EUR")
}

func TestUnitValidateLanguageCode(t *testing.T) {
	client := newTestClient(t)
	activateHttpMocks(t)

	httpmock.RegisterResponder(http.MethodGet,
		"https://api.bap.microsoft.com/providers/Microsoft.BusinessAppPlatform/locations/europe/environmentLanguages?api-version=2023-06-01",
		httpmock.NewStringResponder(http.StatusOK, `{"value":[{"name":"1033"},{"name":"1031"}]}`))

	assert.NoError(t, client.ValidateLanguageCode(context.Background(), "europe", "1033"))
	assert.EqualError(t, client.ValidateLanguageCode(context.Background(), "europe", "9999"),
		"language Code 9999 is not valid. valid language codes are: 1033, 1031")
}
