// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package powerapps

type PowerAppBapiDto struct {
	Name       string                    `json:"name"`
	Properties PowerAppPropertiesBapiDto `json:"properties"`
}

type PowerAppPropertiesBapiDto struct {
	DisplayName      string                 `json:"displayName"`
	Owner            PowerAppCreatedByDto   `json:"owner"`
	CreatedBy        PowerAppCreatedByDto   `json:"createdBy"`
	LastModifiedBy   PowerAppCreatedByDto   `json:"lastModifiedBy"`
	LastPublishedBy  PowerAppCreatedByDto   `json:"lastPublishedBy"`
	CreatedTime      string                 `json:"createdTime"`
	LastModifiedTime string                 `json:"lastModifiedTime"`
	LastPublishTime  string                 `json:"lastPublishTime"`
	Environment      PowerAppEnvironmentDto `json:"environment"`
}

type PowerAppEnvironmentDto struct {
	Id       string `json:"id"`
	Location string `json:"location"`
	Name     string `json:"name"`
}

type PowerAppCreatedByDto struct {
	DisplayName       string `json:"displayName"`
	Id                string `json:"id"`
	UserPrincipalName string `json:"userPrincipalName"`
}

type PowerAppArrayDto struct {
	Value []PowerAppBapiDto `json:"value"`
}
