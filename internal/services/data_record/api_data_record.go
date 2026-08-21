// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package data_record

import (
	"context"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

func newDataRecordClient(apiClient *api.Client) client {
	return client{
		Client:          bapi.NewClient(apiClient),
		dataverseClient: dataverse.NewClient(apiClient),
	}
}

type client struct {
	*bapi.Client
	dataverseClient *dataverse.Client
}

func getEntityDefinition(ctx context.Context, client *client, environmentId, entityLogicalName string) (*dataverse.EntityDefinitionsDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverseClient.GetEntityDefinition(ctx, environmentHost, entityLogicalName)
}

func (client *client) GetDataRecordsByODataQuery(ctx context.Context, environmentId, query string, headers map[string]string) (*dataverse.ODataQueryResponse, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverseClient.GetDataRecordsByODataQuery(ctx, environmentHost, query, headers)
}

func (client *client) GetDataRecord(ctx context.Context, recordId, environmentId, tableName string) (map[string]any, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverseClient.GetDataRecord(ctx, recordId, environmentHost, tableName)
}

func (client *client) GetRelationData(ctx context.Context, environmentId, tableName, recordId, relationName string) ([]any, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverseClient.GetRelationData(ctx, environmentHost, tableName, recordId, relationName)
}

func (client *client) GetTableSingularNameFromPlural(ctx context.Context, environmentId, logicalCollectionName string) (*string, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverseClient.GetTableSingularNameFromPlural(ctx, environmentHost, logicalCollectionName)
}

func (client *client) GetEntityAttributesDefinition(ctx context.Context, environmentId string, entityLogicalName string) ([]dataverse.AttributesApiBodyDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverseClient.GetEntityAttributesDefinition(ctx, environmentHost, entityLogicalName)
}

func (client *client) GetEntityRelationDefinitionInfo(ctx context.Context, environmentId string, entityLogicalName string, relationLogicalName string) (tableName string, err error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return "", err
	}
	return client.dataverseClient.GetEntityRelationDefinitionInfo(ctx, environmentHost, entityLogicalName, relationLogicalName)
}

func (client *client) ApplyDataRecord(ctx context.Context, recordId, environmentId, tableName string, columns map[string]any) (*dataverse.DataRecordDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	return client.dataverseClient.ApplyDataRecord(ctx, recordId, environmentHost, tableName, columns)
}

func (client *client) DeleteDataRecord(ctx context.Context, recordId string, environmentId string, tableName string, columns map[string]any) error {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}
	return client.dataverseClient.DeleteDataRecord(ctx, recordId, environmentHost, tableName, columns)
}
