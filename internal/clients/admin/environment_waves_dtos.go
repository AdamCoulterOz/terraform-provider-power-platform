// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package admin

type FeatureDto struct {
	FeatureName      string `json:"featureName"`
	DisplayName      string `json:"displayName"`
	CanBeReset       bool   `json:"canBeReset"`
	Enabled          bool   `json:"enabled"`
	IsAllowed        bool   `json:"isAllowed"`
	NotBefore        string `json:"notBefore"`
	NotAfter         string `json:"notAfter"`
	MinVersion       string `json:"minVersion"`
	MaxVersion       string `json:"maxVersion"`
	State            string `json:"state"`
	AppsUpgradeState string `json:"appsUpgradeState"`
}

type FeaturesArrayDto struct {
	Values []FeatureDto `json:"values"`
}

type OrganizationDto struct {
	TenantId         string `json:"tenantId"`
	Name             string `json:"name"`
	Id               string `json:"id"`
	CrmGeo           string `json:"crmGeo"`
	RelationType     string `json:"relationType"`
	OrganizationType int    `json:"organizationType"`
	Url              string `json:"url"`
}

type OrganizationsArrayDto []OrganizationDto
