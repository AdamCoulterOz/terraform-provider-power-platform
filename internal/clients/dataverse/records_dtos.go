// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

// DataRecordDto is the wire shape of a created/updated data record response.
type DataRecordDto struct {
	Id           string `json:"id"`
	OdataContext string `json:"@odata.context"`
	OdataEtag    string `json:"@odata.etag"`
}

// EntityDefinitionsDto is the wire shape of an EntityDefinitions response.
type EntityDefinitionsDto struct {
	OdataContext          string `json:"@odata.context"`
	PrimaryIDAttribute    string `json:"PrimaryIdAttribute"`
	LogicalCollectionName string `json:"LogicalCollectionName"`
	MetadataID            string `json:"MetadataId"`
}

// RelationApiResponseDto is the wire shape of a relation $ref collection response.
type RelationApiResponseDto struct {
	OdataContext string               `json:"@odata.context"`
	Value        []RelationApiBodyDto `json:"value"`
}

// RelationApiBodyDto is the wire shape of a single relation $ref entry.
type RelationApiBodyDto struct {
	OdataID string `json:"@odata.id"`
}

// AttributesApiResponseDto is the wire shape of an entity Attributes response.
type AttributesApiResponseDto struct {
	OdataContext string                 `json:"@odata.context"`
	Value        []AttributesApiBodyDto `json:"value"`
}

// AttributesApiBodyDto is the wire shape of a single entity attribute.
type AttributesApiBodyDto struct {
	LogicalName string `json:"LogicalName"`
	MetadataId  string `json:"MetadataId"`
}

// ODataQueryResponse is the parsed result of an OData query against the
// Dataverse Web API.
type ODataQueryResponse struct {
	Records                  []map[string]any
	TotalRecord              *int64
	TotalRecordLimitExceeded *bool
	TableMetadataUrl         string
	TablePluralName          string
}
