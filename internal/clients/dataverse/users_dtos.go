// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

// UserDto is a Dataverse systemuser record.
type UserDto struct {
	Id             string            `json:"systemuserid"`
	DomainName     string            `json:"domainname"`
	FirstName      string            `json:"firstname"`
	LastName       string            `json:"lastname"`
	AadObjectId    string            `json:"azureactivedirectoryobjectid"`
	BusinessUnitId string            `json:"_businessunitid_value"`
	SecurityRoles  []SecurityRoleDto `json:"systemuserroles_association,omitempty"`
}

// SecurityRolesArray returns the role ids of the user's security roles.
func (u *UserDto) SecurityRolesArray() []string {
	if len(u.SecurityRoles) == 0 {
		return []string{}
	}
	var roles []string
	for _, role := range u.SecurityRoles {
		roles = append(roles, role.RoleId)
	}
	return roles
}

// UserArrayDto is a collection of systemuser records.
type UserArrayDto struct {
	Value []UserDto `json:"value"`
}

// SecurityRoleDto is a Dataverse security role record.
type SecurityRoleDto struct {
	RoleId         string `json:"roleid"`
	Name           string `json:"name"`
	IsManaged      bool   `json:"ismanaged"`
	BusinessUnitId string `json:"_businessunitid_value"`
}

// SecurityRoleArrayDto is a collection of security role records.
type SecurityRoleArrayDto struct {
	Value []SecurityRoleDto `json:"value"`
}
