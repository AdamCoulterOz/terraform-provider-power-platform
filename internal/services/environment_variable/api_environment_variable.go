// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environmentvariable

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

func newEnvironmentVariableClient(apiClient *api.Client) client {
	return client{
		Client:    bapi.NewClient(apiClient),
		dataverse: dataverse.NewClient(apiClient),
	}
}

// client is the environment variable service client. BAPI environment lookups
// (including GetEnvironmentHostById) are promoted from the embedded
// bapi.Client; the Dataverse Web API operations are delegated to the dataverse
// client after resolving the environment host.
type client struct {
	*bapi.Client
	dataverse *dataverse.Client
}

func (client *client) DataverseExists(ctx context.Context, environmentId string) (bool, error) {
	env, err := client.GetEnvironment(ctx, environmentId)
	if err != nil {
		return false, err
	}
	return env.Properties.LinkedEnvironmentMetadata != nil && env.Properties.LinkedEnvironmentMetadata.InstanceURL != "", nil
}

func (client *client) GetEnvironmentVariable(ctx context.Context, environmentId, schemaName string) (*resolvedEnvironmentVariableDto, error) {
	definition, err := client.getDefinitionBySchemaName(ctx, environmentId, schemaName)
	if err != nil {
		return nil, err
	}

	value, err := client.getCurrentValueByDefinitionID(ctx, environmentId, definition.EnvironmentVariableDefinitionId)
	if err != nil {
		return nil, err
	}

	return &resolvedEnvironmentVariableDto{
		Definition: *definition,
		Value:      value,
	}, nil
}

func (client *client) UpsertEnvironmentVariableValue(ctx context.Context, environmentId, schemaName, value string) (*resolvedEnvironmentVariableDto, error) {
	definition, err := client.getDefinitionBySchemaName(ctx, environmentId, schemaName)
	if err != nil {
		return nil, err
	}

	if err := validateValueForDefinitionType(definition.Type, value); err != nil {
		return nil, err
	}

	currentValue, err := client.getCurrentValueByDefinitionID(ctx, environmentId, definition.EnvironmentVariableDefinitionId)
	if err != nil {
		return nil, err
	}

	if currentValue == nil {
		currentValue, err = client.createCurrentValue(ctx, environmentId, definition.EnvironmentVariableDefinitionId, schemaName, value)
		if err != nil {
			return nil, err
		}
	} else {
		currentValue, err = client.updateCurrentValue(ctx, environmentId, currentValue.EnvironmentVariableValueId, value)
		if err != nil {
			return nil, err
		}
	}

	return &resolvedEnvironmentVariableDto{
		Definition: *definition,
		Value:      currentValue,
	}, nil
}

func (client *client) DeleteEnvironmentVariableValue(ctx context.Context, environmentId, schemaName string) error {
	definition, err := client.getDefinitionBySchemaName(ctx, environmentId, schemaName)
	if err != nil {
		if isEnvironmentVariableDefinitionNotFound(err) {
			return nil
		}
		return err
	}

	currentValue, err := client.getCurrentValueByDefinitionID(ctx, environmentId, definition.EnvironmentVariableDefinitionId)
	if err != nil {
		return err
	}
	if currentValue == nil {
		return nil
	}

	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}

	return client.dataverse.DeleteEnvironmentVariableValue(ctx, environmentHost, currentValue.EnvironmentVariableValueId)
}

func (client *client) getDefinitionBySchemaName(ctx context.Context, environmentId, schemaName string) (*dataverse.EnvironmentVariableDefinitionDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverse.GetEnvironmentVariableDefinitionBySchemaName(ctx, environmentHost, schemaName)
}

func (client *client) getCurrentValueByDefinitionID(ctx context.Context, environmentId, definitionID string) (*dataverse.EnvironmentVariableValueDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverse.GetEnvironmentVariableValueByDefinitionId(ctx, environmentHost, definitionID)
}

func (client *client) createCurrentValue(ctx context.Context, environmentId, definitionID, schemaName, value string) (*dataverse.EnvironmentVariableValueDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	entityID, err := client.dataverse.CreateEnvironmentVariableValue(ctx, environmentHost, definitionID, schemaName, value)
	if err != nil {
		return nil, err
	}

	return client.getCurrentValueByID(ctx, environmentId, entityID)
}

func (client *client) updateCurrentValue(ctx context.Context, environmentId, valueID, value string) (*dataverse.EnvironmentVariableValueDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	if err := client.dataverse.UpdateEnvironmentVariableValue(ctx, environmentHost, valueID, value); err != nil {
		return nil, err
	}

	return client.getCurrentValueByID(ctx, environmentId, valueID)
}

func (client *client) getCurrentValueByID(ctx context.Context, environmentId, valueID string) (*dataverse.EnvironmentVariableValueDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverse.GetEnvironmentVariableValueById(ctx, environmentHost, valueID)
}

func environmentVariableResourceID(environmentID, schemaName string) string {
	return fmt.Sprintf("%s/%s", environmentID, schemaName)
}

func parseEnvironmentVariableResourceID(id string) (environmentID string, schemaName string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid import id %q, expected {environment_id}/{schema_name}", id)
	}

	return parts[0], parts[1], nil
}

func isEnvironmentVariableDefinitionNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "environment variable definition with schema name")
}

const (
	environmentVariableTypeString     = 100000000
	environmentVariableTypeNumber     = 100000001
	environmentVariableTypeBoolean    = 100000002
	environmentVariableTypeJSON       = 100000003
	environmentVariableTypeDataSource = 100000004
	environmentVariableTypeSecret     = 100000005
)

func environmentVariableTypeLabel(value int64) string {
	switch value {
	case environmentVariableTypeString:
		return "String"
	case environmentVariableTypeNumber:
		return "Number"
	case environmentVariableTypeBoolean:
		return "Boolean"
	case environmentVariableTypeJSON:
		return "JSON"
	case environmentVariableTypeDataSource:
		return "Data Source"
	case environmentVariableTypeSecret:
		return "Secret"
	default:
		return fmt.Sprintf("%d", value)
	}
}

// validateValueForDefinitionType checks the configured value against the definition's
// declared type before it is written to Dataverse. Dataverse itself does not validate
// environment variable values (the value column is free text), so without this check
// an invalid value would be accepted and only fail later in the consuming components.
func validateValueForDefinitionType(definitionType int64, value string) error {
	switch definitionType {
	case environmentVariableTypeBoolean:
		// Dataverse stores boolean ("Two options") environment variable values as "yes"/"no",
		// which is what the maker portal writes for this type.
		if lowered := strings.ToLower(value); lowered != "yes" && lowered != "no" {
			return fmt.Errorf("environment variable definition has type %s: value must be \"yes\" or \"no\", got %q", environmentVariableTypeLabel(definitionType), value)
		}
	case environmentVariableTypeNumber:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("environment variable definition has type %s: value %q is not a valid number", environmentVariableTypeLabel(definitionType), value)
		}
	case environmentVariableTypeJSON:
		if !json.Valid([]byte(value)) {
			return fmt.Errorf("environment variable definition has type %s: value is not valid JSON", environmentVariableTypeLabel(definitionType))
		}
	default:
		// String, Data Source, Secret and unknown types have no value format to enforce.
	}
	return nil
}

func environmentVariableSecretStoreLabel(value int64) string {
	switch value {
	case 0:
		return "Azure Key Vault"
	case 1:
		return "Microsoft Dataverse"
	default:
		return fmt.Sprintf("%d", value)
	}
}
