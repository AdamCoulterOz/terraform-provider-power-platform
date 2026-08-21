// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerapps

type ConnectorDto struct {
	Name       string                 `json:"name"`
	Id         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties ConnectorPropertiesDto `json:"properties"`
}

type ConnectorPropertiesDto struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Tier        string `json:"tier"`
	Publisher   string `json:"publisher"`
	Unblockable bool
}

type ConnectorArrayDto struct {
	Value []ConnectorDto `json:"value"`
}
