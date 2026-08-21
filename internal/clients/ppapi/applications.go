// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// GetTenantApplications lists the Dynamics 365 application packages available
// in the tenant.
func (client *Client) GetTenantApplications(ctx context.Context) ([]TenantApplicationDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   "/appmanagement/applicationPackages",
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.APPLICATION_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	application := TenantApplicationArrayDto{}

	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &application)
	if err != nil {
		return nil, err
	}

	return application.Value, nil
}

// GetApplicationsByEnvironmentId lists the application packages of an
// environment.
func (client *Client) GetApplicationsByEnvironmentId(ctx context.Context, environmentId string) ([]EnvironmentApplicationDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/appmanagement/environments/%s/applicationPackages", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.APPLICATION_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	application := EnvironmentApplicationArrayDto{}

	_, err := client.Api.Execute(ctx, client.scopes(), "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &application)
	if err != nil {
		return nil, err
	}

	return application.Value, nil
}

// InstallApplicationInEnvironment installs an application package into an
// environment and polls the install operation until it finishes.
func (client *Client) InstallApplicationInEnvironment(ctx context.Context, environmentId string, uniqueName string) (string, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.host(),
		Path:   fmt.Sprintf("/appmanagement/environments/%s/applicationPackages/%s/install", environmentId, uniqueName),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, constants.APPLICATION_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	response, err := client.Api.Execute(ctx, client.scopes(), "POST", apiUrl.String(), nil, nil, []int{http.StatusAccepted}, nil)
	if err != nil {
		return "", err
	}

	applicationId := ""
	if response.HttpResponse.StatusCode == http.StatusAccepted {
		operationLocationHeader := response.GetHeader(constants.HEADER_OPERATION_LOCATION)
		if operationLocationHeader == "" {
			tflog.Error(ctx, "Missing operation location header in response")
			return "", errors.New("missing operation location header in response")
		}
		tflog.Debug(ctx, "Opeartion Location Header: "+operationLocationHeader)

		_, err = url.Parse(operationLocationHeader)
		if err != nil {
			tflog.Error(ctx, "Error parsing location header: "+err.Error())
			return "", err
		}

		for {
			lifecycleResponse := EnvironmentApplicationLifecycleDto{}
			response, err := client.Api.Execute(ctx, client.scopes(), "GET", operationLocationHeader, nil, nil, []int{http.StatusOK, http.StatusConflict}, &lifecycleResponse)
			if err != nil {
				return "", err
			}

			if response.HttpResponse.StatusCode == http.StatusConflict {
				tflog.Debug(ctx, "Lifecycle Operation HTTP Status: '"+response.HttpResponse.Status+"'")
				continue
			}

			if lifecycleResponse.Status == "Succeeded" {
				parts := strings.Split(lifecycleResponse.CreatedDateTime, "/")
				if len(parts) == 0 {
					return "", errors.New("can't parse application id from response " + lifecycleResponse.CreatedDateTime)
				}
				applicationId = parts[len(parts)-1]
				tflog.Debug(ctx, "Created Application Id: "+applicationId)
				break
			} else if lifecycleResponse.Status == "Failed" {
				return "", errors.New("application installation failed. status message: " + lifecycleResponse.Error.Message)
			}
		}
	} else if response.HttpResponse.StatusCode == http.StatusCreated {
		appCreatedResponse := EnvironmentApplicationLifecycleCreatedDto{}
		err = response.MarshallTo(&appCreatedResponse)
		if err != nil {
			return "", err
		}
		if appCreatedResponse.Properties.ProvisioningState != "Succeeded" {
			return "", errors.New("application installation failed. provisioning state: " + appCreatedResponse.Properties.ProvisioningState)
		}
		applicationId = appCreatedResponse.Name
	}

	return applicationId, nil
}
