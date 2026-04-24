package cloudhub2

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &TransitGatewayResource{}
	_ resource.ResourceWithConfigure   = &TransitGatewayResource{}
	_ resource.ResourceWithImportState = &TransitGatewayResource{}
)

// TransitGatewayResource is the resource implementation.
type TransitGatewayResource struct {
	client *cloudhub2.TransitGatewayClient
}

// TransitGatewayResourceModel describes the resource data model.
type TransitGatewayResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	PrivateSpaceID       types.String `tfsdk:"private_space_id"`
	OrganizationID       types.String `tfsdk:"organization_id"`
	Name                 types.String `tfsdk:"name"`
	ResourceShareID      types.String `tfsdk:"resource_share_id"`
	ResourceShareAccount types.String `tfsdk:"resource_share_account"`
	Routes               types.List   `tfsdk:"routes"`
	// Computed fields from response
	Spec   types.Object `tfsdk:"spec"`
	Status types.Object `tfsdk:"status"`
}

func NewTransitGatewayResource() resource.Resource {
	return &TransitGatewayResource{}
}

// Metadata returns the resource type name.
func (r *TransitGatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transit_gateway"
}

// Schema defines the schema for the resource.
func (r *TransitGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Transit Gateway in a CloudHub 2.0 private space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the Transit Gateway.",
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
				Description: "The name of the Transit Gateway.",
				Required:    true,
			},
			"resource_share_id": schema.StringAttribute{
				Description: "The resource share ID for the Transit Gateway.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_share_account": schema.StringAttribute{
				Description: "The resource share account for the Transit Gateway.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"routes": schema.ListAttribute{
				Description: "List of route CIDR blocks for the Transit Gateway.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"spec": schema.SingleNestedAttribute{
				Description: "The specification of the Transit Gateway.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"resource_share": schema.SingleNestedAttribute{
						Description: "Resource share information.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"id": schema.StringAttribute{
								Description: "Resource share ID.",
								Computed:    true,
							},
							"account": schema.StringAttribute{
								Description: "Resource share account.",
								Computed:    true,
							},
						},
					},
					"region": schema.StringAttribute{
						Description: "The region of the Transit Gateway.",
						Computed:    true,
					},
					"space_name": schema.StringAttribute{
						Description: "The space name.",
						Computed:    true,
					},
				},
			},
			"status": schema.SingleNestedAttribute{
				Description: "The status of the Transit Gateway.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"gateway": schema.StringAttribute{
						Description: "Gateway status.",
						Computed:    true,
					},
					"attachment": schema.StringAttribute{
						Description: "Attachment status.",
						Computed:    true,
					},
					"tgw_resource": schema.StringAttribute{
						Description: "TGW resource link.",
						Computed:    true,
					},
					"routes": schema.ListAttribute{
						Description: "List of active routes.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *TransitGatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	transitGatewayClient, err := cloudhub2.NewTransitGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Transit Gateway Client",
			"An unexpected error occurred when creating the Transit Gateway client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = transitGatewayClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *TransitGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TransitGatewayResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert routes from Terraform list to string slice
	var routes []string
	resp.Diagnostics.Append(data.Routes.ElementsAs(ctx, &routes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Build the request
	createRequest := &cloudhub2.CreateTransitGatewayRequest{
		Name:                 data.Name.ValueString(),
		ResourceShareID:      data.ResourceShareID.ValueString(),
		ResourceShareAccount: data.ResourceShareAccount.ValueString(),
		Routes:               routes,
	}

	// Create the Transit Gateway
	transitGateway, err := r.client.CreateTransitGateway(ctx, orgID, data.PrivateSpaceID.ValueString(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Transit Gateway",
			"Could not create Transit Gateway: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(transitGateway.ID)
	data.OrganizationID = types.StringValue(orgID) // Set the actual org ID used
	data.Name = types.StringValue(transitGateway.Name)

	// Map spec
	resourceShareObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"id":      types.StringType,
			"account": types.StringType,
		},
		map[string]attr.Value{
			"id":      types.StringValue(transitGateway.Spec.ResourceShare.ID),
			"account": types.StringValue(transitGateway.Spec.ResourceShare.Account),
		},
	)

	specObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"resource_share": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":      types.StringType,
					"account": types.StringType,
				},
			},
			"region":     types.StringType,
			"space_name": types.StringType,
		},
		map[string]attr.Value{
			"resource_share": resourceShareObj,
			"region":         types.StringValue(transitGateway.Spec.Region),
			"space_name":     types.StringValue(transitGateway.Spec.SpaceName),
		},
	)

	data.Spec = specObj

	// Map status
	statusRoutes, _ := types.ListValueFrom(ctx, types.StringType, transitGateway.Status.Routes)

	statusObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"gateway":      types.StringType,
			"attachment":   types.StringType,
			"tgw_resource": types.StringType,
			"routes":       types.ListType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"gateway":      types.StringValue(transitGateway.Status.Gateway),
			"attachment":   types.StringValue(transitGateway.Status.Attachment),
			"tgw_resource": types.StringValue(transitGateway.Status.TGWResource),
			"routes":       statusRoutes,
		},
	)

	data.Status = statusObj

	tflog.Trace(ctx, "created Transit Gateway")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TransitGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TransitGatewayResourceModel

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

	// Get the Transit Gateway from the API
	transitGateway, err := r.client.GetTransitGateway(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Transit Gateway",
			"Could not read Transit Gateway ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(transitGateway.ID)
	data.Name = types.StringValue(transitGateway.Name)

	// Map spec
	resourceShareObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"id":      types.StringType,
			"account": types.StringType,
		},
		map[string]attr.Value{
			"id":      types.StringValue(transitGateway.Spec.ResourceShare.ID),
			"account": types.StringValue(transitGateway.Spec.ResourceShare.Account),
		},
	)

	specObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"resource_share": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":      types.StringType,
					"account": types.StringType,
				},
			},
			"region":     types.StringType,
			"space_name": types.StringType,
		},
		map[string]attr.Value{
			"resource_share": resourceShareObj,
			"region":         types.StringValue(transitGateway.Spec.Region),
			"space_name":     types.StringValue(transitGateway.Spec.SpaceName),
		},
	)

	data.Spec = specObj

	// Map status
	statusRoutes, _ := types.ListValueFrom(ctx, types.StringType, transitGateway.Status.Routes)

	statusObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"gateway":      types.StringType,
			"attachment":   types.StringType,
			"tgw_resource": types.StringType,
			"routes":       types.ListType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"gateway":      types.StringValue(transitGateway.Status.Gateway),
			"attachment":   types.StringValue(transitGateway.Status.Attachment),
			"tgw_resource": types.StringValue(transitGateway.Status.TGWResource),
			"routes":       statusRoutes,
		},
	)

	data.Status = statusObj

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *TransitGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TransitGatewayResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from plan or default to client's org
	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Build the request (only name can be updated)
	updateRequest := &cloudhub2.UpdateTransitGatewayRequest{
		Name: plan.Name.ValueString(),
	}

	// Update the Transit Gateway
	transitGateway, err := r.client.UpdateTransitGateway(ctx, orgID, plan.PrivateSpaceID.ValueString(), plan.ID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Transit Gateway",
			"Could not update Transit Gateway: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	plan.ID = types.StringValue(transitGateway.ID)
	plan.Name = types.StringValue(transitGateway.Name)

	// Map spec
	resourceShareObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"id":      types.StringType,
			"account": types.StringType,
		},
		map[string]attr.Value{
			"id":      types.StringValue(transitGateway.Spec.ResourceShare.ID),
			"account": types.StringValue(transitGateway.Spec.ResourceShare.Account),
		},
	)

	specObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"resource_share": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":      types.StringType,
					"account": types.StringType,
				},
			},
			"region":     types.StringType,
			"space_name": types.StringType,
		},
		map[string]attr.Value{
			"resource_share": resourceShareObj,
			"region":         types.StringValue(transitGateway.Spec.Region),
			"space_name":     types.StringValue(transitGateway.Spec.SpaceName),
		},
	)

	plan.Spec = specObj

	// Map status
	statusRoutes, _ := types.ListValueFrom(ctx, types.StringType, transitGateway.Status.Routes)

	statusObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"gateway":      types.StringType,
			"attachment":   types.StringType,
			"tgw_resource": types.StringType,
			"routes":       types.ListType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"gateway":      types.StringValue(transitGateway.Status.Gateway),
			"attachment":   types.StringValue(transitGateway.Status.Attachment),
			"tgw_resource": types.StringValue(transitGateway.Status.TGWResource),
			"routes":       statusRoutes,
		},
	)

	plan.Status = statusObj

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TransitGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TransitGatewayResourceModel

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

	// Delete the Transit Gateway
	err := r.client.DeleteTransitGateway(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Transit Gateway",
			"Could not delete Transit Gateway: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *TransitGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: private_space_id:transit_gateway_id
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected import identifier with format: private_space_id:transit_gateway_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}
