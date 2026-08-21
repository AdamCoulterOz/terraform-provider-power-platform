// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

// DTOs for the Power Platform Authorization RBAC API

type RoleAssignmentRequestDto struct {
	PrincipalObjectId string `json:"principalObjectId"`
	PrincipalType     string `json:"principalType"`
	RoleDefinitionId  string `json:"roleDefinitionId"`
	Scope             string `json:"scope"`
}

type RoleAssignmentDto struct {
	RoleAssignmentId           string  `json:"roleAssignmentId"`
	Scope                      string  `json:"scope"`
	PrincipalType              string  `json:"principalType"`
	PrincipalObjectId          string  `json:"principalObjectId"`
	RoleDefinitionId           string  `json:"roleDefinitionId"`
	CreatedByPrincipalType     string  `json:"createdByPrincipalType"`
	CreatedByPrincipalObjectId string  `json:"createdByPrincipalObjectId"`
	CreatedOn                  string  `json:"createdOn"`
	ExpiresOn                  *string `json:"expiresOn"`
}

type RoleAssignmentsListDto struct {
	Value []RoleAssignmentDto `json:"value"`
}

type RoleDefinitionDto struct {
	RoleDefinitionId   string   `json:"roleDefinitionId"`
	RoleDefinitionName string   `json:"roleDefinitionName"`
	Permissions        []string `json:"permissions"`
}

type RoleDefinitionsListDto struct {
	Value []RoleDefinitionDto `json:"value"`
}
