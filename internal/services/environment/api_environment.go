// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment

import (
	"context"
	"fmt"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/services/solution"
)

func NewEnvironmentClient(apiClient *api.Client) Client {
	return Client{
		Client:         bapi.NewClient(apiClient),
		solutionClient: solution.NewSolutionClient(apiClient),
	}
}

// Client is the environment service client. BAPI environment operations are
// promoted from the embedded bapi.Client; only workflows that span another
// API boundary live here.
type Client struct {
	*bapi.Client
	solutionClient solution.Client
}

// GetDefaultCurrencyForEnvironment reads the default transaction currency of
// an environment from Dataverse.
func (client *Client) GetDefaultCurrencyForEnvironment(ctx context.Context, environmentId string) (*TransactionCurrencyDto, error) {
	orgSettings := organizationSettingsArrayDto{}
	err := client.solutionClient.GetTableData(ctx, environmentId, "organizations", "", &orgSettings)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Add("$filter", "transactioncurrencyid eq "+orgSettings.Value[0].BaseCurrencyId)

	currencies := transactionCurrencyArrayDto{}
	err = client.solutionClient.GetTableData(ctx, environmentId, "transactioncurrencies", values.Encode(), &currencies)
	if err != nil {
		return nil, err
	}
	if len(currencies.Value) == 0 {
		return nil, fmt.Errorf("no default currency found for environment %s", environmentId)
	}
	return &currencies.Value[0], nil
}
