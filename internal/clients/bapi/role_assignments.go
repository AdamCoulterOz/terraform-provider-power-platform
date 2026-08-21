// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

// Role definition display names of the built-in environment roles that can be
// assigned through BAPI role assignments.
const (
	RoleEnvironmentAdmin = "Environment Admin"
	RoleEnvironmentMaker = "Environment Maker"
)

const environmentUserDisplayNameRetryCount = 5

// RoleDefinitionDto identifies a BAPI role definition or role assignment.
type RoleDefinitionDto struct {
	Id   string `json:"id"`
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// PrincipalDto is the principal of a BAPI role assignment.
type PrincipalDto struct {
	Id          string `json:"id"`
	Email       string `json:"email,omitempty"`
	Type        string `json:"type"`
	TenantId    string `json:"tenantId,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// PropertiesDto are the properties of a BAPI role assignment.
type PropertiesDto struct {
	Scope          string            `json:"scope,omitempty"`
	RoleDefinition RoleDefinitionDto `json:"roleDefinition"`
	Principal      PrincipalDto      `json:"principal"`
}

// AddItemRequestDto is one role assignment to add in a modifyRoleAssignments
// request.
type AddItemRequestDto struct {
	Properties PropertiesDto `json:"properties,omitempty"`
}

// EnvironmentUserRequestDto is the body of a modifyRoleAssignments request.
type EnvironmentUserRequestDto struct {
	Add    []AddItemRequestDto `json:"add,omitempty"`
	Remove []RoleDefinitionDto `json:"remove,omitempty"`
}

// RoleAssignmentDto is a BAPI environment role assignment.
type RoleAssignmentDto struct {
	Id         string        `json:"id"`
	Type       string        `json:"type"`
	Name       string        `json:"name"`
	Properties PropertiesDto `json:"properties"`
}

// ItemResponseDto is one role assignment result in a modifyRoleAssignments
// response.
type ItemResponseDto struct {
	RoleAssignment RoleAssignmentDto `json:"roleAssignment"`
	HttpStatus     string            `json:"httpStatus"`
}

// EnvironmentUserResponseDto is the body of a modifyRoleAssignments response.
type EnvironmentUserResponseDto struct {
	Add    []ItemResponseDto `json:"add,omitempty"`
	Remove []ItemResponseDto `json:"remove,omitempty"`
}

// EnvironmentUserGetResponseDto is the body of a roleAssignments list
// response.
type EnvironmentUserGetResponseDto struct {
	Value []RoleAssignmentDto `json:"value"`
}

// EnvironmentUserDto describes an environment user assembled from BAPI role
// assignments. It intentionally duplicates the Dataverse systemuser wire
// shape (dataverse.UserDto): the same JSON travels on both boundaries and
// boundary packages must not import each other.
type EnvironmentUserDto struct {
	Id             string                           `json:"systemuserid"`
	DomainName     string                           `json:"domainname"`
	FirstName      string                           `json:"firstname"`
	LastName       string                           `json:"lastname"`
	AadObjectId    string                           `json:"azureactivedirectoryobjectid"`
	BusinessUnitId string                           `json:"_businessunitid_value"`
	SecurityRoles  []EnvironmentUserSecurityRoleDto `json:"systemuserroles_association,omitempty"`
}

// EnvironmentUserSecurityRoleDto is a security role held by an environment
// user. It intentionally duplicates the Dataverse security role wire shape
// (dataverse.SecurityRoleDto); see EnvironmentUserDto.
type EnvironmentUserSecurityRoleDto struct {
	RoleId         string `json:"roleid"`
	Name           string `json:"name"`
	IsManaged      bool   `json:"ismanaged"`
	BusinessUnitId string `json:"_businessunitid_value"`
}

// GetEnvironmentUserByAadObjectId reads the environment role assignments and
// assembles the environment user with the given Entra object id.
func (client *Client) GetEnvironmentUserByAadObjectId(ctx context.Context, environmentId, aadObjectId string) (*EnvironmentUserDto, error) {
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl := helpers.BuildBapiUrl(client.host(),
		fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/roleAssignments", environmentId),
		values)

	respObj := EnvironmentUserGetResponseDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl, nil, nil, []int{http.StatusOK}, &respObj)
	if err != nil {
		return nil, err
	}
	user := &EnvironmentUserDto{
		SecurityRoles: []EnvironmentUserSecurityRoleDto{},
	}

	for _, roleAssignment := range respObj.Value {
		if roleAssignment.Properties.Principal.Id == aadObjectId {
			isAdminRole := roleAssignment.Properties.RoleDefinition.Name == "EnvironmentAdmin"
			isMakerRole := roleAssignment.Properties.RoleDefinition.Name == "EnvironmentMaker"

			user.Id = roleAssignment.Properties.Principal.Id
			user.AadObjectId = roleAssignment.Properties.Principal.Id
			user.DomainName = roleAssignment.Properties.Principal.DisplayName
			if isAdminRole {
				user.SecurityRoles = append(user.SecurityRoles, EnvironmentUserSecurityRoleDto{
					Name:   roleAssignment.Name,
					RoleId: RoleEnvironmentAdmin,
				})
			} else if isMakerRole {
				user.SecurityRoles = append(user.SecurityRoles, EnvironmentUserSecurityRoleDto{
					Name:   roleAssignment.Name,
					RoleId: RoleEnvironmentMaker,
				})
			}
		}
	}

	return user, nil
}

// RemoveEnvironmentUserSecurityRoles removes environment role assignments from
// a user and reads the user's remaining role assignments back.
func (client *Client) RemoveEnvironmentUserSecurityRoles(ctx context.Context, environmentId, aadObjectId string, securityRoles []string, savedRoles []EnvironmentUserSecurityRoleDto) (*EnvironmentUserDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/modifyRoleAssignments", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	userRead, err := client.GetEnvironmentUserByAadObjectId(ctx, environmentId, aadObjectId)
	if err != nil {
		return nil, err
	}
	userDisplayName := userRead.DomainName

	remove := EnvironmentUserRequestDto{
		Remove: []RoleDefinitionDto{},
	}

	for _, role := range securityRoles {
		savedRoleData := helpers.ArrayFind(savedRoles, func(roleDto EnvironmentUserSecurityRoleDto) bool {
			return roleDto.RoleId == role
		})

		remove.Remove = append(remove.Remove, RoleDefinitionDto{
			Id: fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/roleAssignments/%s", environmentId, savedRoleData.Name),
		})
	}

	resp, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, remove, []int{http.StatusOK, http.StatusInternalServerError}, nil)
	if err != nil {
		return nil, err
	}

	if resp.HttpResponse.StatusCode == http.StatusInternalServerError {
		tflog.Debug(ctx, fmt.Sprintf("Error Status Code: %d", resp.HttpResponse.StatusCode))
		tflog.Debug(ctx, fmt.Sprintf("Error removing security roles: %s", resp.BodyAsBytes))
	}

	userRead, err = client.GetEnvironmentUserByAadObjectId(ctx, environmentId, aadObjectId)
	if err != nil {
		return nil, err
	}

	user := EnvironmentUserDto{
		Id:            aadObjectId,
		AadObjectId:   aadObjectId,
		DomainName:    userDisplayName,
		SecurityRoles: userRead.SecurityRoles,
	}
	return &user, nil
}

// AddEnvironmentUserSecurityRoles adds environment role assignments to a user.
func (client *Client) AddEnvironmentUserSecurityRoles(ctx context.Context, environmentId, aadObjectId string, securityRoles []string) (*EnvironmentUserDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/modifyRoleAssignments", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_2021_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	add := EnvironmentUserRequestDto{
		Add: []AddItemRequestDto{},
	}

	for _, role := range securityRoles {
		role = strings.ToLower(strings.ReplaceAll(role, " ", ""))
		add.Add = append(add.Add, AddItemRequestDto{
			Properties: PropertiesDto{
				RoleDefinition: RoleDefinitionDto{
					Id: fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/roleDefinitions/%s", environmentId, role),
				},
				Principal: PrincipalDto{
					Id:   aadObjectId,
					Type: "User",
				},
			},
		})
	}

	respObj := EnvironmentUserResponseDto{}
	_, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, add, []int{http.StatusOK}, &respObj)
	if err != nil {
		return nil, err
	}

	displayName := respObj.Add[0].RoleAssignment.Properties.Principal.DisplayName
	if displayName == "" {
		displayName = client.pollEnvironmentUserDisplayName(ctx, environmentId, aadObjectId)
	}

	user := EnvironmentUserDto{
		Id:            aadObjectId,
		AadObjectId:   aadObjectId,
		DomainName:    displayName,
		SecurityRoles: []EnvironmentUserSecurityRoleDto{},
	}

	for _, roleAssignment := range respObj.Add {
		user.SecurityRoles = append(user.SecurityRoles, EnvironmentUserSecurityRoleDto{
			RoleId: roleAssignment.RoleAssignment.Properties.RoleDefinition.Name,
			Name:   roleAssignment.RoleAssignment.Name,
		})
	}
	return &user, nil
}

// pollEnvironmentUserDisplayName re-reads the role assignments because a freshly created Entra principal
// is not resolvable right away and comes back without a display name. Returns an empty string when the
// display name never shows up, as it is not essential to managing the role assignment.
func (client *Client) pollEnvironmentUserDisplayName(ctx context.Context, environmentId, aadObjectId string) string {
	for attempt := 0; attempt < environmentUserDisplayNameRetryCount; attempt++ {
		if err := client.Api.SleepWithContext(ctx, api.DefaultRetryAfter()); err != nil {
			return ""
		}

		user, err := client.GetEnvironmentUserByAadObjectId(ctx, environmentId, aadObjectId)
		if err != nil {
			tflog.Debug(ctx, fmt.Sprintf("Error reading display name of environment user '%s': %s", aadObjectId, err.Error()))
			return ""
		}
		if user.DomainName != "" {
			return user.DomainName
		}
		tflog.Debug(ctx, fmt.Sprintf("Display name of environment user '%s' not available yet (attempt %d)", aadObjectId, attempt+1))
	}
	return ""
}

// CreateEnvironmentUser adds a user to an environment by assigning it
// environment security roles.
func (client *Client) CreateEnvironmentUser(ctx context.Context, environmentId, aadObjectId string, securityRoles []string) (*EnvironmentUserDto, error) {
	// adding security roles and creating user is the same API call.
	return client.AddEnvironmentUserSecurityRoles(ctx, environmentId, aadObjectId, securityRoles)
}

// AddUserToEnvironment forces the sync of an Entra user into an environment's
// Dataverse instance, retrying while the user's license assignment is still
// propagating.
func (client *Client) AddUserToEnvironment(ctx context.Context, environmentId, aadObjectId string) error {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s/addUser", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	userToCreate := map[string]any{
		"objectId": aadObjectId,
	}

	// 9 minutes of retries.
	retryCount := 6 * 9
	var err error

	for retryCount > 0 {
		_, err = client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, userToCreate, []int{http.StatusOK}, nil)
		// the license assignment in Entra is async, so we need to wait for that to happen if a user is created in the same terraform run.
		if err == nil || !strings.Contains(err.Error(), "userNotLicensed") {
			break
		}
		tflog.Debug(ctx, fmt.Sprintf("Error creating user: %s", err.Error()))
		err = client.Api.SleepWithContext(ctx, 10*time.Second)
		if err != nil {
			return err
		}

		retryCount--
	}
	if err != nil {
		if retryCount == 0 {
			return fmt.Errorf("failed to create Dataverse user after maximum retries: %w", err)
		}
		return err
	}

	return nil
}
