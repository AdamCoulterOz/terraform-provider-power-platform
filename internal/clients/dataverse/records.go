// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetEntityDefinition reads the entity definition of a table by its logical name.
func (client *Client) GetEntityDefinition(ctx context.Context, environmentHost, entityLogicalName string) (*EntityDefinitionsDto, error) {
	entityDefinitionApiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     environmentHost,
		Path:     fmt.Sprintf("/api/data/%s/EntityDefinitions(LogicalName='%s')", constants.DATAVERSE_API_VERSION, entityLogicalName),
		Fragment: "$select=PrimaryIdAttribute,LogicalCollectionName",
	}

	entityDefinition := EntityDefinitionsDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", entityDefinitionApiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &entityDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity definition for %s: %w", entityLogicalName, err)
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	return &entityDefinition, nil
}

// GetDataRecordsByODataQuery executes an OData query against the Dataverse Web
// API and returns the parsed rows plus paging annotations.
func (client *Client) GetDataRecordsByODataQuery(ctx context.Context, environmentHost, query string, headers map[string]string) (*ODataQueryResponse, error) {
	var h = make(http.Header)
	for k, v := range headers {
		h.Add(k, v)
	}

	apiUrl := fmt.Sprintf("https://%s/api/data/%s/%s", environmentHost, constants.DATAVERSE_API_VERSION, query)

	response := map[string]any{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl, h, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to execute OData query: %w", err)
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	var totalRecords *int64
	if rawCount, exists := response["@Microsoft.Dynamics.CRM.totalrecordcount"]; exists && rawCount != nil {
		if count, ok := rawCount.(float64); ok {
			countInt := int64(count)
			totalRecords = &countInt
		}
	}
	var totalRecordsCountLimitExceeded *bool
	if val, ok := response["@Microsoft.Dynamics.CRM.totalrecordcountlimitexceeded"].(bool); ok {
		isLimitExceeded := val
		totalRecordsCountLimitExceeded = &isLimitExceeded
	}

	records := []map[string]any{}
	if response["value"] != nil {
		valueSlice, ok := response["value"].([]any)
		if !ok {
			return nil, errors.New("value field is not of type []any")
		}
		for _, item := range valueSlice {
			value, ok := item.(map[string]any)
			if !ok {
				return nil, errors.New("item is not of type map[string]any")
			}
			records = append(records, value)
		}
	} else {
		records = append(records, response)
	}

	// Safe extraction of @odata.context
	odataCtxRaw, exists := response["@odata.context"]
	if !exists {
		return nil, errors.New("@odata.context field missing from response")
	}
	odataCtx, ok := odataCtxRaw.(string)
	if !ok {
		return nil, errors.New("@odata.context field is not a string")
	}
	splitParts := strings.Split(odataCtx, "#")
	if len(splitParts) < 2 {
		return nil, errors.New("@odata.context string is malformed")
	}
	pluralName := splitParts[1]
	if index := strings.IndexAny(pluralName, "(/"); index != -1 {
		pluralName = pluralName[:index]
	}

	return &ODataQueryResponse{
		Records:                  records,
		TotalRecord:              totalRecords,
		TotalRecordLimitExceeded: totalRecordsCountLimitExceeded,
		TableMetadataUrl:         odataCtx,
		// url will be as example: https://org.crm4.dynamics.com/api/data/v9.2/$metadata#tablepluralname/$entity.
		TablePluralName: pluralName,
	}, nil
}

// GetDataRecord reads a single data record by its id.
func (client *Client) GetDataRecord(ctx context.Context, recordId, environmentHost, tableName string) (map[string]any, error) {
	entityDefinition, err := client.GetEntityDefinition(ctx, environmentHost, tableName)
	if err != nil {
		return nil, err
	}

	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/%s(%s)", constants.DATAVERSE_API_VERSION, entityDefinition.LogicalCollectionName, recordId),
	}

	result := make(map[string]any, 0)

	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden}, &result)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	return result, nil
}

