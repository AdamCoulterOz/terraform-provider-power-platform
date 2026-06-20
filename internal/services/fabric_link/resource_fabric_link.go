// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

var _ resource.Resource = &FabricLinkResource{}
var _ resource.ResourceWithImportState = &FabricLinkResource{}

type FabricLinkResource struct {
	helpers.TypeInfo
	FabricLinkClient client
}

func NewFabricLinkResource() resource.Resource {
	return &FabricLinkResource{
		TypeInfo: helpers.TypeInfo{
			TypeName: "fabric_link",
		},
	}
}

func (r *FabricLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	r.ProviderTypeName = req.ProviderTypeName
	ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
	defer exitContext()
	resp.TypeName = r.FullTypeName()
	tflog.Debug(ctx, fmt.Sprintf("METADATA: %s", resp.TypeName))
}

func (r *FabricLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
	defer exitContext()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Dataverse **Link to Microsoft Fabric** (an Azure Synapse Link profile, `synapselinkprofile`, targeting a Fabric workspace). A Dataverse environment links 1:1 to a Fabric workspace, so this is an environment singleton. The created mirror lakehouse + workspace ids are exposed as computed attributes for downstream Fabric items to reference by stable id. **Preview/scaffold:** the Fabric target payload is not yet wired — see DESIGN.md.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Delete: true, Read: true}),
			"id": schema.StringAttribute{
				MarkdownDescription: "Synapse Link profile id (`synapselinkprofileid`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Dataverse environment id to link to Fabric.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the Synapse Link / Fabric link profile.",
				Required:            true,
			},
			"unique_name": schema.StringAttribute{
				MarkdownDescription: "Unique name of the profile. Computed from the service when not supplied.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"fabric_workspace_id": schema.StringAttribute{
				MarkdownDescription: "Target Fabric workspace id that hosts the mirror lakehouse (the environment-singleton target).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"connection_id": schema.StringAttribute{
				MarkdownDescription: "DatasourceId of the Fabric->Dataverse connection (a CommonDataService cloud connection authenticated by the workspace identity). This is what athena uses as its ConnectionId. See DESIGN.md for the composition (fabric_workspace identity + Synapse Link Service Access role + fabric_connection).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"table_names": schema.SetAttribute{
				MarkdownDescription: "Logical names of the Dataverse tables to include in the link. Each must have change tracking enabled.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"mirror_lakehouse_id": schema.StringAttribute{
				MarkdownDescription: "Id of the Dataverse-managed Fabric mirror lakehouse created by the link (stable identifier; never reference the CDS2/CDS3 display name).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"mirror_workspace_id": schema.StringAttribute{
				MarkdownDescription: "Id of the workspace holding the mirror lakehouse (equals fabric_workspace_id once provisioned).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"profile_state": schema.StringAttribute{
				MarkdownDescription: "Synapse Link profile state (Inactive, Active, Error, ...).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"datalake_folder_id": schema.StringAttribute{
				MarkdownDescription: "Dataverse datalakefolder id that the unlink targets (resolved after create).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *FabricLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
	defer exitContext()
	if req.ProviderData == nil {
		return
	}
	providerClient, ok := req.ProviderData.(*api.ProviderClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.ProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	if providerClient.Api == nil {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Provider client Api is nil.")
		return
	}
	r.FabricLinkClient = newFabricLinkClient(providerClient.Api)
}

func (r *FabricLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
	defer exitContext()

	var plan FabricLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tables []string
	resp.Diagnostics.Append(plan.TableNames.ElementsAs(ctx, &tables, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.FabricLinkClient.CreateFabricLink(
		ctx,
		plan.EnvironmentId.ValueString(),
		plan.FabricWorkspaceId.ValueString(),
		plan.ConnectionId.ValueString(),
		tables,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Link to Fabric", err.Error())
		return
	}

	plan.Id = types.StringValue(result.LakehouseId)
	plan.MirrorLakehouseId = types.StringValue(result.LakehouseId)
	plan.MirrorWorkspaceId = types.StringValue(result.WorkspaceId)
	plan.DatalakeFolderId = types.StringValue(result.DatalakeFolderId)
	if plan.UniqueName.IsUnknown() {
		plan.UniqueName = types.StringValue(plan.Name.ValueString())
	}
	plan.ProfileState = types.StringValue("Active")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FabricLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
	defer exitContext()

	var state FabricLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The link is server-managed by the athena service; a live read-back of the profile state
	// (GET synapselinkprofiles / a status endpoint) is not yet wired (see DESIGN.md), so state is
	// preserved as-is. TODO: read the synapselinkprofile status to detect drift / external unlink.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FabricLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
	defer exitContext()
	resp.Diagnostics.AddError(
		"powerplatform_fabric_link is a scaffold",
		"Updating a Link to Fabric (e.g. changing table_names via synapselinkprofileentity rows) is not yet wired. See internal/services/fabric_link/DESIGN.md.",
	)
}

func (r *FabricLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, exitContext := helpers.EnterRequestContext(ctx, r.TypeInfo, req)
	defer exitContext()

	var state FabricLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.FabricLinkClient.DeleteFabricLink(ctx, state.EnvironmentId.ValueString(), state.DatalakeFolderId.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to unlink Link to Fabric", err.Error())
	}
}

func (r *FabricLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func profileStateLabel(state int) string {
	switch state {
	case 0:
		return "Inactive"
	case 1:
		return "Active"
	case 2:
		return "Error"
	case 3:
		return "Deleted"
	case 4:
		return "Aborting"
	case 5:
		return "Aborted"
	case 6:
		return "Suspended"
	case 7:
		return "Suspending"
	case 8:
		return "Deactivated"
	default:
		return fmt.Sprintf("Unknown(%d)", state)
	}
}
