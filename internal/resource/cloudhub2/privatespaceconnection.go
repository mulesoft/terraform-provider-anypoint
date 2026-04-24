package cloudhub2

import (
	"context"
	"fmt"
	"strings"

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

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &PrivateSpaceConnectionResource{}
	_ resource.ResourceWithConfigure   = &PrivateSpaceConnectionResource{}
	_ resource.ResourceWithImportState = &PrivateSpaceConnectionResource{}
)

// PrivateSpaceConnectionResource is the resource implementation.
type PrivateSpaceConnectionResource struct {
	client *cloudhub2.PrivateSpaceConnectionClient
}

// PrivateSpaceConnectionResourceModel describes the resource data model.
type PrivateSpaceConnectionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	PrivateSpaceID types.String `tfsdk:"private_space_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Status         types.String `tfsdk:"status"`
}

// VPNModel represents a VPN configuration in Terraform (matching official API)
type VPNModel struct {
	Name            types.String     `tfsdk:"name"`
	ConnectionId    types.String     `tfsdk:"connection_id"`
	ConnectionName  types.String     `tfsdk:"connection_name"`
	RemoteAsn       types.Int64      `tfsdk:"remote_asn"`
	LocalAsn        types.Int64      `tfsdk:"local_asn"`
	RemoteIpAddress types.String     `tfsdk:"remote_ip_address"`
	StaticRoutes    []types.String   `tfsdk:"static_routes"`
	VPNTunnels      []VPNTunnelModel `tfsdk:"vpn_tunnels"`
}

// VPNTunnelModel represents individual VPN tunnel configuration in Terraform
type VPNTunnelModel struct {
	// VPNConnectionId            types.String   `tfsdk:"vpn_connection_id"`
	Psk     types.String `tfsdk:"psk"`
	PtpCidr types.String `tfsdk:"ptp_cidr"`
	// LocalExternalIpAddress     types.String   `tfsdk:"local_external_ip_address"`
	// LocalPtpIpAddress          types.String   `tfsdk:"local_ptp_ip_address"`
	// RemotePtpIpAddress         types.String   `tfsdk:"remote_ptp_ip_address"`
	StartupAction              types.String   `tfsdk:"startup_action"`
	DPDTimeoutAction           types.String   `tfsdk:"dpd_timeout_action"`
	RekeyMarginInSeconds       types.Int64    `tfsdk:"rekey_margin_in_seconds"`
	RekeyFuzz                  types.Int64    `tfsdk:"rekey_fuzz"`
	IkeVersions                []types.String `tfsdk:"ike_versions"`
	Phase1DhGroups             []types.Int64  `tfsdk:"phase1_dh_groups"`
	Phase2DhGroups             []types.Int64  `tfsdk:"phase2_dh_groups"`
	Phase1EncryptionAlgorithms []types.String `tfsdk:"phase1_encryption_algorithms"`
	Phase2EncryptionAlgorithms []types.String `tfsdk:"phase2_encryption_algorithms"`
	Phase1IntegrityAlgorithms  []types.String `tfsdk:"phase1_integrity_algorithms"`
	Phase2IntegrityAlgorithms  []types.String `tfsdk:"phase2_integrity_algorithms"`
	// AcceptedRouteCount         types.Int64    `tfsdk:"accepted_route_count"`
	// LastStatusChange           types.String   `tfsdk:"last_status_change"`
	// Status                     types.String   `tfsdk:"status"`
	// StatusMessage              types.String   `tfsdk:"status_message"`
}

func NewPrivateSpaceConnectionResource() resource.Resource {
	return &PrivateSpaceConnectionResource{}
}

// Metadata returns the resource type name.
func (r *PrivateSpaceConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_connection"
}

// Schema defines the schema for the resource.
func (r *PrivateSpaceConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Private Space Connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the private space connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space this connection belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the private space connection.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the private space connection.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the private space connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *PrivateSpaceConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	connectionClient, err := cloudhub2.NewPrivateSpaceConnectionClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Space Connection Client",
			"An unexpected error occurred when creating the Private Space Connection client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = connectionClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *PrivateSpaceConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateSpaceConnectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Create the private space connection
	createRequest := &cloudhub2.CreatePrivateSpaceConnectionRequest{
		Name: data.Name.ValueString(),
		Type: data.Type.ValueString(),
		VPNs: []cloudhub2.VPN{}, // Empty VPNs for now
	}

	connection, err := r.client.CreatePrivateSpaceConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating private space connection",
			"Could not create private space connection: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(connection.ID)
	data.OrganizationID = types.StringValue(orgID) // Set the actual org ID used
	data.Name = types.StringValue(connection.Name)
	data.Type = types.StringValue(connection.Type)
	data.Status = types.StringValue(connection.Status)

	tflog.Trace(ctx, "created private space connection")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *PrivateSpaceConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateSpaceConnectionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Get the private space connection from the API
	connection, err := r.client.GetPrivateSpaceConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading private space connection",
			"Could not read private space connection ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(connection.ID)
	data.Name = types.StringValue(connection.Name)
	data.Type = types.StringValue(connection.Type)
	data.Status = types.StringValue(connection.Status)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *PrivateSpaceConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PrivateSpaceConnectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateRequest := &cloudhub2.UpdatePrivateSpaceConnectionRequest{}
	hasChanges := false

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		updateRequest.Name = &name
		hasChanges = true
	}

	if hasChanges {
		// Determine organization ID from plan or default to client's org
		orgID := plan.OrganizationID.ValueString()
		if orgID == "" {
			orgID = r.client.OrgID
		}

		updateRequest.VPNs = []cloudhub2.VPN{} // Empty VPNs for now

		connection, err := r.client.UpdatePrivateSpaceConnection(ctx, orgID, plan.PrivateSpaceID.ValueString(), plan.ID.ValueString(), updateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating private space connection",
				"Could not update private space connection: "+err.Error(),
			)
			return
		}

		// Map response body to schema
		plan.ID = types.StringValue(connection.ID)
		plan.Name = types.StringValue(connection.Name)
		plan.Type = types.StringValue(connection.Type)
		plan.Status = types.StringValue(connection.Status)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *PrivateSpaceConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateSpaceConnectionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Delete the private space connection
	err := r.client.DeletePrivateSpaceConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting private space connection",
			"Could not delete private space connection: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *PrivateSpaceConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: private_space_id:connection_id
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: private_space_id:connection_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}
