package cloudhub2

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &PrivateNetworkResource{}
	_ resource.ResourceWithConfigure   = &PrivateNetworkResource{}
	_ resource.ResourceWithImportState = &PrivateNetworkResource{}
)

// PrivateNetworkResource is the resource implementation.
type PrivateNetworkResource struct {
	client *cloudhub2.PrivateNetworkClient
}

// PrivateNetworkResourceModel describes the resource data model.
type PrivateNetworkResourceModel struct {
	ID             types.String `tfsdk:"id"`
	PrivateSpaceID types.String `tfsdk:"private_space_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	// Configurable network fields
	Region        types.String `tfsdk:"region"`
	CidrBlock     types.String `tfsdk:"cidr_block"`
	ReservedCIDRs types.List   `tfsdk:"reserved_cidrs"`
	// Computed network fields
	InboundStaticIPs         types.List   `tfsdk:"inbound_static_ips"`
	InboundInternalStaticIPs types.List   `tfsdk:"inbound_internal_static_ips"`
	OutboundStaticIPs        types.List   `tfsdk:"outbound_static_ips"`
	DNSTarget                types.String `tfsdk:"dns_target"`
}

func NewPrivateNetworkResource() resource.Resource {
	return &PrivateNetworkResource{}
}

// Metadata returns the resource type name.
func (r *PrivateNetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network"
}

// Schema defines the schema for the resource.
func (r *PrivateNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Private Network configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the private network.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space this network belongs to.",
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
				Description: "The name of the private network.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The region for the private network.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr_block": schema.StringAttribute{
				Description: "The CIDR block for the private network.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"reserved_cidrs": schema.ListAttribute{
				Description: "The reserved CIDR blocks for the private network.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"inbound_static_ips": schema.ListAttribute{
				Description: "The inbound static IPs for the private network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"inbound_internal_static_ips": schema.ListAttribute{
				Description: "The inbound internal static IPs for the private network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"outbound_static_ips": schema.ListAttribute{
				Description: "The outbound static IPs for the private network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"dns_target": schema.StringAttribute{
				Description: "The DNS target for the private network.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *PrivateNetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	privateNetworkClient, err := cloudhub2.NewPrivateNetworkClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Network Client",
			"An unexpected error occurred when creating the Private Network client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = privateNetworkClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *PrivateNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateNetworkResourceModel

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

	// Create the network configuration request
	var reservedCIDRs []string
	if !data.ReservedCIDRs.IsNull() && !data.ReservedCIDRs.IsUnknown() {
		data.ReservedCIDRs.ElementsAs(ctx, &reservedCIDRs, false)
	}

	networkConfig := &cloudhub2.CreatePrivateNetworkRequest{
		Network: cloudhub2.NetworkConfiguration{
			Region:        data.Region.ValueString(),
			CidrBlock:     data.CidrBlock.ValueString(),
			ReservedCIDRs: reservedCIDRs,
		},
	}

	// Create the private network
	privateSpace, err := r.client.CreatePrivateNetwork(ctx, orgID, data.PrivateSpaceID.ValueString(), networkConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating private network",
			"Could not create private network for space "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(privateSpace.ID)
	data.OrganizationID = types.StringValue(orgID) // Set the actual org ID used
	data.Name = types.StringValue(privateSpace.Name)

	// Map network configuration from the response
	if len(privateSpace.Network.Region) > 0 {
		data.Region = types.StringValue(privateSpace.Network.Region)
	}
	if len(privateSpace.Network.CidrBlock) > 0 {
		data.CidrBlock = types.StringValue(privateSpace.Network.CidrBlock)
	}
	if len(privateSpace.Network.DNSTarget) > 0 {
		data.DNSTarget = types.StringValue(privateSpace.Network.DNSTarget)
	}

	// Convert IP lists to Terraform lists
	inboundStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.InboundStaticIPs)
	data.InboundStaticIPs = inboundStaticIPs

	inboundInternalStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.InboundInternalStaticIPs)
	data.InboundInternalStaticIPs = inboundInternalStaticIPs

	outboundStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.OutboundStaticIPs)
	data.OutboundStaticIPs = outboundStaticIPs

	// Map reserved CIDRs
	reservedCIDRsList, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.ReservedCIDRs)
	data.ReservedCIDRs = reservedCIDRsList

	tflog.Trace(ctx, "created private network")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *PrivateNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateNetworkResourceModel

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

	// Get the private network from the API
	privateSpace, err := r.client.GetPrivateNetwork(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading private network",
			"Could not read private network for space "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(privateSpace.ID)
	data.Name = types.StringValue(privateSpace.Name)

	// Map network configuration from the response
	if len(privateSpace.Network.Region) > 0 {
		data.Region = types.StringValue(privateSpace.Network.Region)
	}
	if len(privateSpace.Network.CidrBlock) > 0 {
		data.CidrBlock = types.StringValue(privateSpace.Network.CidrBlock)
	}
	if len(privateSpace.Network.DNSTarget) > 0 {
		data.DNSTarget = types.StringValue(privateSpace.Network.DNSTarget)
	}

	// Convert IP lists to Terraform lists
	inboundStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.InboundStaticIPs)
	data.InboundStaticIPs = inboundStaticIPs

	inboundInternalStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.InboundInternalStaticIPs)
	data.InboundInternalStaticIPs = inboundInternalStaticIPs

	outboundStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.OutboundStaticIPs)
	data.OutboundStaticIPs = outboundStaticIPs

	// Map reserved CIDRs
	reservedCIDRsList, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.ReservedCIDRs)
	data.ReservedCIDRs = reservedCIDRsList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *PrivateNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PrivateNetworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	var reservedCIDRs []string
	if !plan.ReservedCIDRs.IsNull() && !plan.ReservedCIDRs.IsUnknown() {
		plan.ReservedCIDRs.ElementsAs(ctx, &reservedCIDRs, false)
	}

	updateRequest := &cloudhub2.UpdatePrivateNetworkRequest{
		Network: cloudhub2.NetworkConfiguration{
			Region:        plan.Region.ValueString(),
			CidrBlock:     plan.CidrBlock.ValueString(),
			ReservedCIDRs: reservedCIDRs,
		},
	}

	// Determine organization ID from plan or default to client's org
	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Update the private network
	privateSpace, err := r.client.UpdatePrivateNetwork(ctx, orgID, plan.PrivateSpaceID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating private network",
			"Could not update private network for space "+plan.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	plan.ID = types.StringValue(privateSpace.ID)
	plan.Name = types.StringValue(privateSpace.Name)

	// Map network configuration from the response
	if len(privateSpace.Network.Region) > 0 {
		plan.Region = types.StringValue(privateSpace.Network.Region)
	}
	if len(privateSpace.Network.CidrBlock) > 0 {
		plan.CidrBlock = types.StringValue(privateSpace.Network.CidrBlock)
	}
	if len(privateSpace.Network.DNSTarget) > 0 {
		plan.DNSTarget = types.StringValue(privateSpace.Network.DNSTarget)
	}

	// Convert IP lists to Terraform lists
	inboundStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.InboundStaticIPs)
	plan.InboundStaticIPs = inboundStaticIPs

	inboundInternalStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.InboundInternalStaticIPs)
	plan.InboundInternalStaticIPs = inboundInternalStaticIPs

	outboundStaticIPs, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.OutboundStaticIPs)
	plan.OutboundStaticIPs = outboundStaticIPs

	// Map reserved CIDRs
	reservedCIDRsList, _ := types.ListValueFrom(ctx, types.StringType, privateSpace.Network.ReservedCIDRs)
	plan.ReservedCIDRs = reservedCIDRsList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *PrivateNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Private networks are deleted as part of private space deletion
	// This is a no-op resource deletion
}

// ImportState imports the resource into Terraform state.
func (r *PrivateNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: private_space_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), req.ID)...)
}
