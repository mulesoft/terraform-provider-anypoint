package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &RoleGroupResource{}
	_ resource.ResourceWithConfigure   = &RoleGroupResource{}
	_ resource.ResourceWithImportState = &RoleGroupResource{}
)

// RoleGroupResource is the resource implementation.
type RoleGroupResource struct {
	client *accessmanagement.RoleGroupClient
}

// ExternalNameModel represents an external name in the resource model
type ExternalNameModel struct {
	ExternalGroupName types.String `tfsdk:"external_group_name"`
	ProviderID        types.String `tfsdk:"provider_id"`
}

// RoleGroupResourceModel describes the resource data model.
type RoleGroupResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	ExternalNames  types.List   `tfsdk:"external_names"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Editable       types.Bool   `tfsdk:"editable"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewRoleGroupResource() resource.Resource {
	return &RoleGroupResource{}
}

// Metadata returns the resource type name.
func (r *RoleGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rolegroup"
}

// Schema defines the schema for the resource.
func (r *RoleGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Platform role group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the role group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the role group.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the role group.",
				Required:    true,
			},
			"external_names": schema.ListNestedAttribute{
				Description: "List of external names for the role group.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"external_group_name": schema.StringAttribute{
							Description: "The external group name.",
							Required:    true,
						},
						"provider_id": schema.StringAttribute{
							Description: "The provider ID.",
							Required:    true,
						},
					},
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the role group will be created. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"editable": schema.BoolAttribute{
				Description: "Whether the role group is editable.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the role group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the role group.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RoleGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	roleGroupClient, err := accessmanagement.NewRoleGroupClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Group Client",
			"An unexpected error occurred when creating the role group client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = roleGroupClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *RoleGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleGroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build external names for the request
	var externalNames []accessmanagement.ExternalNameRequest
	if !data.ExternalNames.IsNull() && !data.ExternalNames.IsUnknown() {
		var externalNameModels []ExternalNameModel
		resp.Diagnostics.Append(data.ExternalNames.ElementsAs(ctx, &externalNameModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, externalName := range externalNameModels {
			externalNames = append(externalNames, accessmanagement.ExternalNameRequest{
				ExternalGroupName: externalName.ExternalGroupName.ValueString(),
				ProviderID:        externalName.ProviderID.ValueString(),
			})
		}
	}

	// Create the role group
	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	createRequest := &accessmanagement.CreateRoleGroupRequest{
		Name:          data.Name.ValueString(),
		Description:   data.Description.ValueString(),
		ExternalNames: externalNames,
	}

	roleGroup, err := r.client.CreateRoleGroup(ctx, orgID, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating role group",
			"Could not create role group: "+err.Error(),
		)
		return
	}

	// Map response to state
	data.ID = types.StringValue(roleGroup.RoleGroupID)
	data.Name = types.StringValue(roleGroup.Name)
	data.Description = types.StringValue(roleGroup.Description)
	// Set the actual organization ID used
	data.OrganizationID = types.StringValue(orgID)
	data.Editable = types.BoolValue(roleGroup.Editable)
	data.CreatedAt = types.StringValue(roleGroup.CreatedAt)
	data.UpdatedAt = types.StringValue(roleGroup.UpdatedAt)

	// Map external names from response (array of strings) to state
	// Note: The API response returns external_names as simple strings, not as objects
	var externalNameElements []attr.Value
	for _, externalName := range roleGroup.ExternalNames {
		// Since the response only gives us strings, we can't reconstruct the full objects
		// We'll set these fields as empty for now
		externalNameObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"external_group_name": types.StringType,
				"provider_id":         types.StringType,
			},
			map[string]attr.Value{
				"external_group_name": types.StringValue(externalName),
				"provider_id":         types.StringValue(""), // Not available in response
			},
		)
		externalNameElements = append(externalNameElements, externalNameObj)
	}

	if len(externalNameElements) > 0 {
		externalNamesList, _ := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"external_group_name": types.StringType,
					"provider_id":         types.StringType,
				},
			},
			externalNameElements,
		)
		data.ExternalNames = externalNamesList
	} else {
		data.ExternalNames = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"external_group_name": types.StringType,
				"provider_id":         types.StringType,
			},
		})
	}

	tflog.Trace(ctx, "created role group")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleGroupResourceModel

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

	// Get the role group from the API
	roleGroup, err := r.client.GetRoleGroup(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading role group",
			"Could not read role group ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update computed fields
	data.ID = types.StringValue(roleGroup.RoleGroupID)
	data.Name = types.StringValue(roleGroup.Name)
	data.Description = types.StringValue(roleGroup.Description)
	data.OrganizationID = types.StringValue(roleGroup.OrgID)
	data.Editable = types.BoolValue(roleGroup.Editable)
	data.CreatedAt = types.StringValue(roleGroup.CreatedAt)
	data.UpdatedAt = types.StringValue(roleGroup.UpdatedAt)

	// Map external names from response
	var externalNameElements []attr.Value
	for _, externalName := range roleGroup.ExternalNames {
		externalNameObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"external_group_name": types.StringType,
				"provider_id":         types.StringType,
			},
			map[string]attr.Value{
				"external_group_name": types.StringValue(externalName),
				"provider_id":         types.StringValue(""), // Not available in response
			},
		)
		externalNameElements = append(externalNameElements, externalNameObj)
	}

	if len(externalNameElements) > 0 {
		externalNamesList, _ := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"external_group_name": types.StringType,
					"provider_id":         types.StringType,
				},
			},
			externalNameElements,
		)
		data.ExternalNames = externalNamesList
	} else {
		data.ExternalNames = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"external_group_name": types.StringType,
				"provider_id":         types.StringType,
			},
		})
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *RoleGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RoleGroupResourceModel

	// Read Terraform plan and state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build external names for the request
	var externalNames []accessmanagement.ExternalNameRequest
	if !plan.ExternalNames.IsNull() && !plan.ExternalNames.IsUnknown() {
		var externalNameModels []ExternalNameModel
		resp.Diagnostics.Append(plan.ExternalNames.ElementsAs(ctx, &externalNameModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, externalName := range externalNameModels {
			externalNames = append(externalNames, accessmanagement.ExternalNameRequest{
				ExternalGroupName: externalName.ExternalGroupName.ValueString(),
				ProviderID:        externalName.ProviderID.ValueString(),
			})
		}
	}

	// Update the role group
	updateRequest := &accessmanagement.UpdateRoleGroupRequest{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		ExternalNames: externalNames,
	}

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	roleGroup, err := r.client.UpdateRoleGroup(ctx, orgID, state.ID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating role group",
			"Could not update role group ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(roleGroup.RoleGroupID)
	plan.Name = types.StringValue(roleGroup.Name)
	plan.Description = types.StringValue(roleGroup.Description)
	plan.OrganizationID = types.StringValue(roleGroup.OrgID)
	plan.Editable = types.BoolValue(roleGroup.Editable)
	plan.CreatedAt = types.StringValue(roleGroup.CreatedAt)
	plan.UpdatedAt = types.StringValue(roleGroup.UpdatedAt)

	// Map external names from response
	var externalNameElements []attr.Value
	for _, externalName := range roleGroup.ExternalNames {
		externalNameObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"external_group_name": types.StringType,
				"provider_id":         types.StringType,
			},
			map[string]attr.Value{
				"external_group_name": types.StringValue(externalName),
				"provider_id":         types.StringValue(""), // Not available in response
			},
		)
		externalNameElements = append(externalNameElements, externalNameObj)
	}

	if len(externalNameElements) > 0 {
		externalNamesList, _ := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"external_group_name": types.StringType,
					"provider_id":         types.StringType,
				},
			},
			externalNameElements,
		)
		plan.ExternalNames = externalNamesList
	} else {
		plan.ExternalNames = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"external_group_name": types.StringType,
				"provider_id":         types.StringType,
			},
		})
	}

	tflog.Trace(ctx, "updated role group")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RoleGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleGroupResourceModel

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

	// Delete the role group
	err := r.client.DeleteRoleGroup(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting role group",
			"Could not delete role group ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted role group")
}

// ImportState imports the resource state.
func (r *RoleGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
