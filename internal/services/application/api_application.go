// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/ppapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

func newApplicationClient(apiClient *api.Client) client {
	return client{
		Client:        bapi.NewClient(apiClient),
		Dataverse:     dataverse.NewClient(apiClient),
		PowerPlatform: ppapi.NewClient(apiClient),
	}
}

// client orchestrates the application service across three API boundaries:
// BAPI for environment lookups and the addAppUser admin operation (embedded so
// its methods promote), Dataverse for the per-environment application-user and
// security-role operations, and the Power Platform API for the appmanagement
// application-package operations.
type client struct {
	*bapi.Client
	Dataverse     *dataverse.Client
	PowerPlatform *ppapi.Client
}

func (client *client) DataverseExists(ctx context.Context, environmentId string) (bool, error) {
	env, err := client.GetEnvironment(ctx, environmentId)
	if err != nil {
		return false, err
	}
	return env.Properties.LinkedEnvironmentMetadata != nil && env.Properties.LinkedEnvironmentMetadata.InstanceURL != "", nil
}

func (client *client) GetRootBusinessUnitId(ctx context.Context, environmentId string) (string, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return "", err
	}

	return client.Dataverse.GetRootBusinessUnitId(ctx, environmentHost, environmentId)
}

func (client *client) ApplicationUserExists(ctx context.Context, environmentId string, applicationId string) (bool, error) {
	// Reuse GetApplicationUserSystemId to check if application user exists
	_, err := client.GetApplicationUserSystemId(ctx, environmentId, applicationId)
	if err != nil {
		if errors.Is(err, customerrors.ErrObjectNotFound) {
			return false, nil
		}
		// For other errors (like forbidden access), propagate the error
		return false, err
	}

	// If no error, the application user exists
	return true, nil
}

func (client *client) GetApplicationUser(ctx context.Context, environmentId string, applicationId string) (*dataverse.ApplicationUserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.Dataverse.GetApplicationUser(ctx, environmentHost, environmentId, applicationId)
}

func (client *client) getApplicationUsersByApplicationId(ctx context.Context, environmentId string, applicationId string) ([]dataverse.ApplicationUserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.Dataverse.GetApplicationUsersByApplicationId(ctx, environmentHost, environmentId, applicationId)
}

func (client *client) GetApplicationUserSystemId(ctx context.Context, environmentId string, applicationId string) (string, error) {
	user, err := client.GetApplicationUser(ctx, environmentId, applicationId)
	if err != nil {
		return "", err
	}
	return user.SystemUserId, nil
}

func (client *client) CreateScopedApplicationUser(ctx context.Context, environmentId, applicationId, businessUnitId string) (*dataverse.ApplicationUserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	systemUserId, err := client.Dataverse.CreateApplicationUser(ctx, environmentHost, applicationId, businessUnitId)
	if err != nil {
		if purgeErr := client.purgeDeletedApplicationUsersByApplicationId(ctx, environmentId, applicationId, err); purgeErr != nil {
			return nil, purgeErr
		}

		systemUserId, err = client.Dataverse.CreateApplicationUser(ctx, environmentHost, applicationId, businessUnitId)
		if err != nil {
			return nil, err
		}
	}

	return client.GetApplicationUserBySystemUserId(ctx, environmentId, systemUserId)
}

func (client *client) purgeDeletedApplicationUsersByApplicationId(ctx context.Context, environmentId, applicationId string, createErr error) error {
	users, err := client.getApplicationUsersByApplicationId(ctx, environmentId, applicationId)
	if err != nil {
		if errors.Is(err, customerrors.ErrObjectNotFound) {
			return createErr
		}

		return fmt.Errorf("%w\n\nAn additional lookup for deleted application users failed: %s", createErr, err.Error())
	}

	purgedAny := false
	for _, user := range users {
		if user.DeletedState == 0 {
			continue
		}

		if err := client.PermanentlyDeleteSystemUser(ctx, environmentId, user.SystemUserId); err != nil {
			return fmt.Errorf("%w\n\nA conflicting deleted application user '%s' was found but could not be permanently deleted: %s", createErr, user.SystemUserId, err.Error())
		}

		purgedAny = true
	}

	if !purgedAny {
		return createErr
	}

	return nil
}

func (client *client) SetApplicationUserDisabledState(ctx context.Context, environmentId, systemUserId, applicationId string, disabled bool) (*dataverse.ApplicationUserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	if err := client.Dataverse.SetApplicationUserDisabledState(ctx, environmentHost, systemUserId, applicationId, disabled); err != nil {
		return nil, err
	}

	return client.GetApplicationUserBySystemUserId(ctx, environmentId, systemUserId)
}

