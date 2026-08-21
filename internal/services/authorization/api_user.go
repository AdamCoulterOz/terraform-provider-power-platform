// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package authorization

import (
	"context"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

func newUserClient(apiClient *api.Client) client {
	return client{
		Client:          bapi.NewClient(apiClient),
		dataverseClient: dataverse.NewClient(apiClient),
	}
}

type client struct {
	*bapi.Client
	dataverseClient *dataverse.Client
}

func (client *client) EnvironmentHasDataverse(ctx context.Context, environmentId string) (bool, error) {
	env, err := client.GetEnvironment(ctx, environmentId)
	if err != nil {
		return false, err
	}
	return env.Properties.LinkedEnvironmentMetadata != nil, nil
}

func (client *client) DataverseExists(ctx context.Context, environmentId string) (bool, error) {
	env, err := client.GetEnvironment(ctx, environmentId)
	if err != nil {
		return false, err
	}
	return env.Properties.LinkedEnvironmentMetadata != nil && env.Properties.LinkedEnvironmentMetadata.InstanceURL != "", nil
}

func (client *client) GetDataverseUsers(ctx context.Context, environmentId string) ([]dataverse.UserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.GetUsers(ctx, environmentHost)
}

func (client *client) GetDataverseUserBySystemUserId(ctx context.Context, environmentId, systemUserId string) (*dataverse.UserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.GetUserBySystemUserId(ctx, environmentHost, systemUserId)
}

// GetEnvironmentUserByAadObjectId reads a user's BAPI environment role
// assignments as a Dataverse-shaped user record.
func (client *client) GetEnvironmentUserByAadObjectId(ctx context.Context, environmentId, aadObjectId string) (*dataverse.UserDto, error) {
	user, err := client.Client.GetEnvironmentUserByAadObjectId(ctx, environmentId, aadObjectId)
	if err != nil {
		return nil, err
	}
	return userFromEnvironmentUserDto(user), nil
}

func (client *client) GetDataverseUserByAadObjectId(ctx context.Context, environmentId, aadObjectId string) (*dataverse.UserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.GetUserByAadObjectId(ctx, environmentHost, aadObjectId)
}

func (client *client) RemoveEnvironmentUserSecurityRoles(ctx context.Context, environmentId, aadObjectId string, securityRoles []string, savedRoles []dataverse.SecurityRoleDto) (*dataverse.UserDto, error) {
	user, err := client.Client.RemoveEnvironmentUserSecurityRoles(ctx, environmentId, aadObjectId, securityRoles, environmentUserSecurityRolesFromDataverse(savedRoles))
	if err != nil {
		return nil, err
	}
	return userFromEnvironmentUserDto(user), nil
}

func (client *client) AddEnvironmentUserSecurityRoles(ctx context.Context, environmentId, aadObjectId string, securityRoles []string) (*dataverse.UserDto, error) {
	user, err := client.Client.AddEnvironmentUserSecurityRoles(ctx, environmentId, aadObjectId, securityRoles)
	if err != nil {
		return nil, err
	}
	return userFromEnvironmentUserDto(user), nil
}

func (client *client) CreateEnvironmentUser(ctx context.Context, environmentId, aadObjectId string, securityRoles []string) (*dataverse.UserDto, error) {
	user, err := client.Client.CreateEnvironmentUser(ctx, environmentId, aadObjectId, securityRoles)
	if err != nil {
		return nil, err
	}
	return userFromEnvironmentUserDto(user), nil
}

// CreateDataverseUser is a cross-boundary workflow: it forces the user into
// the environment through BAPI, then reads the created systemuser back from
// Dataverse.
func (client *client) CreateDataverseUser(ctx context.Context, environmentId, aadObjectId string) (*dataverse.UserDto, error) {
	if err := client.AddUserToEnvironment(ctx, environmentId, aadObjectId); err != nil {
		return nil, err
	}

	user, err := client.GetDataverseUserByAadObjectId(ctx, environmentId, aadObjectId)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (client *client) UpdateDataverseUser(ctx context.Context, environmentId, systemUserId string, userUpdate *dataverse.UserDto) (*dataverse.UserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.UpdateUser(ctx, environmentHost, systemUserId, userUpdate)
}

func (client *client) DeleteDataverseUser(ctx context.Context, environmentId, systemUserId string) error {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}

	return client.dataverseClient.DeleteUser(ctx, environmentHost, systemUserId)
}

func (client *client) RemoveDataverseSecurityRoles(ctx context.Context, environmentId, systemUserId string, securityRolesIds []string) (*dataverse.UserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.RemoveSecurityRoles(ctx, environmentHost, systemUserId, securityRolesIds)
}

func (client *client) AddDataverseSecurityRoles(ctx context.Context, environmentId, systemUserId string, securityRolesIds []string) (*dataverse.UserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.AddSecurityRoles(ctx, environmentHost, systemUserId, securityRolesIds)
}

func (client *client) GetDataverseSecurityRoles(ctx context.Context, environmentId, businessUnitId string) ([]dataverse.SecurityRoleDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.GetSecurityRoles(ctx, environmentHost, businessUnitId)
}

// userFromEnvironmentUserDto converts a BAPI environment user into the
// Dataverse-shaped user record the resource works with. The two types carry
// the same wire shape on either side of the client-package boundary.
func userFromEnvironmentUserDto(user *bapi.EnvironmentUserDto) *dataverse.UserDto {
	if user == nil {
		return nil
	}
	converted := dataverse.UserDto{
		Id:             user.Id,
		DomainName:     user.DomainName,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		AadObjectId:    user.AadObjectId,
		BusinessUnitId: user.BusinessUnitId,
	}
	if user.SecurityRoles != nil {
		converted.SecurityRoles = make([]dataverse.SecurityRoleDto, 0, len(user.SecurityRoles))
		for _, role := range user.SecurityRoles {
			converted.SecurityRoles = append(converted.SecurityRoles, dataverse.SecurityRoleDto{
				RoleId:         role.RoleId,
				Name:           role.Name,
				IsManaged:      role.IsManaged,
				BusinessUnitId: role.BusinessUnitId,
			})
		}
	}
	return &converted
}

func environmentUserSecurityRolesFromDataverse(roles []dataverse.SecurityRoleDto) []bapi.EnvironmentUserSecurityRoleDto {
	if roles == nil {
		return nil
	}
	converted := make([]bapi.EnvironmentUserSecurityRoleDto, 0, len(roles))
	for _, role := range roles {
		converted = append(converted, bapi.EnvironmentUserSecurityRoleDto{
			RoleId:         role.RoleId,
			Name:           role.Name,
			IsManaged:      role.IsManaged,
			BusinessUnitId: role.BusinessUnitId,
		})
	}
	return converted
}
