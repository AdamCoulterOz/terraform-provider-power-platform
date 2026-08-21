// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

type UnblockableConnectorDto struct {
	Id       string                          `json:"id"`
	Metadata UnblockableConnectorMetadataDto `json:"metadata"`
}

type UnblockableConnectorMetadataDto struct {
	Unblockable bool `json:"unblockable"`
}

type VirtualConnectorDto struct {
	Id       string                      `json:"id"`
	Metadata VirtualConnectorMetadataDto `json:"metadata"`
}

type VirtualConnectorMetadataDto struct {
	VirtualConnector bool   `json:"virtualConnector"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	DisplayName      string `json:"displayName"`
}