// GetRelationData reads the related records of a record's navigation property.
func (client *Client) GetRelationData(ctx context.Context, environmentHost, tableName, recordId, relationName string) ([]any, error) {
	entityDefinition, err := client.GetEntityDefinition(ctx, environmentHost, tableName)
	if err != nil {
		return nil, err
	}

	apiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     environmentHost,
		Path:     fmt.Sprintf("/api/data/%s/%s(%s)/%s", constants.DATAVERSE_API_VERSION, entityDefinition.LogicalCollectionName, recordId, relationName),
		RawQuery: "$select=createdon",
	}

	result := make(map[string]any, 0)

	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &result)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}

	field, ok := result["value"]
	if !ok {
		return nil, errors.New("value field not found in result when retrieving relational data")
	}

	value, ok := field.([]any)
	if !ok {
		return nil, errors.New("value field is not of type []any in relational data")
	}

	return value, nil
}

// GetTableSingularNameFromPlural resolves a table's logical name from its
// logical collection name.
func (client *Client) GetTableSingularNameFromPlural(ctx context.Context, environmentHost, logicalCollectionName string) (*string, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/EntityDefinitions", constants.DATAVERSE_API_VERSION),
	}
	q := apiUrl.Query()
	q.Add("$filter", fmt.Sprintf("LogicalCollectionName eq '%s'", logicalCollectionName))
	q.Add("$select", "PrimaryIdAttribute,LogicalCollectionName,LogicalName")
	apiUrl.RawQuery = q.Encode()

	response, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(response); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(response); err != nil {
		return nil, err
	}

	var mapResponse map[string]any
	err = json.Unmarshal(response.BodyAsBytes, &mapResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response for table singular name: %w", err)
	}

	var result string
	if rawVal, exists := mapResponse["value"]; exists && rawVal != nil {
		valueSlice, ok := rawVal.([]any)
		if !ok {
			return nil, errors.New("value field is not of type []any")
		}
		if len(valueSlice) > 0 {
			if value, ok := valueSlice[0].(map[string]any); ok {
				if logicalName, ok := value["LogicalName"].(string); ok {
					result = logicalName
				}
			}
		}
	} else if logicalName, ok := mapResponse["LogicalName"].(string); ok {
		result = logicalName
	} else {
		return nil, errors.New("logicalName field not found in result when retrieving table singular name")
	}
	return &result, nil
}

func getEntityRelationDefinitionOneToMany(mapResponse map[string]any, entityLogicalName, relationLogicalName string) (string, error) {
	var tableName string
	oneToMany, ok := mapResponse["OneToManyRelationships"].([]any)
	if !ok {
		return "", errors.New("OneToManyRelationships field is not of type []any")
	}
	for _, list := range oneToMany {
		item, ok := list.(map[string]any)
		if !ok {
			return "", errors.New("item is not of type map[string]any")
		}
		if item["ReferencingEntityNavigationPropertyName"] == relationLogicalName && item["Entity1LogicalName"] != entityLogicalName {
			var ok bool
			tableName, ok = item["ReferencedEntity"].(string)
			if !ok {
				return "", errors.New("ReferencedEntity field is not of type string")
			}
			break
		}
		if item["ReferencedEntityNavigationPropertyName"] == relationLogicalName && item["Entity2LogicalName"] != entityLogicalName {
			var ok bool
			tableName, ok = item["ReferencingEntity"].(string)
			if !ok {
				return "", errors.New("ReferencedEntity field is not of type string")
			}
			break
		}
	}
	return tableName, nil
}

func getEntityRelationDefinitionManyToOne(mapResponse map[string]any, relationLogicalName string) (string, error) {
	var tableName string
	manyToOne, ok := mapResponse["ManyToOneRelationships"].([]any)
	if !ok {
		return "", errors.New("ManyToOneRelationships field is not of type []any")
	}
	for _, list := range manyToOne {
		item, ok := list.(map[string]any)
		if !ok {
			return "", errors.New("item is not of type map[string]any")
		}
		if item["ReferencingEntityNavigationPropertyName"] == relationLogicalName {
			var ok bool
			tableName, ok = item["ReferencedEntity"].(string)
			if !ok {
				return "", errors.New("ReferencedEntity field is not of type string")
			}
			break
		}
		if item["ReferencedEntityNavigationPropertyName"] == relationLogicalName {
			var ok bool
			tableName, ok = item["ReferencingEntity"].(string)
			if !ok {
				return "", errors.New("ReferencedEntity field is not of type string")
			}
			break
		}
	}
	return tableName, nil
}

