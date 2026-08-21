// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment

import (
	"context"
	"fmt"
	"net/url"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/bapi"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

func NewEnvironmentClient(apiClient *api.Client) Client {
	return Client{
		Client:    bapi.NewClient(apiClient),
		dataverse: dataverse.NewClient(apiClient),
	}
}

// Client is the environment service client. BAPI environment operations are
// promoted from the embedded bapi.Client; only workflows that span another
// API boundary live here.
type Client struct {
	*bapi.Client
	dataverse *dataverse.Client
}

// GetDefaultCurrencyForEnvironment reads the default transaction currency of
// an environment from Dataverse.
func (client *Client) GetDefaultCurrencyForEnvironment(ctx context.Context, environmentId string) (*dataverse.TransactionCurrencyDto, error) {
	orgSettings := dataverse.OrganizationSettingsArrayDto{}
	err := client.getTableData(ctx, environmentId, "organizations", "", &orgSettings)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Add("$filter", "transactioncurrencyid eq "+orgSettings.Value[0].BaseCurrencyId)

	currencies := dataverse.TransactionCurrencyArrayDto{}
	err = client.getTableData(ctx, environmentId, "transactioncurrencies", values.Encode(), &currencies)
	if err != nil {
		return nil, err
	}
	if len(currencies.Value) == 0 {
		return nil, fmt.Errorf("no default currency found for environment %s", environmentId)
	}
	return &currencies.Value[0], nil
}

// getTableData resolves the Dataverse host of the environment and reads rows
// of a table through the dataverse client.
func (client *Client) getTableData(ctx context.Context, environmentId, tableName, odataQuery string, responseObj any) error {
	environmentHost, err := client.GetEnvironmentHostById(ctx, environmentId)
	if err != nil {
		return err
	}
	return client.dataverse.GetTableData(ctx, environmentHost, tableName, odataQuery, responseObj)
}
