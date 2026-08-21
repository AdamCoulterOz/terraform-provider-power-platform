// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package environmentvariable

import (
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

// resolvedEnvironmentVariableDto pairs an environment variable definition with
// its current value record (nil when no value is set). It is a service-side
// composite, not a wire shape.
type resolvedEnvironmentVariableDto struct {
	Definition dataverse.EnvironmentVariableDefinitionDto
	Value      *dataverse.EnvironmentVariableValueDto
}
