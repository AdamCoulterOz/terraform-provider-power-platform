// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.
package data_record

type dataRecordDto struct {
	Id           string `json:"id"`
	OdataContext string `json:"@odata.context"`
	OdataEtag    string `json:"@odata.etag"`
}

type entityDefinitionsDto struct {
	OdataContext          string `json:"@odata.context"`
	PrimaryIDAttribute    string `json:"PrimaryIdAttribute"`
	LogicalCollectionName string `json:"LogicalCollectionName"`
	MetadataID            string `json:"MetadataId"`
}

type relationApiResponseDto struct {
	OdataContext string               `json:"@odata.context"`
	Value        []relationApiBodyDto `json:"value"`
}

type relationApiBodyDto struct {
	OdataID string `json:"@odata.id"`
}

type attributesApiResponseDto struct {
	OdataContext string                 `json:"@odata.context"`
	Value        []attributesApiBodyDto `json:"value"`
}
type attributesApiBodyDto struct {
	LogicalName string `json:"LogicalName"`
	MetadataId  string `json:"MetadataId"`
}
