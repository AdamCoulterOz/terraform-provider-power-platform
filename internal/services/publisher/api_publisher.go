// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package publisher

import (
	"context"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

type client struct {
	*bapi.Client
	dataverseClient *dataverse.Client
}

func newPublisherClient(apiClient *api.Client) client {
	return client{
		Client:          bapi.NewClient(apiClient),
		dataverseClient: dataverse.NewClient(apiClient),
	}
}

func (client *client) CreatePublisher(ctx context.Context, environmentId string, model *ResourceModel) (*dataverse.PublisherDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.CreatePublisher(ctx, environmentHost, publisherBodyFromModel(model))
}

func (client *client) GetPublisherById(ctx context.Context, environmentId, publisherId string) (*dataverse.PublisherDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.GetPublisherById(ctx, environmentHost, publisherId)
}

func (client *client) GetPublishers(ctx context.Context, environmentId string) ([]dataverse.PublisherDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.GetPublishers(ctx, environmentHost)
}

func (client *client) GetPublisherByUniqueName(ctx context.Context, environmentId, uniqueName string) (*dataverse.PublisherDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.GetPublisherByUniqueName(ctx, environmentHost, uniqueName)
}

func (client *client) UpdatePublisher(ctx context.Context, environmentId, publisherId string, model *ResourceModel) (*dataverse.PublisherDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	return client.dataverseClient.UpdatePublisher(ctx, environmentHost, publisherId, publisherBodyFromModel(model))
}

func (client *client) DeletePublisher(ctx context.Context, environmentId, publisherId string) error {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}

	return client.dataverseClient.DeletePublisher(ctx, environmentHost, publisherId)
}
