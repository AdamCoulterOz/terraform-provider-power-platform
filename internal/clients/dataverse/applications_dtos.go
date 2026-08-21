// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

// ApplicationUsersResponseDto is the OData envelope for a systemusers query.
type ApplicationUsersResponseDto struct {
	Value []ApplicationUserDto `json:"value"`
}

// ApplicationUserDto is a Dataverse application user (systemuser row) with its
// associated security roles.
type ApplicationUserDto struct {
	FullName       string                       `json:"fullname"`
	ApplicationId  string                       `json:"applicationid"`
	SystemUserId   string                       `json:"systemuserid"`
	BusinessUnitId string                       `json:"_businessunitid_value"`
	IsDisabled     bool                         `json:"isdisabled"`
	DeletedState   int64                        `json:"deletedstate"`
	SecurityRoles  []ApplicationSecurityRoleDto `json:"systemuserroles_association,omitempty"`
}

// ApplicationSecurityRoleDto is a Dataverse security role.
type ApplicationSecurityRoleDto struct {
	RoleId         string `json:"roleid"`
	Name           string `json:"name"`
	BusinessUnitId string `json:"_businessunitid_value"`
}

// ApplicationSecurityRoleArrayDto is the OData envelope for a roles query.
type ApplicationSecurityRoleArrayDto struct {
	Value []ApplicationSecurityRoleDto `json:"value"`
}

// ApplicationBusinessUnitArrayDto is the OData envelope for a businessunits
// query.
type ApplicationBusinessUnitArrayDto struct {
	Value []ApplicationBusinessUnitDto `json:"value"`
}

// ApplicationBusinessUnitDto is a Dataverse business unit.
type ApplicationBusinessUnitDto struct {
	BusinessUnitId string `json:"businessunitid"`
	Name           string `json:"name"`
}
