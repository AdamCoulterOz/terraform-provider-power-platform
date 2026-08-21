// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

// DTOs for the Power Platform API connectivity (connections) surface, served
// from the environment-scoped hosts ({environmentId}.environment.api.powerplatform.com).

type ConnectionArrayDto struct {
	Value []ConnectionDto `json:"value"`
}

type ConnectionDto struct {
	Name       string                  `json:"name"`
	Id         string                  `json:"id"`
	Type       string                  `json:"type"`
	Properties ConnectionPropertiesDto `json:"properties"`
}

type ConnectionPropertiesDto struct {
	ApiId                   string                    `json:"apiId"`
	DisplayName             string                    `json:"displayName"`
	IconUri                 string                    `json:"iconUri"`
	Statuses                []ConnectionStatusDto     `json:"statuses"`
	ConnectionParametersSet map[string]any            `json:"connectionParametersSet,omitempty"`
	ConnectionParameters    map[string]any            `json:"connectionParameters,omitempty"`
	KeywordsRemaining       int                       `json:"keywordsRemaining"`
	CreatedBy               ConnectionCreatedByDto    `json:"createdBy"`
	CreatedTime             string                    `json:"createdTime"`
	LastModifiedTime        string                    `json:"lastModifiedTime"`
	ExpirationTime          string                    `json:"expirationTime"`
	TestLinks               []ConnectionTestLinkDto   `json:"testLinks"`
	Environment             ConnectionEnvironmentDto  `json:"environment"`
	Permissions             []ConnectionPermissionDto `json:"permissions"`
	AccountName             string                    `json:"accountName"`
	AllowSharing            bool                      `json:"allowSharing"`
}

type ConnectionStatusDto struct {
	Status string `json:"status"`
}

type ConnectionCreatedByDto struct {
	Id                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Email             string `json:"email"`
	Type              string `json:"type"`
	TenantId          string `json:"tenantId"`
	UserPrincipalName string `json:"userPrincipalName"`
}

type ConnectionTestLinkDto struct {
	RequestURI string `json:"requestUri"`
	Method     string `json:"method"`
}

type ConnectionEnvironmentDto struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ConnectionPermissionDto struct {
	Name       string                            `json:"name"`
	Id         string                            `json:"id"`
	Type       string                            `json:"type"`
	Properties ConnectionPermissionPropertiesDto `json:"properties"`
}

type ConnectionPermissionPropertiesDto struct {
	RoleName                string                 `json:"roleName"`
	Principal               ConnectionPrincipalDto `json:"principal"`
	NotifyShareTargetOption string                 `json:"NotifyShareTargetOption"`
	InviteGuestToTenant     bool                   `json:"inviteGuestToTenant"`
}

type ConnectionPrincipalDto struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	TenantId string `json:"tenantId"`
}

type ConnectionCreateDto struct {
	Properties ConnectionCreatePropertiesDto `json:"properties"`
}

type ConnectionCreatePropertiesDto struct {
	DisplayName             string                         `json:"displayName"`
	ConnectionParametersSet map[string]any                 `json:"connectionParametersSet,omitempty"`
	ConnectionParameters    map[string]any                 `json:"connectionParameters,omitempty"`
	Environment             ConnectionCreateEnvironmentDto `json:"environment"`
}

type ConnectionCreateEnvironmentDto struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ShareConnectionRequestDto struct {
	Put    []ShareConnectionRequestPutDto    `json:"put"`
	Delete []ShareConnectionRequestDeleteDto `json:"delete"`
}

type ShareConnectionRequestPutDto struct {
	Properties ShareConnectionRequestPutPropertiesDto `json:"properties"`
}

type ShareConnectionRequestPutPropertiesDto struct {
	RoleName                string                                          `json:"roleName"`
	Capabilities            []any                                           `json:"capabilities"`
	Principal               ShareConnectionRequestPutPropertiesPrincipalDto `json:"principal"`
	NotifyShareTargetOption string                                          `json:"NotifyShareTargetOption"`
}

type ShareConnectionRequestPutPropertiesPrincipalDto struct {
	Id       string  `json:"id"`
	Type     string  `json:"type"`
	TenantId *string `json:"tenantId"`
}

type ShareConnectionRequestDeleteDto struct {
	Id string `json:"id"`
}

type ShareConnectionResponseArrayDto struct {
	Value []ShareConnectionResponseDto `json:"value"`
}

type ShareConnectionResponseDto struct {
	Name       string                               `json:"name"`
	Id         string                               `json:"id"`
	Type       string                               `json:"type"`
	Properties ShareConnectionResponsePropertiesDto `json:"properties"`
}

type ShareConnectionResponsePropertiesDto struct {
	RoleName                string         `json:"roleName"`
	Principal               map[string]any `json:"principal"`
	NotifyShareTargetOption string         `json:"NotifyShareTargetOption"`
	InviteGuestToTenant     bool           `json:"inviteGuestToTenant"`
}
