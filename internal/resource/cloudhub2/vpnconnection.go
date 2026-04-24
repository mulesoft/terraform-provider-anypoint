package cloudhub2

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

var (
	_ resource.Resource                = &VPNConnectionResource{}
	_ resource.ResourceWithConfigure   = &VPNConnectionResource{}
	_ resource.ResourceWithImportState = &VPNConnectionResource{}
	_ resource.ResourceWithModifyPlan  = &VPNConnectionResource{}
)

type VPNConnectionResource struct {
	client *cloudhub2.VPNConnectionClient
}

type VPNConnectionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	PrivateSpaceID types.String `tfsdk:"private_space_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	VPNs           types.List   `tfsdk:"vpns"`
}

type VPNResourceModel struct {
	LocalASN        types.String `tfsdk:"local_asn"`
	RemoteASN       types.String `tfsdk:"remote_asn"`
	RemoteIPAddress types.String `tfsdk:"remote_ip_address"`
	StaticRoutes    types.List   `tfsdk:"static_routes"`
	VPNTunnels      types.List   `tfsdk:"vpn_tunnels"`
	// Computed
	Name                types.String `tfsdk:"name"`
	ConnectionName      types.String `tfsdk:"connection_name"`
	VPNConnectionStatus types.String `tfsdk:"vpn_connection_status"`
	VPNID               types.String `tfsdk:"vpn_id"`
	ConnectionID        types.String `tfsdk:"connection_id"`
}

type VPNTunnelResourceModel struct {
	PSK           types.String `tfsdk:"psk"`
	PTPCidr       types.String `tfsdk:"ptp_cidr"`
	StartupAction types.String `tfsdk:"startup_action"`
	// Computed
	IsLogsEnabled types.Bool `tfsdk:"is_logs_enabled"`
}

func NewVPNConnectionResource() resource.Resource {
	return &VPNConnectionResource{}
}

func (r *VPNConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_connection"
}

func (r *VPNConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a VPN connection in a CloudHub 2.0 private space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the VPN connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the VPN connection.",
				Required:    true,
			},
			"vpns": schema.ListNestedAttribute{
				Description: "List of VPN configurations.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"local_asn": schema.StringAttribute{
							Description: "Local ASN for the VPN.",
							Required:    true,
						},
						"remote_asn": schema.StringAttribute{
							Description: "Remote ASN for the VPN.",
							Required:    true,
						},
						"remote_ip_address": schema.StringAttribute{
							Description: "Remote IP address for the VPN.",
							Required:    true,
						},
						"static_routes": schema.ListAttribute{
							Description: "List of static routes.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"vpn_tunnels": schema.ListNestedAttribute{
							Description: "List of VPN tunnel configurations.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"psk": schema.StringAttribute{
										Description: "Pre-shared key for the VPN tunnel.",
										Required:    true,
									},
									"ptp_cidr": schema.StringAttribute{
										Description: "Point-to-point CIDR for the VPN tunnel.",
										Optional:    true,
									},
									"startup_action": schema.StringAttribute{
										Description: "Startup action for the VPN tunnel.",
										Required:    true,
									},
									"is_logs_enabled": schema.BoolAttribute{
										Description: "Whether logs are enabled for the VPN tunnel.",
										Computed:    true,
									},
								},
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the VPN.",
							Optional:    true,
							Computed:    true,
						},
						"connection_name": schema.StringAttribute{
							Description: "The connection name of the VPN.",
							Computed:    true,
						},
						"vpn_connection_status": schema.StringAttribute{
							Description: "The status of the VPN connection.",
							Computed:    true,
						},
						"vpn_id": schema.StringAttribute{
							Description: "The ID of the VPN.",
							Computed:    true,
						},
						"connection_id": schema.StringAttribute{
							Description: "The connection ID of the VPN.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *VPNConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	vpnConnectionClient, err := cloudhub2.NewVPNConnectionClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create VPN Connection Client", "An unexpected error occurred when creating the VPN Connection client. "+err.Error())
		return
	}
	r.client = vpnConnectionClient
}

func (r *VPNConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VPNConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	apiRequest, diags := vpnConnectionResourceModelToAPIRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpnConnection, err := r.client.CreateVPNConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), apiRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create vpn connection, got error: %s", err))
		return
	}

	resourceModel, diags := vpnConnectionAPIResponseToResourceModel(ctx, vpnConnection, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the actual org ID used
	resourceModel.OrganizationID = types.StringValue(orgID)

	tflog.Trace(ctx, "VPN Connection created")
	resp.Diagnostics.Append(resp.State.Set(ctx, &resourceModel)...)
}

func (r *VPNConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VPNConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	vpnConnection, err := r.client.GetVPNConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "VPN connection not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vpn connection, got error: %s", err))
		return
	}

	resourceModel, diags := vpnConnectionAPIResponseToResourceModel(ctx, vpnConnection, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "VPN Connection read")
	resp.Diagnostics.Append(resp.State.Set(ctx, &resourceModel)...)
}

func (r *VPNConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state VPNConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateVPNs, planVPNs []VPNResourceModel
	resp.Diagnostics.Append(state.VPNs.ElementsAs(ctx, &stateVPNs, false)...)
	resp.Diagnostics.Append(plan.VPNs.ElementsAs(ctx, &planVPNs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planVPNsMap := make(map[string]bool)
	for _, vpn := range planVPNs {
		planVPNsMap[vpn.RemoteIPAddress.ValueString()] = true
	}

	// Determine organization ID from plan or default to client's org
	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	for _, stateVPN := range stateVPNs {
		if _, existsInPlan := planVPNsMap[stateVPN.RemoteIPAddress.ValueString()]; !existsInPlan {
			vpnID := stateVPN.VPNID.ValueString()
			tflog.Trace(ctx, fmt.Sprintf("Deleting VPN %s for connection %s", vpnID, state.ID.ValueString()))
			err := r.client.DeleteVPN(ctx, orgID, state.PrivateSpaceID.ValueString(), state.ID.ValueString(), vpnID)
			if err != nil {
				resp.Diagnostics.AddError("Error deleting VPN", fmt.Sprintf("Could not delete VPN %s: %s", vpnID, err.Error()))
				return
			}
		}
	}

	// After any update, read the latest state.
	// If the resource was deleted, this Read will handle removing it from the state.
	readReq := resource.ReadRequest{
		State:        resp.State,
		Private:      req.Private,
		ProviderMeta: req.ProviderMeta,
	}
	readResp := resource.ReadResponse{
		State:   resp.State,
		Private: resp.Private,
	}
	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
}

func (r *VPNConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VPNConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	err := r.client.DeleteVPNConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete vpn connection, got error: %s", err))
		return
	}

	// Wait for the VPN connection to be deleted.
	timeout := time.Minute * 2
	pollInterval := time.Second * 5
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		_, err := r.client.GetVPNConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
		if err != nil {
			if client.IsNotFound(err) {
				// Success, resource is gone
				break
			}
			// Another error occurred
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error while waiting for VPN connection to be deleted: %s", err))
			return
		}

		// If we're here, the resource still exists. Wait before polling again.
		select {
		case <-time.After(pollInterval):
			// Continue to next iteration
		case <-ctx.Done():
			resp.Diagnostics.AddError("Timeout", "Timed out waiting for VPN connection to be deleted.")
			return
		}
	}
}

func (r *VPNConnectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Do nothing if we're creating the resource or the state is otherwise unknown
	if req.State.Raw.IsNull() {
		return
	}

	var planVPNs, stateVPNs types.List

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("vpns"), &planVPNs)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("vpns"), &stateVPNs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the plan has no VPNs, but the state does, then the resource needs to be destroyed.
	// Mark the plan as requiring replacement, which will translate to a destroy operation.
	if (planVPNs.IsNull() || len(planVPNs.Elements()) == 0) && (!stateVPNs.IsNull() && len(stateVPNs.Elements()) > 0) {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("vpns"))
	}
}

func (r *VPNConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "/")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: private_space_id/connection_id. Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}

func vpnConnectionResourceModelToAPIRequest(ctx context.Context, data *VPNConnectionResourceModel) (*cloudhub2.CreateVPNConnectionRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	var vpns []VPNResourceModel
	diags.Append(data.VPNs.ElementsAs(ctx, &vpns, false)...)
	if diags.HasError() {
		return nil, diags
	}

	vpnRequests := make([]cloudhub2.VPNRequest, len(vpns))
	for i, vpn := range vpns {
		var vpnTunnels []VPNTunnelResourceModel
		diags.Append(vpn.VPNTunnels.ElementsAs(ctx, &vpnTunnels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		tunnelRequests := make([]cloudhub2.VPNConnectionTunnelRequest, len(vpnTunnels))
		for j, tunnel := range vpnTunnels {
			tunnelRequests[j] = cloudhub2.VPNConnectionTunnelRequest{
				PSK:           tunnel.PSK.ValueString(),
				PTPCidr:       tunnel.PTPCidr.ValueString(),
				StartupAction: tunnel.StartupAction.ValueString(),
			}
		}

		var staticRoutes []string
		if !vpn.StaticRoutes.IsNull() && !vpn.StaticRoutes.IsUnknown() {
			diags.Append(vpn.StaticRoutes.ElementsAs(ctx, &staticRoutes, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}

		vpnRequests[i] = cloudhub2.VPNRequest{
			LocalASN:        vpn.LocalASN.ValueString(),
			RemoteASN:       vpn.RemoteASN.ValueString(),
			RemoteIPAddress: vpn.RemoteIPAddress.ValueString(),
			StaticRoutes:    staticRoutes,
			VPNTunnels:      tunnelRequests,
		}
	}

	req := &cloudhub2.CreateVPNConnectionRequest{
		Name: data.Name.ValueString(),
		VPNs: vpnRequests,
	}

	return req, diags
}

func vpnConnectionAPIResponseToResourceModel(ctx context.Context, vpnConnection *cloudhub2.VPNConnection, plan *VPNConnectionResourceModel) (*VPNConnectionResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := &VPNConnectionResourceModel{
		ID:             types.StringValue(vpnConnection.ID),
		Name:           types.StringValue(vpnConnection.Name),
		PrivateSpaceID: plan.PrivateSpaceID,
	}

	var plannedVPNs []VPNResourceModel
	diags.Append(plan.VPNs.ElementsAs(ctx, &plannedVPNs, false)...)
	if diags.HasError() {
		return nil, diags
	}
	planVPNsMap := make(map[string]VPNResourceModel)
	for _, vpn := range plannedVPNs {
		planVPNsMap[vpn.RemoteIPAddress.ValueString()] = vpn
	}

	vpnModels := make([]VPNResourceModel, len(vpnConnection.VPNs))
	for i, vpnResp := range vpnConnection.VPNs {
		tunnelModels := make([]VPNTunnelResourceModel, len(vpnResp.VPNTunnels))
		for j, tunnelResp := range vpnResp.VPNTunnels {
			tunnelModels[j] = VPNTunnelResourceModel{
				PSK:           types.StringValue(tunnelResp.PSK),
				PTPCidr:       types.StringValue(tunnelResp.PTPCidr),
				StartupAction: types.StringValue(tunnelResp.StartupAction),
				IsLogsEnabled: types.BoolValue(tunnelResp.IsLogsEnabled),
			}
		}
		tunnelsList, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: getVPNTunnelAttrTypes()}, tunnelModels)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}

		planVPN := planVPNsMap[vpnResp.RemoteIPAddress]

		vpnModels[i] = VPNResourceModel{
			LocalASN:            types.StringValue(strconv.Itoa(vpnResp.LocalASN)),
			RemoteASN:           types.StringValue(strconv.Itoa(vpnResp.RemoteASN)),
			RemoteIPAddress:     types.StringValue(vpnResp.RemoteIPAddress),
			VPNTunnels:          tunnelsList,
			Name:                types.StringValue(vpnResp.Name),
			ConnectionName:      types.StringValue(vpnResp.ConnectionName),
			VPNConnectionStatus: types.StringValue(vpnResp.VPNConnectionStatus),
			VPNID:               types.StringValue(vpnResp.VPNID),
			ConnectionID:        types.StringValue(vpnResp.ConnectionID),
			StaticRoutes:        planVPN.StaticRoutes,
		}
	}
	vpnsList, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: getVPNAttrTypes()}, vpnModels)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	model.VPNs = vpnsList

	return model, diags
}

func getVPNAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"local_asn":             types.StringType,
		"remote_asn":            types.StringType,
		"remote_ip_address":     types.StringType,
		"static_routes":         types.ListType{ElemType: types.StringType},
		"vpn_tunnels":           types.ListType{ElemType: types.ObjectType{AttrTypes: getVPNTunnelAttrTypes()}},
		"name":                  types.StringType,
		"connection_name":       types.StringType,
		"vpn_connection_status": types.StringType,
		"vpn_id":                types.StringType,
		"connection_id":         types.StringType,
	}
}

func getVPNTunnelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"psk":             types.StringType,
		"ptp_cidr":        types.StringType,
		"startup_action":  types.StringType,
		"is_logs_enabled": types.BoolType,
	}
}