func (client *client) GetApplicationUserBySystemUserId(ctx context.Context, environmentId, systemUserId string) (*dataverse.ApplicationUserDto, error) {
	return client.GetPrincipalBySystemUserId(ctx, environmentId, systemUserId)
}

func (client *client) GetPrincipalBySystemUserId(ctx context.Context, environmentId, systemUserId string) (*dataverse.ApplicationUserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.Dataverse.GetPrincipalBySystemUserId(ctx, environmentHost, systemUserId)
}

func (client *client) GetDataverseSecurityRoles(ctx context.Context, environmentId, businessUnitId string) ([]dataverse.ApplicationSecurityRoleDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.Dataverse.GetDataverseSecurityRoles(ctx, environmentHost, businessUnitId)
}

func (client *client) ResolveSecurityRoleNames(ctx context.Context, environmentId, businessUnitId string, roleNames []string) ([]dataverse.ApplicationSecurityRoleDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.Dataverse.ResolveSecurityRoleNames(ctx, environmentHost, businessUnitId, roleNames)
}

func (client *client) AddPrincipalSecurityRoles(ctx context.Context, environmentId, systemUserId string, roleIds []string) (*dataverse.ApplicationUserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	if err := client.Dataverse.AddPrincipalSecurityRoles(ctx, environmentHost, systemUserId, roleIds); err != nil {
		return nil, err
	}

	return client.GetPrincipalBySystemUserId(ctx, environmentId, systemUserId)
}

func (client *client) RemovePrincipalSecurityRoles(ctx context.Context, environmentId, systemUserId string, roleIds []string) (*dataverse.ApplicationUserDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	if err := client.Dataverse.RemovePrincipalSecurityRoles(ctx, environmentHost, systemUserId, roleIds); err != nil {
		return nil, err
	}

	return client.GetPrincipalBySystemUserId(ctx, environmentId, systemUserId)
}

func (client *client) DeactivateSystemUser(ctx context.Context, environmentId string, systemUserId string) error {
	// Get the application user to find the application ID
	appUser, err := client.getApplicationUserBySystemId(ctx, environmentId, systemUserId)
	if err != nil {
		return err
	}

	if _, err = client.SetApplicationUserDisabledState(ctx, environmentId, systemUserId, appUser.ApplicationId, true); err != nil {
		return err
	}

	return nil
}

func (client *client) DeleteSystemUser(ctx context.Context, environmentId string, systemUserId string) error {
	return client.executeSystemUserDelete(ctx, environmentId, systemUserId, []int{http.StatusNoContent, http.StatusOK})
}

func (client *client) PermanentlyDeleteSystemUser(ctx context.Context, environmentId string, systemUserId string) error {
	if err := client.executeSystemUserDelete(ctx, environmentId, systemUserId, []int{http.StatusNoContent, http.StatusOK, http.StatusNotFound}); err != nil {
		return err
	}

	return client.executeSystemUserDelete(ctx, environmentId, systemUserId, []int{http.StatusNoContent, http.StatusOK, http.StatusNotFound})
}

func (client *client) executeSystemUserDelete(ctx context.Context, environmentId string, systemUserId string, expectedStatusCodes []int) error {
	// Get the environment host
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}

	return client.Dataverse.DeleteSystemUser(ctx, environmentHost, systemUserId, expectedStatusCodes)
}

func (client *client) GetTenantApplications(ctx context.Context) ([]ppapi.TenantApplicationDto, error) {
	return client.PowerPlatform.GetTenantApplications(ctx)
}

func (client *client) GetApplicationsByEnvironmentId(ctx context.Context, environmentId string) ([]ppapi.EnvironmentApplicationDto, error) {
	return client.PowerPlatform.GetApplicationsByEnvironmentId(ctx, environmentId)
}

func (client *client) InstallApplicationInEnvironment(ctx context.Context, environmentId string, uniqueName string) (string, error) {
	return client.PowerPlatform.InstallApplicationInEnvironment(ctx, environmentId, uniqueName)
}

func (client *client) getApplicationUserBySystemId(ctx context.Context, environmentId string, systemUserId string) (*dataverse.ApplicationUserDto, error) {
	return client.GetPrincipalBySystemUserId(ctx, environmentId, systemUserId)
}
