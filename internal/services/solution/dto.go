// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package solution

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

// SolutionDto is the canonical Dataverse solution wire shape, which moved to
// the dataverse client library. The alias keeps existing references
// (solution.SolutionDto) in this and dependent services compiling unchanged.
type SolutionDto = dataverse.SolutionDto
