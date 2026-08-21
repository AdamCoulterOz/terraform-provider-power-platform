// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package bapi

// LinkEnterprisePolicyDto is the request body for linking or unlinking an
// enterprise policy to/from an environment. It is distinct from
// EnterprisePolicyDto, which is the policy shape returned inside an
// environment read.
type LinkEnterprisePolicyDto struct {
	SystemId string `json:"systemId"`
}
