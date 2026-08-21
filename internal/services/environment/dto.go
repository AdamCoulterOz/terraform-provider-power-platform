// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environment

import (
	"fmt"
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
