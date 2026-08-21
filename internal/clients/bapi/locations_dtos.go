// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

type LocationArrayDto struct {
	Value []LocationDto `json:"value"`
}

type LocationDto struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"`
	Name       string                `json:"name"`
	Properties LocationPropertiesDto `json:"properties"`
}

type LocationPropertiesDto struct {
	DisplayName                            string   `json:"displayName"`
	Code                                   string   `json:"code"`
	IsDefault                              bool     `json:"isDefault"`
	IsDisabled                             bool     `json:"isDisabled"`
	CanProvisionDatabase                   bool     `json:"canProvisionDatabase"`
	CanProvisionCustomerEngagementDatabase bool     `json:"canProvisionCustomerEngagementDatabase"`
	AzureRegions                           []string `json:"azureRegions"`
}

type CurrenciesArrayDto struct {
	Value []CurrencyDto `json:"value"`
}

type CurrencyDto struct {
	Name       string                `json:"name"`
	ID         string                `json:"id"`
	Type       string                `json:"type"`
	Properties CurrencyPropertiesDto `json:"properties"`
}

type CurrencyPropertiesDto struct {
	Code            string `json:"code"`
	Symbol          string `json:"symbol"`
	IsTenantDefault bool   `json:"isTenantDefault"`
}

type LanguagesArrayDto struct {
	Value []LanguageDto `json:"value"`
}

type LanguageDto struct {
	Name       string                `json:"name"`
	ID         string                `json:"id"`
	Type       string                `json:"type"`
	Properties LanguagePropertiesDto `json:"properties"`
}

type LanguagePropertiesDto struct {
	LocaleID        int64  `json:"localeId"`
	LocalizedName   string `json:"localizedName"`
	DisplayName     string `json:"displayName"`
	IsTenantDefault bool   `json:"isTenantDefault"`
}
