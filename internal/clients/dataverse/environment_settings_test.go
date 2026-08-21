// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testOrganizationId = "dddddddd-0000-0000-0000-000000000001"
	testOrgBody        = `{"value":[{"organizationid":"dddddddd-0000-0000-0000-000000000001","maxuploadfilesize":5242880,"isauditenabled":false}]}`
	testBackendBody    = `{"SettingDetailCollection":[{"Name":"AllowFlowRunDetails","Value":"true","DataType":2}]}`
)

func TestUnitGetEnvironmentBackendSettings_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: testBackendBody})

	settings, err := client.GetEnvironmentBackendSettings(context.Background(), testEnvironmentHost)

	require.NoError(t, err)
	// Backend settings live on the v9.0 surface, not v9.2.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.0/RetrieveSettingList%28%29"},
	})
	assert.Equal(t, "true", settings.GetValue("AllowFlowRunDetails"))
}

func TestUnitGetEnvironmentOrgSettings_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: testOrgBody})

	settings, err := client.GetEnvironmentOrgSettings(context.Background(), testEnvironmentHost)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.0/organizations"},
	})
	require.NotNil(t, settings.OrganizationId)
	assert.Equal(t, testOrganizationId, *settings.OrganizationId)
}

// TestUnitUpdateEnvironmentOrgSettings_PatchReadBackSequence pins the full
// read-modify-read sequence: resolve the organization id, PATCH only the
// fields the caller set (omitempty), then read the settings back.
func TestUnitUpdateEnvironmentOrgSettings_PatchReadBackSequence(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusOK, body: testOrgBody},
		step{status: http.StatusNoContent},
		step{status: http.StatusOK, body: testOrgBody},
	)

	auditEnabled := true
	settings, err := client.UpdateEnvironmentOrgSettings(context.Background(), testEnvironmentHost, EnvironmentOrgSettingsDto{IsAuditEnabled: &auditEnabled})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.0/organizations"},
		{method: "PATCH", url: "https://env.crm.dynamics.com/api/data/v9.0/organizations%28" + testOrganizationId + "%29", body: `{"isauditenabled":true}`},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.0/organizations"},
	})
	require.NotNil(t, settings.OrganizationId)
	assert.Equal(t, testOrganizationId, *settings.OrganizationId)
}

// TestUnitUpdateEnvironmentOrgSettings_InternalServerErrorIsTranslated pins a
// distinctive status contract: 500 is an accepted (not retried) status for the
// PATCH and is translated into ERROR_ENVIRONMENT_SETTINGS_FAILED carrying the
// response body verbatim.
func TestUnitUpdateEnvironmentOrgSettings_InternalServerErrorIsTranslated(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusOK, body: testOrgBody},
		step{status: http.StatusInternalServerError, body: `{"error":{"message":"setting rejected"}}`},
	)

	auditEnabled := true
	_, err := client.UpdateEnvironmentOrgSettings(context.Background(), testEnvironmentHost, EnvironmentOrgSettingsDto{IsAuditEnabled: &auditEnabled})

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrEnvironmentSettingsFailed))
	assert.Contains(t, err.Error(), "setting rejected")
	require.Len(t, *recorded, 2, "a 500 must not be retried and must stop the sequence")
}

// TestUnitUpdateEnvironmentBackendSettings_SavesEachSettingThenReadsBack pins
// the save loop: one SaveSettingValue() POST per setting, entries with an
// empty name or value are skipped, then the settings are read back.
func TestUnitUpdateEnvironmentBackendSettings_SavesEachSettingThenReadsBack(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusNoContent},
		step{status: http.StatusNoContent},
		step{status: http.StatusOK, body: testBackendBody},
	)

	backendSettings := EnvironmentBackendSettingsValueDto{
		SettingDetailCollection: []EnvironmentBackendSettingDto{
			{Name: "AllowFlowRunDetails", Value: "true", DataType: 2},
			{Name: "SkippedEmptyValue", Value: ""},
			{Name: "", Value: "skipped-empty-name"},
			{Name: "MaxRecords", Value: "5000", DataType: 0},
		},
	}
	settings, err := client.UpdateEnvironmentBackendSettings(context.Background(), testEnvironmentHost, backendSettings)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "POST", url: "https://env.crm.dynamics.com/api/data/v9.0/SaveSettingValue%28%29", body: `{"SettingName":"AllowFlowRunDetails","Value":"true"}`},
		{method: "POST", url: "https://env.crm.dynamics.com/api/data/v9.0/SaveSettingValue%28%29", body: `{"SettingName":"MaxRecords","Value":"5000"}`},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.0/RetrieveSettingList%28%29"},
	})
	assert.Equal(t, "true", settings.GetValue("AllowFlowRunDetails"))
}

func TestUnitUpdateEnvironmentBackendSettings_InternalServerErrorIsTranslated(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusInternalServerError, body: `{"error":{"message":"invalid setting"}}`})

	backendSettings := EnvironmentBackendSettingsValueDto{
		SettingDetailCollection: []EnvironmentBackendSettingDto{
			{Name: "AllowFlowRunDetails", Value: "true", DataType: 2},
		},
	}
	_, err := client.UpdateEnvironmentBackendSettings(context.Background(), testEnvironmentHost, backendSettings)

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrEnvironmentSettingsFailed))
	assert.Contains(t, err.Error(), "invalid setting")
	require.Len(t, *recorded, 1)
}

func TestUnitEnvironmentBackendSettings_GetValueSetValue(t *testing.T) {
	settings := EnvironmentBackendSettingsValueDto{
		SettingDetailCollection: []EnvironmentBackendSettingDto{
			{Name: "Existing", Value: "old", DataType: 1},
		},
	}

	assert.Equal(t, "old", settings.GetValue("Existing"))
	assert.Empty(t, settings.GetValue("Missing"))

	require.NoError(t, settings.SetValue("Existing", "new"))
	assert.Equal(t, "new", settings.GetValue("Existing"))

	require.NoError(t, settings.SetValue("NewBool", true))
	require.NoError(t, settings.SetValue("NewInt", 7))
	assert.Equal(t, "true", settings.GetValue("NewBool"))
	assert.Equal(t, "7", settings.GetValue("NewInt"))

	err := settings.SetValue("Bad", 1.5)
	require.Error(t, err)
	assert.EqualError(t, err, "unsupported value type: float64")

	// Appended entries carry the data type derived from the Go value.
	last := settings.SettingDetailCollection[len(settings.SettingDetailCollection)-1]
	assert.Equal(t, EnvironmentBackendSettingDto{Name: "NewInt", Value: "7", DataType: 0}, last)
}