func getEntityRelationDefinitionManyToMany(mapResponse map[string]any, entityLogicalName, relationLogicalName string) (string, error) {
	var tableName string
	manyToMany, ok := mapResponse["ManyToManyRelationships"].([]any)
	if !ok {
		return "", errors.New("ManyToManyRelationships field is not of type []any")
	}
	for _, list := range manyToMany {
		item, ok := list.(map[string]any)
		if !ok {
			return "", errors.New("item is not of type map[string]any")
		}
		if item["Entity1NavigationPropertyName"] == relationLogicalName && item["Entity1LogicalName"] != entityLogicalName {
			tableName, ok = item["Entity1LogicalName"].(string)
			if !ok {
				return "", errors.New("Entity1LogicalName field is not of type string")
			}
			break
		}
		if item["Entity2NavigationPropertyName"] == relationLogicalName && item["Entity2LogicalName"] != entityLogicalName {
			tableName, ok = item["Entity2LogicalName"].(string)
			if !ok {
				return "", errors.New("Entity2LogicalName field is not of type string")
			}
			break
		}
	}
	return tableName, nil
}

// GetEntityAttributesDefinition lists the attribute logical names of a table.
func (client *Client) GetEntityAttributesDefinition(ctx context.Context, environmentHost string, entityLogicalName string) ([]AttributesApiBodyDto, error) {
	apiUrl := fmt.Sprintf("https://%s/api/data/%s/EntityDefinitions(LogicalName='%s')/Attributes?$select=LogicalName", environmentHost, constants.DATAVERSE_API_VERSION, entityLogicalName)

	results := AttributesApiResponseDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &results)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, err
	}
	return results.Value, nil
}

// GetEntityRelationDefinitionInfo resolves the table that a relation
// (navigation property) of an entity points at.
func (client *Client) GetEntityRelationDefinitionInfo(ctx context.Context, environmentHost string, entityLogicalName string, relationLogicalName string) (tableName string, err error) {
	apiUrl := fmt.Sprintf("https://%s/api/data/%s/EntityDefinitions(LogicalName='%s')?$expand=OneToManyRelationships,ManyToManyRelationships,ManyToOneRelationships", environmentHost, constants.DATAVERSE_API_VERSION, entityLogicalName)

	response, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return "", err
	}
	if err := client.Api.HandleForbiddenResponse(response); err != nil {
		return "", err
	}
	if err := client.Api.HandleNotFoundResponse(response); err != nil {
		return "", err
	}

	var mapResponse map[string]any
	err = json.Unmarshal(response.BodyAsBytes, &mapResponse)
	if err != nil {
		return "", err
	}

	tableName, err = getEntityRelationDefinitionOneToMany(mapResponse, entityLogicalName, relationLogicalName)
	if err != nil {
		return "", err
	}
	if tableName != "" {
		return tableName, nil
	}

	tableName, err = getEntityRelationDefinitionManyToOne(mapResponse, relationLogicalName)
	if err != nil {
		return "", err
	}
	if tableName != "" {
		return tableName, nil
	}

	tableName, err = getEntityRelationDefinitionManyToMany(mapResponse, entityLogicalName, relationLogicalName)
	if err != nil {
		return "", err
	}

	return tableName, nil
}

