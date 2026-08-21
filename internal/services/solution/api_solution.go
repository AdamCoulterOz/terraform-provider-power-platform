// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package solution

import (
	"context"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

func NewSolutionClient(apiClient *api.Client) Client {
	return Client{
		Client:    bapi.NewClient(apiClient),
		Dataverse: dataverse.NewClient(apiClient),
	}
}

// Client is the solution service client. BAPI environment lookups (including
// GetEnvironmentHostById) are promoted from the embedded bapi.Client; the
// Dataverse Web API operations are delegated to the dataverse client after
// resolving the environment host.
type Client struct {
	*bapi.Client
	Dataverse *dataverse.Client
}

func (client *Client) DataverseExists(ctx context.Context, environmentId string) (bool, error) {
	env, err := client.GetEnvironment(ctx, environmentId)
	if err != nil {
		return false, err
	}
	return env.Properties.LinkedEnvironmentMetadata != nil && env.Properties.LinkedEnvironmentMetadata.InstanceURL != "", nil
}

func (client *Client) GetSolutionUniqueName(ctx context.Context, environmentId, name string) (*SolutionDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	solution, err := client.Dataverse.GetSolutionUniqueName(ctx, environmentHost, name)
	if err != nil {
		return nil, err
	}

	solution.EnvironmentId = environmentId

	return solution, nil
}

func (client *Client) GetSolutionById(ctx context.Context, environmentId, solutionId string) (*SolutionDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	solution, err := client.Dataverse.GetSolutionById(ctx, environmentHost, solutionId)
	if err != nil {
		return nil, err
	}

	solution.EnvironmentId = environmentId

	return solution, nil
}

func (client *Client) GetSolutions(ctx context.Context, environmentId string) ([]SolutionDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	solutions, err := client.Dataverse.GetSolutions(ctx, environmentHost)
	if err != nil {
		return nil, err
	}

	for inx := range solutions {
		solutions[inx].EnvironmentId = environmentId
	}

	return solutions, nil
}

func (client *Client) GetInstalledSolutions(ctx context.Context, environmentId string) ([]SolutionDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	solutions, err := client.Dataverse.GetInstalledSolutions(ctx, environmentHost)
	if err != nil {
		return nil, err
	}

	for inx := range solutions {
		solutions[inx].EnvironmentId = environmentId
	}

	return solutions, nil
}

func (client *Client) CreateSolution(ctx context.Context, environmentId string, content []byte, settings []byte) (*SolutionDto, error) {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	solution, err := client.Dataverse.CreateSolution(ctx, environmentHost, content, settings)
	if err != nil {
		return nil, err
	}

	solution.EnvironmentId = environmentId

	return solution, nil
}

func (client *Client) DeleteSolution(ctx context.Context, environmentId, solutionId string) error {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}
	return client.Dataverse.DeleteSolution(ctx, environmentHost, solutionId)
}

func (client *Client) GetTableData(ctx context.Context, environmentId, tableName, odataQuery string, responseObj any) error {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}
	return client.Dataverse.GetTableData(ctx, environmentHost, tableName, odataQuery, responseObj)
}
