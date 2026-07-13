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

var (
	_ resource.Resource                = &TransitGatewayRouteResource{}
	_ resource.ResourceWithConfigure   = &TransitGatewayRouteResource{}
	_ resource.ResourceWithImportState = &TransitGatewayRouteResource{}
)

// TransitGatewayRouteResource implements the anypoint_transit_gateway_route resource.
type TransitGatewayRouteResource struct {
	client *cloudhub2.TransitGatewayClient
}

// TransitGatewayRouteResourceModel describes the resource data model.
type TransitGatewayRouteResourceModel struct {
	ID               types.String `tfsdk:"id"`
	CIDR             types.String `tfsdk:"cidr"`
	TransitGatewayID types.String `tfsdk:"transit_gateway_id"`
	PrivateSpaceID   types.String `tfsdk:"private_space_id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
}

func NewTransitGatewayRouteResource() resource.Resource {
	return &TransitGatewayRouteResource{}
}

func (r *TransitGatewayRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transit_gateway_route"
}

func (r *TransitGatewayRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a static route on a Transit Gateway attachment in a CloudHub 2.0 Private Space. " +
			"Routes can only be added after the transit gateway reaches 'Available' status. " +
			"Use depends_on to ensure the transit gateway is ready before adding routes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier for the route (org_id/private_space_id/tgw_id/cidr).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cidr": schema.StringAttribute{
				Description: "The CIDR block for the route (e.g. '10.1.0.0/16'). " +
					"This defines the destination network that should be routed through the transit gateway.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"transit_gateway_id": schema.StringAttribute{
				Description: "The ID of the transit gateway to add the route to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the Private Space containing the transit gateway.",
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
		},
	}
}

func (r *TransitGatewayRouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TransitGatewayRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TransitGatewayRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating transit gateway route", map[string]interface{}{
		"cidr":               plan.CIDR.ValueString(),
		"transit_gateway_id": plan.TransitGatewayID.ValueString(),
	})

	// Get current routes, add the new one, then PATCH the full list
	currentRoutes, err := r.client.GetTransitGatewayRoutes(
		ctx,
		plan.OrganizationID.ValueString(),
		plan.PrivateSpaceID.ValueString(),
		plan.TransitGatewayID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading current transit gateway routes",
			"Could not read routes before adding: "+err.Error(),
		)
		return
	}

	// Check if route already exists
	newCIDR := plan.CIDR.ValueString()
	for _, route := range currentRoutes {
		if route == newCIDR {
			resp.Diagnostics.AddError(
				"Route already exists",
				fmt.Sprintf("Route %s already exists on transit gateway %s", newCIDR, plan.TransitGatewayID.ValueString()),
			)
			return
		}
	}

	// Append new route and patch
	updatedRoutes := append(currentRoutes, newCIDR)
	err = r.client.UpdateTransitGatewayRoutes(
		ctx,
		plan.OrganizationID.ValueString(),
		plan.PrivateSpaceID.ValueString(),
		plan.TransitGatewayID.ValueString(),
		updatedRoutes,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating transit gateway route",
			"Could not add route: "+err.Error(),
		)
		return
	}

	// Composite ID: org_id/private_space_id/tgw_id/cidr
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s",
		plan.OrganizationID.ValueString(),
		plan.PrivateSpaceID.ValueString(),
		plan.TransitGatewayID.ValueString(),
		plan.CIDR.ValueString(),
	))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TransitGatewayRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TransitGatewayRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	routes, err := r.client.GetTransitGatewayRoutes(
		ctx,
		state.OrganizationID.ValueString(),
		state.PrivateSpaceID.ValueString(),
		state.TransitGatewayID.ValueString(),
	)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Transit gateway not found, removing route from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading transit gateway routes",
			"Could not read transit gateway routes: "+err.Error(),
		)
		return
	}

	// Check if our specific route still exists
	found := false
	for _, route := range routes {
		if route == state.CIDR.ValueString() {
			found = true
			break
		}
	}

	if !found {
		tflog.Warn(ctx, "Transit gateway route not found, removing from state", map[string]interface{}{
			"cidr": state.CIDR.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TransitGatewayRouteResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Routes are immutable — all attributes have RequiresReplace, so Update should never be called.
	resp.Diagnostics.AddError(
		"Update not supported",
		"Transit gateway routes are immutable. Change any attribute to trigger replacement.",
	)
}

func (r *TransitGatewayRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TransitGatewayRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting transit gateway route", map[string]interface{}{
		"cidr":               state.CIDR.ValueString(),
		"transit_gateway_id": state.TransitGatewayID.ValueString(),
	})

	// Get current routes, remove the target one, then PATCH the full list
	currentRoutes, err := r.client.GetTransitGatewayRoutes(
		ctx,
		state.OrganizationID.ValueString(),
		state.PrivateSpaceID.ValueString(),
		state.TransitGatewayID.ValueString(),
	)
	if err != nil {
		if client.IsNotFound(err) {
			return // TGW already deleted, route is gone too
		}
		resp.Diagnostics.AddError(
			"Error reading current transit gateway routes",
			"Could not read routes before removing: "+err.Error(),
		)
		return
	}

	// Remove the target route
	targetCIDR := state.CIDR.ValueString()
	updatedRoutes := make([]string, 0, len(currentRoutes))
	for _, route := range currentRoutes {
		if route != targetCIDR {
			updatedRoutes = append(updatedRoutes, route)
		}
	}

	// If no change (route wasn't there), nothing to do
	if len(updatedRoutes) == len(currentRoutes) {
		return
	}

	err = r.client.UpdateTransitGatewayRoutes(
		ctx,
		state.OrganizationID.ValueString(),
		state.PrivateSpaceID.ValueString(),
		state.TransitGatewayID.ValueString(),
		updatedRoutes,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting transit gateway route",
			"Could not remove route: "+err.Error(),
		)
		return
	}
}

// ImportState supports importing an existing transit gateway route.
// Import ID format: "org_id/private_space_id/transit_gateway_id/cidr"
func (r *TransitGatewayRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	// CIDR contains a "/" so we need at least 5 parts (org/ps/tgw/ip/mask)
	if len(parts) < 5 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID format: org_id/private_space_id/transit_gateway_id/cidr (e.g. org123/ps456/tgw789/10.1.0.0/16)",
		)
		return
	}

	orgID := parts[0]
	psID := parts[1]
	tgwID := parts[2]
	// CIDR is everything after the third "/"
	cidr := strings.Join(parts[3:], "/")

	compositeID := fmt.Sprintf("%s/%s/%s/%s", orgID, psID, tgwID, cidr)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), orgID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), psID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("transit_gateway_id"), tgwID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidr"), cidr)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), compositeID)...)
}