// ApplyDataRecord creates or updates a data record, including its
// single-valued (@odata.bind) and collection-valued ($ref) relations.
func (client *Client) ApplyDataRecord(ctx context.Context, recordId, environmentHost, tableName string, columns map[string]any) (*DataRecordDto, error) {
	result := DataRecordDto{}

	relations := make(map[string]any, 0)

	for key, value := range columns {
		if nestedMap, ok := value.(map[string]any); ok {
			delete(columns, key)
			if len(nestedMap) > 0 {
				tableLogicalName, dataRecordId, err := getTableLogicalNameAndDataRecordIdFromMap(nestedMap)
				if err != nil {
					return nil, err
				}

				entityDefinition, err := client.GetEntityDefinition(ctx, environmentHost, tableLogicalName)
				if err != nil {
					return nil, err
				}

				columns[fmt.Sprintf("%s@odata.bind", key)] = fmt.Sprintf("/%s(%s)", entityDefinition.LogicalCollectionName, dataRecordId)
			}
		} else if nestedMapList, ok := value.([]any); ok {
			delete(columns, key)
			relations[key] = nestedMapList
		}
	}

	entityDefinition, err := client.GetEntityDefinition(ctx, environmentHost, tableName)
	if err != nil {
		return nil, err
	}

	// we will send create operation as default.
	method := "POST"
	apiPath := fmt.Sprintf("/api/data/%s/%s", constants.DATAVERSE_API_VERSION, entityDefinition.LogicalCollectionName)

	if val, ok := columns[entityDefinition.PrimaryIDAttribute]; ok {
		// if one of the sent attributes is the primaryId then send an update.
		method = "PATCH"
		apiPath = fmt.Sprintf("%s(%s)", apiPath, val)
	} else if recordId != "" {
		// if we are referencing the record by its primaryId and its not empty (update or delete) then we send an update
		method = "PATCH"
		apiPath = fmt.Sprintf("%s(%s)", apiPath, recordId)
	}

	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   apiPath,
	}

	response, err := client.Api.Execute(ctx, scopes(environmentHost), method, apiUrl.String(), nil, columns, []int{http.StatusOK, http.StatusNoContent, http.StatusPreconditionFailed, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return nil, err
	}
	if response.HttpResponse.StatusCode == http.StatusPreconditionFailed {
		// if record was already found then we will try to update it.
		response, err = client.Api.Execute(ctx, scopes(environmentHost), "PATCH", apiUrl.String(), nil, columns, []int{http.StatusOK, http.StatusNoContent}, nil)
		// if the PATCH URL is not pointing to specific recording using PrimaryIDAttribute then we have to inform a user about it.
		if response.HttpResponse.StatusCode == http.StatusMethodNotAllowed {
			return nil, errors.New("record already exists. To update an existing record, primaryId must be provided in the columns attribute")
		}
		if err != nil {
			return nil, err
		}
	}
	if err := client.Api.HandleForbiddenResponse(response); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(response); err != nil {
		return nil, err
	}

	if len(response.BodyAsBytes) != 0 {
		err = json.Unmarshal(response.BodyAsBytes, &result)
		if err != nil {
			return nil, err
		}
	} else if response.HttpResponse.Header.Get(constants.HEADER_ODATA_ENTITY_ID) != "" {
		re := regexp.MustCompile("[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
		match := re.FindAllStringSubmatch(response.HttpResponse.Header.Get(constants.HEADER_ODATA_ENTITY_ID), -1)
		if len(match) == 0 {
			return nil, errors.New("no entity record id returned from the odata-entityid header")
		}
		result.Id = match[len(match)-1][0]
	} else {
		return nil, errors.New("no entity record id returned from the API")
	}

	err = applyRelations(ctx, client, relations, environmentHost, result.Id, entityDefinition)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteDataRecord deletes a data record after detaching its collection-valued
// relations.
func (client *Client) DeleteDataRecord(ctx context.Context, recordId string, environmentHost string, tableName string, columns map[string]any) error {
	tableEntityDefinition, err := client.GetEntityDefinition(ctx, environmentHost, tableName)
	if err != nil {
		return err
	}

	for key, value := range columns {
		if _, ok := value.(map[string]any); ok {
			delete(columns, key)
		}
		if nestedMapList, ok := value.([]any); ok {
			delete(columns, key)

			for _, nestedItem := range nestedMapList {
				nestedMap, ok := nestedItem.(map[string]any)
				if !ok {
					return errors.New("nestedItem is not of type map[string]any")
				}

				dataRecordId, ok := nestedMap["data_record_id"].(string)
				if !ok {
					return errors.New("data_record_id field is missing or not a string")
				}

				apiUrl := &url.URL{
					Scheme: constants.HTTPS,
					Host:   environmentHost,
					Path:   fmt.Sprintf("/api/data/%s/%s(%s)/%s(%s)/$ref", constants.DATAVERSE_API_VERSION, tableEntityDefinition.LogicalCollectionName, recordId, key, dataRecordId),
				}
				resp, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound, http.StatusForbidden}, nil)
				if err != nil {
					return fmt.Errorf("error while deleting data record: %w", err)
				}
				if err := client.Api.HandleForbiddenResponse(resp); err != nil {
					return err
				}
			}
		}
	}

	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/%s(%s)", constants.DATAVERSE_API_VERSION, tableEntityDefinition.LogicalCollectionName, recordId),
	}

	// 200, 201, or 404 are acceptable status codes for delete and not error
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), "DELETE", apiUrl.String(), nil, columns, []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound, http.StatusForbidden}, nil)
	if err != nil {
		return err
	} else if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	}
	return nil
}

