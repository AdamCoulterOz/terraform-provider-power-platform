// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment

import (
	"fmt"
	"time"
)

const (
	EnvironmentTypesDeveloper  = "Developer"
	EnvironmentTypesSandbox    = "Sandbox"
	EnvironmentTypesProduction = "Production"
	EnvironmentTypesTrial      = "Trial"
	EnvironmentTypesDefault    = "Default"

	CadenceTypesFrequent = "Frequent"
	CadenceTypesModerate = "Moderate"

	ReleaseCycleTypesStandard = "Standard"
	ReleaseCycleTypesEarly    = "Early"

	ReleaseCycleFirstReleasePublicDto = "FirstRelease"
	ReleaseCycleFirstReleaseGovDto    = "GovFR"
)

var (
	EnvironmentTypes                     = []string{EnvironmentTypesDeveloper, EnvironmentTypesSandbox, EnvironmentTypesProduction, EnvironmentTypesTrial, EnvironmentTypesDefault}
	EnvironmentTypesDeveloperOnlyRegex   = fmt.Sprintf(`^(%s)$`, EnvironmentTypesDeveloper)
	EnvironmentTypesExceptDeveloperRegex = fmt.Sprintf(`^(%s|%s|%s|%s)$`, EnvironmentTypesSandbox, EnvironmentTypesProduction, EnvironmentTypesTrial, EnvironmentTypesDefault)
	CadenceTypes                         = []string{CadenceTypesFrequent, CadenceTypesModerate}
	ReleaseCycleTypes                    = []string{ReleaseCycleTypesStandard, ReleaseCycleTypesEarly}
	ReleaseCycleFirstReleaseOnlyRegex    = fmt.Sprintf(`^(%s|%s)$`, ReleaseCycleFirstReleasePublicDto, ReleaseCycleFirstReleaseGovDto)

	// EuDataBoundaryLocations are the environment locations that sit inside the EU Data Boundary,
	// which covers the European Union and the EFTA states.
	EuDataBoundaryLocations = []string{"europe", "france", "germany", "italy", "norway", "poland", "sweden", "switzerland"}
)

// The types below are Dataverse Web API wire shapes used by
// GetDefaultCurrencyForEnvironment; they move to the dataverse client library
// with the Dataverse extraction.

type organizationSettingsArrayDto struct {
	Value []organizationSettingsDto `json:"value"`
}

type organizationSettingsDto struct {
	ODataEtag      string    `json:"@odata.etag"`
	CreatedOn      time.Time `json:"createdon"`
	BaseCurrencyId string    `json:"_basecurrencyid_value"`
}

type TransactionCurrencyDto struct {
	OrganizationValue     string  `json:"_organizationid_value"`
	CurrencyName          string  `json:"currencyname"`
	CurrencySymbol        string  `json:"currencysymbol"`
	IsoCurrencyCode       string  `json:"isocurrencycode"`
	CreatedOn             string  `json:"createdon"`
	CurrencyPrecision     int     `json:"currencyprecision"`
	ExchangeRate          float32 `json:"exchangerate"`
	TransactionCurrencyId string  `json:"transactioncurrencyid"`
}

type transactionCurrencyArrayDto struct {
	Value []TransactionCurrencyDto `json:"value"`
}
