// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FabricLinkResourceModel is the Terraform state model for powerplatform_fabric_link.
//
// A "Link to Fabric" is a Dataverse Azure Synapse Link profile (synapselinkprofile,
// ProfileType = SynapseLink) whose target is a Fabric workspace / OneLake lakehouse.
// It is an ENVIRONMENT SINGLETON: one Dataverse environment links to one Fabric
// workspace, which is why callers should treat fabric_workspace_id as 1:1 with
// environment_id. The created mirror lakehouse + SQL endpoint are surfaced as computed
// attributes so downstream Fabric items can reference them by stable id.
type FabricLinkResourceModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`

	Id            types.String `tfsdk:"id"`             // synapselinkprofileid
	EnvironmentId types.String `tfsdk:"environment_id"` // Dataverse environment id
	Name          types.String `tfsdk:"name"`           // profile display name
	UniqueName    types.String `tfsdk:"unique_name"`    // profile unique name (computed if unset)

	// Target Fabric workspace for the mirror (the environment-singleton target).
	FabricWorkspaceId types.String `tfsdk:"fabric_workspace_id"`

	// Dataverse->OneLake connection id the link uses. Until the service-principal connection
	// upsert is wired (DESIGN.md), supply a pre-created connection id here.
	ConnectionId types.String `tfsdk:"connection_id"`

	// Dataverse tables to include in the link. Each must have change tracking enabled.
	TableNames types.Set `tfsdk:"table_names"`

	// Computed read-back identifiers for the Dataverse-managed Fabric artifacts.
	MirrorLakehouseId types.String `tfsdk:"mirror_lakehouse_id"`
	MirrorWorkspaceId types.String `tfsdk:"mirror_workspace_id"`
	ProfileState      types.String `tfsdk:"profile_state"`

	// datalakefolder id the unlink DELETE targets (resolved after create; used by Delete).
	DatalakeFolderId types.String `tfsdk:"datalake_folder_id"`
}
