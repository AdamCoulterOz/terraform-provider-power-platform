// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

// TenantApplicationDto is a tenant-level Dynamics 365 application package.
type TenantApplicationDto struct {
	ApplicationDescription string                            `json:"applicationDescription"`
	ApplicationId          string                            `json:"applicationId"`
	ApplicationName        string                            `json:"applicationName"`
	ApplicationVisibility  string                            `json:"applicationVisibility"`
	CatalogVisibility      string                            `json:"catalogVisibility"`
	LastError              *TenantApplicationErrorDetailsDto `json:"errorDetails,omitempty"`
	LearnMoreUrl           string                            `json:"learnMoreUrl"`
	LocalizedDescription   string                            `json:"localizedDescription"`
	LocalizedName          string                            `json:"localizedName"`
	PublisherId            string                            `json:"publisherId"`
	PublisherName          string                            `json:"publisherName"`
	UniqueName             string                            `json:"uniqueName"`
}

// TenantApplicationErrorDetailsDto describes the last error reported for a
// tenant application package.
type TenantApplicationErrorDetailsDto struct {
	ErrorCode  string `json:"errorCode"`
	ErrorName  string `json:"errorName"`
	Message    string `json:"message"`
	Source     string `json:"source"`
	StatusCode int64  `json:"statusCode"`
	Type       string `json:"type"`
}

// TenantApplicationArrayDto is the envelope for a tenant application packages
// listing.
type TenantApplicationArrayDto struct {
	Value []TenantApplicationDto `json:"value"`
}

// EnvironmentApplicationArrayDto is the envelope for an environment
// application packages listing.
type EnvironmentApplicationArrayDto struct {
	Value []EnvironmentApplicationDto `json:"value"`
}

// EnvironmentApplicationDto is an application package in an environment.
type EnvironmentApplicationDto struct {
	ApplicationId         string `json:"applicationId"`
	Name                  string `json:"applicationName"`
	UniqueName            string `json:"uniqueName"`
	Version               string `json:"version"`
	Description           string `json:"localizedDescription"`
	PublisherId           string `json:"publisherId"`
	PublisherName         string `json:"publisherName"`
	LearnMoreUrl          string `json:"learnMoreUrl"`
	State                 string `json:"state"`
	ApplicationVisibility string `json:"applicationVisibility"`
}

// EnvironmentApplicationLifecycleCreatedDto is the synchronous creation
// response body of an application package install.
type EnvironmentApplicationLifecycleCreatedDto struct {
	Name       string                                              `json:"name"`
	Properties EnvironmentApplicationLifecycleCreatedPropertiesDto `json:"properties"`
}

// EnvironmentApplicationLifecycleCreatedPropertiesDto carries the provisioning
// state of a synchronously created application package install.
type EnvironmentApplicationLifecycleCreatedPropertiesDto struct {
	ProvisioningState string `json:"provisioningState"`
}

// EnvironmentApplicationLifecycleDto is the polled state of an application
// package install operation.
type EnvironmentApplicationLifecycleDto struct {
	OperationId        string                                  `json:"operationId"`
	CreatedDateTime    string                                  `json:"createdDateTime"`
	LastActionDateTime string                                  `json:"lastActionDateTime"`
	Status             string                                  `json:"status"`
	StatusMessage      string                                  `json:"statusMessage"`
	Error              EnvironmentApplicationLifecycleErrorDto `json:"error"`
}

// EnvironmentApplicationLifecycleErrorDto describes a failed application
// package install operation.
type EnvironmentApplicationLifecycleErrorDto struct {
	ErrorName  string `json:"errorName"`
	ErrorCode  int    `json:"errorCode"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	StatusCode int    `json:"statusCode"`
	Source     string `json:"source"`
}
