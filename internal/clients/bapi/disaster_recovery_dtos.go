// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

// DisasterRecoveryPatchDto is the request body for the environment PATCH that
// toggles cross-region disaster recovery.
type DisasterRecoveryPatchDto struct {
	Properties DisasterRecoveryPatchPropertiesDto `json:"properties"`
}

type DisasterRecoveryPatchPropertiesDto struct {
	States DisasterRecoveryPatchStatesDto `json:"states"`
}

type DisasterRecoveryPatchStatesDto struct {
	DisasterRecovery DisasterRecoveryStateDto `json:"disasterRecovery"`
}
