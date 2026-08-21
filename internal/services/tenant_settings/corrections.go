// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package tenant_settings

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// applyCorrections filters the settings returned by the API down to the
// fields opted in by the planned settings, and restores zero-UUID routing
// target ids that the API reports back as null.
func applyCorrections(ctx context.Context, planned bapi.TenantSettingsDto, actual bapi.TenantSettingsDto) (*bapi.TenantSettingsDto, error) {
	correctedFilter := filterDto(ctx, planned, actual)
	corrected, ok := correctedFilter.(*bapi.TenantSettingsDto)
	if !ok {
		tflog.Error(ctx, "Type assertion failed in applyCorrections")
		return nil, errors.New("type assertion to *tenantSettingsDto failed in applyCorrections")
	}

	if planned.PowerPlatform != nil && planned.PowerPlatform.Governance != nil {
		if planned.PowerPlatform.Governance.EnvironmentRoutingTargetSecurityGroupId != nil && *planned.PowerPlatform.Governance.EnvironmentRoutingTargetSecurityGroupId == constants.ZERO_UUID && corrected.PowerPlatform.Governance.EnvironmentRoutingTargetSecurityGroupId == nil {
			zu := constants.ZERO_UUID
			corrected.PowerPlatform.Governance.EnvironmentRoutingTargetSecurityGroupId = &zu
		}

		if planned.PowerPlatform.Governance.EnvironmentRoutingTargetEnvironmentGroupId != nil && *planned.PowerPlatform.Governance.EnvironmentRoutingTargetEnvironmentGroupId == constants.ZERO_UUID && corrected.PowerPlatform.Governance.EnvironmentRoutingTargetEnvironmentGroupId == nil {
			zu := constants.ZERO_UUID
			corrected.PowerPlatform.Governance.EnvironmentRoutingTargetEnvironmentGroupId = &zu
		}
	}
	return corrected, nil
}

// This function is used to filter out the fields that are not opted in to configuration.
// The backend always returns all properties, but Terraform can only handle the properties that are opted in.
func filterDto(ctx context.Context, configuredSettings any, backendSettings any) any {
	configuredType := reflect.TypeOf(configuredSettings)
	backendType := reflect.TypeOf(backendSettings)
	if configuredType != backendType {
		return nil
	}

	output := reflect.New(configuredType).Interface()

	visibleFields := reflect.VisibleFields(configuredType)

	configuredValue := reflect.ValueOf(configuredSettings)
	backendValue := reflect.ValueOf(backendSettings)

	for fieldIndex, fieldInfo := range visibleFields {
		tflog.Debug(ctx, fmt.Sprintf("Field: %s", fieldInfo.Name))

		configuredFieldValue := configuredValue.Field(fieldIndex)
		backendFieldValue := backendValue.Field(fieldIndex)
		outputField := reflect.ValueOf(output).Elem().Field(fieldIndex)

		if !configuredFieldValue.IsNil() && !backendFieldValue.IsNil() && backendFieldValue.IsValid() && outputField.CanSet() {
			if configuredFieldValue.Kind() == reflect.Pointer && configuredFieldValue.Elem().Kind() == reflect.Struct {
				outputStruct := filterDto(ctx, configuredFieldValue.Elem().Interface(), backendFieldValue.Elem().Interface())
				outputField.Set(reflect.ValueOf(outputStruct))
			} else if configuredFieldValue.Kind() == reflect.Pointer && configuredFieldValue.Elem().Kind() == reflect.Bool {
				boolValue := backendFieldValue.Elem().Bool()
				newBool := bool(boolValue)
				outputField.Set(reflect.ValueOf(&newBool))
			} else if configuredFieldValue.Kind() == reflect.Pointer && configuredFieldValue.Elem().Kind() == reflect.String {
				stringValue := backendFieldValue.Elem().String()
				newString := string(stringValue)
				outputField.Set(reflect.ValueOf(&newString))
			} else if configuredFieldValue.Kind() == reflect.Pointer && configuredFieldValue.Elem().Kind() == reflect.Int64 {
				int64Value := backendFieldValue.Elem().Int()
				newInt64 := int64(int64Value)
				outputField.Set(reflect.ValueOf(&newInt64))
			} else {
				tflog.Debug(ctx, fmt.Sprintf("Skipping unknown field type %s", configuredFieldValue.Kind()))
			}
		}
	}

	return output
}
