// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

// TenantCapacityDetailsDto represents the main response model for tenant capacity details.
type TenantCapacityDetailsDto struct {
	CapacitySummary      CapacitySummaryDto                     `json:"capacitySummary"`
	LegacyModelCapacity  *LegacyCapacityModelDto                `json:"legacyModelCapacity,omitempty"`
	LicenseModelType     string                                 `json:"licenseModelType"`
	TemporaryLicenseInfo TemporaryLicenseInfoDto                `json:"temporaryLicenseInfo"`
	TenantCapacities     []TenantCapacityAndConsumptionModelDto `json:"tenantCapacities"`
	TenantId             string                                 `json:"tenantId"`
}

// CapacitySummaryDto represents the capacity summary information.
type CapacitySummaryDto struct {
	FinOpsStatus            string `json:"finOpsStatus"`
	FinOpsStatusMessage     string `json:"finOpsStatusMessage"`
	FinOpsStatusMessageCode string `json:"finOpsStatusMessageCode"`
	Status                  string `json:"status"`
	StatusMessage           string `json:"statusMessage"`
	StatusMessageCode       string `json:"statusMessageCode"`
}

// LegacyCapacityModelDto represents legacy capacity model information.
type LegacyCapacityModelDto struct {
	CapacityUnits    string  `json:"capacityUnits"`
	TotalCapacity    float64 `json:"totalCapacity"`
	TotalConsumption float64 `json:"totalConsumption"`
}

// TemporaryLicenseInfoDto represents temporary license information.
type TemporaryLicenseInfoDto struct {
	HasTemporaryLicense        bool   `json:"hasTemporaryLicense"`
	TemporaryLicenseExpiryDate string `json:"temporaryLicenseExpiryDate,omitempty"`
}

// TenantCapacityAndConsumptionModelDto represents tenant capacity and consumption model.
type TenantCapacityAndConsumptionModelDto struct {
	CapacityEntitlements []TenantCapacityEntitlementModelDto `json:"capacityEntitlements"`
	CapacityType         string                              `json:"capacityType"`
	CapacityUnits        string                              `json:"capacityUnits"`
	Consumption          ConsumptionModelDto                 `json:"consumption"`
	MaxCapacity          float32                             `json:"maxCapacity"`
	OverflowCapacity     []OverflowCapacityModelDto          `json:"overflowCapacity"`
	Status               string                              `json:"status"`
	TotalCapacity        float32                             `json:"totalCapacity"`
}

// TenantCapacityEntitlementModelDto represents tenant capacity entitlement model.
type TenantCapacityEntitlementModelDto struct {
	CapacitySubType      string                   `json:"capacitySubType"`
	CapacityType         string                   `json:"capacityType"`
	Licenses             []LicenseDetailsModelDto `json:"licenses"`
	MaxNextLifecycleDate string                   `json:"maxNextLifecycleDate,omitempty"`
	TotalCapacity        float32                  `json:"totalCapacity"`
}

// LicenseDetailsModelDto represents license details model.
type LicenseDetailsModelDto struct {
	CapabilityStatus           string             `json:"capabilityStatus"`
	DisplayName                string             `json:"displayName"`
	EntitlementCode            string             `json:"entitlementCode"`
	IsTemporaryLicense         bool               `json:"isTemporaryLicense"`
	NextLifecycleDate          string             `json:"nextLifecycleDate,omitempty"`
	Paid                       LicenseQuantityDto `json:"paid"`
	ServicePlanId              string             `json:"servicePlanId"`
	SkuId                      string             `json:"skuId"`
	TemporaryLicenseExpiryDate string             `json:"temporaryLicenseExpiryDate,omitempty"`
	TotalCapacity              float32            `json:"totalCapacity"`
	Trial                      LicenseQuantityDto `json:"trial"`
}

// LicenseQuantityDto represents license quantity information.
type LicenseQuantityDto struct {
	Enabled   int32 `json:"enabled"`
	Suspended int32 `json:"suspended"`
	Warning   int32 `json:"warning"`
}

// ConsumptionModelDto represents consumption model information.
type ConsumptionModelDto struct {
	Actual          float32 `json:"actual"`
	ActualUpdatedOn string  `json:"actualUpdatedOn,omitempty"`
	Rated           float32 `json:"rated"`
	RatedUpdatedOn  string  `json:"ratedUpdatedOn,omitempty"`
}

// OverflowCapacityModelDto represents overflow capacity model.
type OverflowCapacityModelDto struct {
	CapacityType string  `json:"capacityType"`
	Value        float32 `json:"value"`
}
