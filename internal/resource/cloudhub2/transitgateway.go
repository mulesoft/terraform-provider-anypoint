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
	_ resource.Resource                = &TransitGatewayResource{}
	_ resource.ResourceWithConfigure   = &TransitGatewayResource{}
	_ resource.ResourceWithImportState = &TransitGatewayResource{}
)

// transitGatewayStatusDetached is the synthetic status surfaced when the platform
// reports the attachment as detached-but-registered (the PS-scoped GET-by-id
// 400s). It keeps Read from hard-erroring so plan/destroy still work on an
// attachment that was detached out-of-band (W-23819332).
const transitGatewayStatusDetached = "Detached"

// TransitGatewayResource implements the anypoint_transit_gateway_connection resource.
type TransitGatewayResource struct {
	client *cloudhub2.TransitGatewayClient
}

// TransitGatewayResourceModel describes the resource data model.
type TransitGatewayResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	AwsTransitGatewayID  types.String `tfsdk:"aws_transit_gateway_id"`
	AwsConsoleURL        types.String `tfsdk:"aws_console_url"`
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
	resp.TypeName = req.ProviderTypeName + "_transit_gateway_connection"
}

func (r *TransitGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Transit Gateway connection (attachment) in a CloudHub 2.0 Private Space. " +
			"A Transit Gateway connection links a Private Space to an existing AWS Transit Gateway " +
			"(shared to MuleSoft via AWS RAM) for private network connectivity. The connection goes " +
			"through Pending → Available states. Routes are managed inline via the 'routes' attribute " +
			"and can be updated in place after the connection is created.",
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
				Description: "The AWS Transit Gateway ID discovered by the platform from the resource share, " +
					"as a bare `tgw-...` identifier suitable for passing to the AWS provider. " +
					"This is a computed value set after the TGW attachment is created.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"aws_console_url": schema.StringAttribute{
				Description: "Deep link to this transit gateway in the AWS console, as shown by the " +
					"Anypoint UI's \"View on AWS\" link. Empty when the platform does not supply one. " +
					"Use `aws_transit_gateway_id` for the identifier itself.",
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
				Description: "CIDR routes for the transit gateway connection. The attribute is required " +
					"(it must be present in the configuration) but may be empty: set `routes = []` to " +
					"clear all routes, which is also the zero-diff shape when importing a detached " +
					"connection. Routes are managed inline and can be updated in place; updating them " +
					"replaces the full set of routes on the connection.",
				Required:    true,
				ElementType: types.StringType,
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
				Description: "The current status of the transit gateway attachment (e.g. 'Pending', 'Available'). " +
					"Reads 'Detached' when the attachment has been detached from the private space out-of-band " +
					"but is still registered; in that state the connection can no longer be used and should be " +
					"destroyed (or replaced) via Terraform.",
				Computed: true,
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
	plan.AwsTransitGatewayID = types.StringValue(tgw.Status.AWSTransitGatewayID())
	plan.AwsConsoleURL = types.StringValue(tgw.Status.AWSConsoleURL())
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
		// Detached-but-registered (W-23819332): the PS-scoped by-id GET 400s with
		// "attachment is not attached to the private space". This is NOT a transient
		// error and NOT a full delete — the object still exists org-wide. Keep the
		// resource in state and surface a Detached status (preserving the other
		// fields, which cannot be refreshed from the failing GET) so refresh,
		// plan, and destroy keep working instead of the resource being stranded.
		if cloudhub2.IsTransitGatewayDetached(err) {
			tflog.Warn(ctx, "Transit gateway attachment detached from private space; surfacing Detached status so plan/destroy still work",
				map[string]interface{}{"id": state.ID.ValueString()})
			state.Status = types.StringValue(transitGatewayStatusDetached)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading transit gateway",
			"Could not read transit gateway: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(tgw.Name)
	state.AwsTransitGatewayID = types.StringValue(tgw.Status.AWSTransitGatewayID())
	state.AwsConsoleURL = types.StringValue(tgw.Status.AWSConsoleURL())
	state.Status = types.StringValue(tgw.Status.Gateway)

	// Populate resource_share fields from the API so that import works correctly.
	// Without these, an imported resource would show empty values and trigger replacement.
	state.ResourceShareID = types.StringValue(tgw.Spec.ResourceShare.ID)
	state.ResourceShareAccount = types.StringValue(tgw.Spec.ResourceShare.Account)

	// Reconcile routes from the API. Routes are now the single inline source of
	// truth for this resource (there is no separate route sub-resource), so Read
	// must surface genuine out-of-band route drift. To avoid spurious diffs from
	// mere ordering differences, preserve the existing state ordering when the API
	// returns the SAME SET of routes; only overwrite when the set actually changed.
	apiRoutes := tgw.Status.Routes
	if apiRoutes == nil {
		apiRoutes = []string{}
	}
	var stateRoutes []string
	resp.Diagnostics.Append(state.Routes.ElementsAs(ctx, &stateRoutes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !stringSlicesEqualAsSet(stateRoutes, apiRoutes) {
		routesList, diags := types.ListValueFrom(ctx, types.StringType, apiRoutes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Routes = routesList
	}

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

	tflog.Debug(ctx, "Updating transit gateway connection", map[string]interface{}{
		"id":   state.ID.ValueString(),
		"name": plan.Name.ValueString(),
	})

	orgID := plan.OrganizationID.ValueString()
	psID := plan.PrivateSpaceID.ValueString()

	// Extract the desired and prior routes up front. planRoutes is needed by the
	// routes update (step 2), which PATCHes the private-space connection object.
	var planRoutes, stateRoutes []string
	resp.Diagnostics.Append(plan.Routes.ElementsAs(ctx, &planRoutes, false)...)
	resp.Diagnostics.Append(state.Routes.ElementsAs(ctx, &stateRoutes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 1. Update the name if it changed. Renames go to the ORG-scoped endpoint with
	// a name-only body (this is what the Anypoint UI does). The private-space-scoped
	// PATCH the RAML documents silently ignores the name, so it must NOT be used for
	// renames (see UpdateTransitGatewayRequest for the full divergence). Routes are
	// handled entirely by step 2 (the private-space connection PATCH).
	if !plan.Name.Equal(state.Name) {
		updateReq := &cloudhub2.UpdateTransitGatewayRequest{
			Name: plan.Name.ValueString(),
		}
		if _, err := r.client.UpdateTransitGateway(ctx, orgID, psID, state.ID.ValueString(), updateReq); err != nil {
			resp.Diagnostics.AddError(
				"Error updating transit gateway connection",
				"Could not update transit gateway connection name: "+err.Error(),
			)
			return
		}
	}

	// 2. Update routes if the set changed. Routes are a field on the private-space
	// connection object (the RAML's dedicated /routes sub-resource does not exist —
	// it 404s), so this PATCHes the connection with a {name, routes} body. The name
	// is echoed to satisfy the handler's required-name check; the endpoint ignores
	// it for updates (renames go through step 1). Order-only changes are not drift.
	if !stringSlicesEqualAsSet(planRoutes, stateRoutes) {
		if err := r.client.UpdateTransitGatewayRoutes(ctx, orgID, psID, state.ID.ValueString(), plan.Name.ValueString(), planRoutes); err != nil {
			resp.Diagnostics.AddError(
				"Error updating transit gateway routes",
				"Could not update transit gateway routes: "+err.Error(),
			)
			return
		}
	}

	// 3. Re-read the connection to capture the latest computed status/aws id.
	tgw, err := r.client.GetTransitGateway(ctx, orgID, psID, state.ID.ValueString())
	if err != nil {
		// If the attachment detached out-of-band between the PATCH and this re-read,
		// don't fail the whole update — surface the Detached status and keep the
		// applied plan values (W-23819332).
		if cloudhub2.IsTransitGatewayDetached(err) {
			tflog.Warn(ctx, "Transit gateway detached during update re-read; surfacing Detached status",
				map[string]interface{}{"id": state.ID.ValueString()})
			plan.ID = state.ID
			plan.AwsTransitGatewayID = state.AwsTransitGatewayID
			plan.AwsConsoleURL = state.AwsConsoleURL
			plan.Status = types.StringValue(transitGatewayStatusDetached)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading transit gateway connection after update",
			"Could not read transit gateway connection: "+err.Error(),
		)
		return
	}

	plan.ID = state.ID
	plan.AwsTransitGatewayID = types.StringValue(tgw.Status.AWSTransitGatewayID())
	plan.AwsConsoleURL = types.StringValue(tgw.Status.AWSConsoleURL())
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

	orgID := parts[0]
	psID := parts[1]
	tgwID := parts[2]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), orgID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), psID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), tgwID)...)

	// Fetch the TGW from the API to seed routes from actual state so the
	// imported resource matches what's actually deployed (no first-plan drift).
	tgw, err := r.client.GetTransitGateway(ctx, orgID, psID, tgwID)
	if err != nil {
		// Allow importing a detached-but-registered attachment (W-23819332): the
		// PS-scoped GET 400s so routes can't be seeded, but the import must still
		// bind so the user can plan/destroy it. Seed an empty route set and a
		// Detached status; the follow-up Read reconciles the rest via the same
		// detached path.
		if cloudhub2.IsTransitGatewayDetached(err) {
			tflog.Warn(ctx, "Importing detached transit gateway; seeding empty routes and Detached status",
				map[string]interface{}{"id": tgwID})
			emptyRoutes, diags := types.ListValueFrom(ctx, types.StringType, []string{})
			resp.Diagnostics.Append(diags...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("routes"), emptyRoutes)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status"), transitGatewayStatusDetached)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Transit Gateway During Import",
			"Could not read transit gateway to seed routes: "+err.Error(),
		)
		return
	}

	routes := tgw.Status.Routes
	if routes == nil {
		routes = []string{}
	}
	routesList, diags := types.ListValueFrom(ctx, types.StringType, routes)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("routes"), routesList)...)
}

// stringSlicesEqualAsSet reports whether a and b contain the same elements,
// ignoring order and duplicates. Used to distinguish real route drift from
// mere ordering differences returned by the API.
func stringSlicesEqualAsSet(a, b []string) bool {
	seen := make(map[string]struct{}, len(a))
	for _, v := range a {
		seen[v] = struct{}{}
	}
	bSet := make(map[string]struct{}, len(b))
	for _, v := range b {
		bSet[v] = struct{}{}
	}
	if len(seen) != len(bSet) {
		return false
	}
	for v := range bSet {
		if _, ok := seen[v]; !ok {
			return false
		}
	}
	return true
}
