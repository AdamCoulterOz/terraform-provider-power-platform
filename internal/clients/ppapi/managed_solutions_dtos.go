// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

type ConnectionArrayDto struct {
	Value []ConnectionDto `json:"value"`
}

type ConnectionDto struct {
	Name       string                  `json:"name"`
	Properties ConnectionPropertiesDto `json:"properties"`
}

type ConnectionPropertiesDto struct {
	ApiId       string `json:"apiId"`
	DisplayName string `json:"displayName"`
}
