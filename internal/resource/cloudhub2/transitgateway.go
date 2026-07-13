package cloudhub2

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

var (
	_ resource.Resource                = &TransitGatewayResource{}
	_ resource.ResourceWithConfigure   = &TransitGatewayResource{}
	_ resource.ResourceWithImportState = &TransitGatewayResource{}
)

// TransitGatewayResource implements the anypoint_transit_gateway resource.
type TransitGatewayResource struct {
	client *cloudhub2.TransitGatewayClient
}

// TransitGatewayResourceModel describes the resource data model.
type TransitGatewayResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	AwsTransitGatewayID  types.String `tfsdk:"aws_transit_gateway_id"`
	ResourceShareID      types.String `tfsdk:"resource_share_id"`
	ResourceShareAccount types.String `tfsdk:"resource_share_account"`
	Routes               types.List   `tfsdk:"routes"`
	PrivateSpaceID       types.String `tfsdk:"private_space_id"`
	OrganizationID       types.String `tfsdk:"organization_id"`
	Status               types.String `tfsdk:"status"`
}

func NewTransitGatewayResource() resource.Resource {
	return &TransitGatewayResource{}
}

func (r *TransitGatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transit_gateway"
}

func (r *TransitGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Transit Gateway attachment in a CloudHub 2.0 Private Space. " +
			"Transit Gateways connect a Private Space to an AWS Transit Gateway for private network connectivity. " +
			"The attachment goes through Pending → Available states. Additional routes can be managed " +
			"via the anypoint_transit_gateway_route resource after the attachment reaches 'Available' status.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the transit gateway attachment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the transit gateway attachment.",
				Required:    true,
			},
			"aws_transit_gateway_id": schema.StringAttribute{
				Description: "The AWS Transit Gateway ID discovered by the platform from the resource share. " +
					"This is a computed value set after the TGW attachment is created.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_share_id": schema.StringAttribute{
				Description: "The AWS RAM resource share ID in UUID format " +
					"(e.g. 'e8e330a8-4f8c-452b-afd0-7810c41287f1').",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_share_account": schema.StringAttribute{
				Description: "The AWS account ID that owns the Transit Gateway.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"routes": schema.ListAttribute{
				Description: "Initial CIDR routes for the transit gateway (at least one required). " +
					"Additional routes can be managed via the anypoint_transit_gateway_route resource.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the Private Space where this transit gateway is attached.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the transit gateway attachment (e.g. 'Pending', 'Available').",
				Computed:    true,
			},
		},
	}
}

func (r *TransitGatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	tgwClient, err := cloudhub2.NewTransitGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Transit Gateway Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	r.client = tgwClient
}

func (r *TransitGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TransitGatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract routes from the list attribute
	var routes []string
	resp.Diagnostics.Append(plan.Routes.ElementsAs(ctx, &routes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating transit gateway", map[string]interface{}{
		"name":             plan.Name.ValueString(),
		"private_space_id": plan.PrivateSpaceID.ValueString(),
		"organization_id":  plan.OrganizationID.ValueString(),
	})

	createReq := &cloudhub2.CreateTransitGatewayRequest{
		Name:                 plan.Name.ValueString(),
		ResourceShareID:      plan.ResourceShareID.ValueString(),
		ResourceShareAccount: plan.ResourceShareAccount.ValueString(),
		Routes:               routes,
	}

	tgw, err := r.client.CreateTransitGateway(ctx, plan.OrganizationID.ValueString(), plan.PrivateSpaceID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating transit gateway",
			"Could not create transit gateway: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(tgw.ID)
	plan.AwsTransitGatewayID = types.StringValue(tgw.Status.TgwResource)
	plan.Status = types.StringValue(tgw.Status.Gateway)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TransitGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TransitGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tgw, err := r.client.GetTransitGateway(ctx, state.OrganizationID.ValueString(), state.PrivateSpaceID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Transit gateway not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading transit gateway",
			"Could not read transit gateway: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(tgw.Name)
	state.AwsTransitGatewayID = types.StringValue(tgw.Status.TgwResource)
	state.Status = types.StringValue(tgw.Status.Gateway)

	// Populate resource_share fields from the API so that import works correctly.
	// Without these, an imported resource would show empty values and trigger replacement.
	state.ResourceShareID = types.StringValue(tgw.Spec.ResourceShare.ID)
	state.ResourceShareAccount = types.StringValue(tgw.Spec.ResourceShare.Account)

	// NOTE: Do NOT update state.Routes from the API. The API returns ALL routes
	// (including those added by anypoint_transit_gateway_route). If we set state.Routes
	// from the API, the resource would show drift and trigger RequiresReplace whenever
	// additional routes are managed via the separate route resource.
	// For import: routes will be seeded to null/empty — the first plan after import
	// will show drift. One apply settles it (same pattern as AM resources).

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TransitGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TransitGatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TransitGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating transit gateway", map[string]interface{}{
		"id":   state.ID.ValueString(),
		"name": plan.Name.ValueString(),
	})

	updateReq := &cloudhub2.UpdateTransitGatewayRequest{
		Name: plan.Name.ValueString(),
	}

	tgw, err := r.client.UpdateTransitGateway(ctx, plan.OrganizationID.ValueString(), plan.PrivateSpaceID.ValueString(), state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating transit gateway",
			"Could not update transit gateway: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID
	plan.AwsTransitGatewayID = types.StringValue(tgw.Status.TgwResource)
	plan.Status = types.StringValue(tgw.Status.Gateway)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TransitGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TransitGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting transit gateway", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteTransitGateway(ctx, state.OrganizationID.ValueString(), state.PrivateSpaceID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting transit gateway",
			"Could not delete transit gateway: "+err.Error(),
		)
		return
	}
}

// ImportState supports importing an existing transit gateway.
// Import ID format: "org_id/private_space_id/transit_gateway_id"
func (r *TransitGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID format: org_id/private_space_id/transit_gateway_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)

	// Seed routes as empty list so Read doesn't error on the Required field.
	// The first plan after import will show drift (routes = [] vs config).
	emptyRoutes, diags := types.ListValueFrom(ctx, types.StringType, []string{})
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("routes"), emptyRoutes)...)
}