func getTableLogicalNameAndDataRecordIdFromMap(nestedMap map[string]any) (tableLogicalName string, dataRecordId string, err error) {
	tableLogicalName, ok := nestedMap["table_logical_name"].(string)
	if !ok {
		return "", "", errors.New("table_logical_name field is missing or not a string")
	}
	id, ok := nestedMap["data_record_id"].(string)
	if !ok {
		return "", "", errors.New("data_record_id field is missing or not a string")
	}
	return tableLogicalName, id, nil
}

func applyRelations(ctx context.Context, client *Client, relations map[string]any, environmentHost string, parentRecordId string, entityDefinition *EntityDefinitionsDto) error {
	for key, value := range relations {
		if nestedMapList, ok := value.([]any); ok {
			err := applyRelation(ctx, environmentHost, entityDefinition, parentRecordId, key, client, nestedMapList)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func applyRelation(ctx context.Context, environmentHost string, entityDefinition *EntityDefinitionsDto, parentRecordId string, key string, client *Client, nestedMapList []any) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   environmentHost,
		Path:   fmt.Sprintf("/api/data/%s/%s(%s)/%s/$ref", constants.DATAVERSE_API_VERSION, entityDefinition.LogicalCollectionName, parentRecordId, key),
	}

	existingRelationsResponse := RelationApiResponseDto{}

	apiResponse, err := client.Api.Execute(ctx, scopes(environmentHost), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
	if err != nil {
		return err
	}
	if err := client.Api.HandleForbiddenResponse(apiResponse); err != nil {
		return err
	}
	if err := client.Api.HandleNotFoundResponse(apiResponse); err != nil {
		return err
	}

	// TODO: execute will unmarshal the response into the existingRelationsResponse use that instead
	err = json.Unmarshal(apiResponse.BodyAsBytes, &existingRelationsResponse)
	if err != nil {
		return err
	}

	var toBeDeleted = make([]RelationApiBodyDto, 0)

	for _, existingRelation := range existingRelationsResponse.Value {
		shouldDelete := true
		for _, nestedItem := range nestedMapList {
			nestedMap, ok := nestedItem.(map[string]any)
			if !ok {
				return errors.New("nestedItem is not of type map[string]any")
			}

			tableLogicalName, dataRecordId, err := getTableLogicalNameAndDataRecordIdFromMap(nestedMap)
			if err != nil {
				return err
			}

			relationEntityDefinition, err := client.GetEntityDefinition(ctx, environmentHost, tableLogicalName)
			if err != nil {
				return err
			}
			if existingRelation.OdataID == fmt.Sprintf("https://%s/api/data/%s/%s(%s)", environmentHost, constants.DATAVERSE_API_VERSION, relationEntityDefinition.LogicalCollectionName, dataRecordId) {
				shouldDelete = false
				break
			}
		}
		if shouldDelete {
			toBeDeleted = append(toBeDeleted, existingRelation)
		}
	}

	for _, relation := range toBeDeleted {
		_, err = client.Api.Execute(ctx, scopes(environmentHost), "DELETE", relation.OdataID, nil, nil, []int{http.StatusOK, http.StatusNoContent}, nil)
		if err != nil {
			return err
		}
	}

	for _, nestedItem := range nestedMapList {
		nestedMap, ok := nestedItem.(map[string]any)
		if !ok {
			return errors.New("nestedItem is not of type map[string]any")
		}

		tableLogicalName, dataRecordId, err := getTableLogicalNameAndDataRecordIdFromMap(nestedMap)
		if err != nil {
			return err
		}

		entityDefinition, err := client.GetEntityDefinition(ctx, environmentHost, tableLogicalName)
		if err != nil {
			return err
		}

		relation := RelationApiBodyDto{
			OdataID: fmt.Sprintf("https://%s/api/data/%s/%s(%s)", environmentHost, constants.DATAVERSE_API_VERSION, entityDefinition.LogicalCollectionName, dataRecordId),
		}
		resp, err := client.Api.Execute(ctx, scopes(environmentHost), "POST", apiUrl.String(), nil, relation, []int{http.StatusOK, http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}, nil)
		if err != nil {
			return err
		}
		if err := client.Api.HandleForbiddenResponse(resp); err != nil {
			return err
		}
		if err := client.Api.HandleNotFoundResponse(resp); err != nil {
			return err
		}
	}
	return nil
}
