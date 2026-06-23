package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &RoleResource{}
	_ resource.ResourceWithConfigure   = &RoleResource{}
	_ resource.ResourceWithImportState = &RoleResource{}
)

// RoleResource is the resource implementation.
type RoleResource struct {
	client *accessmanagement.RoleClient
}

// RoleResourceModel describes the resource data model.
type RoleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Editable       types.Bool   `tfsdk:"editable"`
	ExternalNames  types.List   `tfsdk:"external_names"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// Metadata returns the resource type name.
func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the resource.
func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Platform role group (custom or default). Requires Organization Administrator privileges.",
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
				Description: "A description of the role group.",
				Optional:    true,
				Computed:    true,
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
				Description: "Whether the role group can be edited. Default (system) role groups are not editable.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external_names": schema.ListAttribute{
				Description: "External group names mapped to this role group (for SSO/SAML integration). Read-only.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the role group was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the role group was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	userConfig := &client.UserClientConfig{
		BaseURL:      config.BaseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     config.Username,
		Password:     config.Password,
		Timeout:      config.Timeout,
	}

	roleClient, err := accessmanagement.NewRoleClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Group Client",
			"An unexpected error occurred when creating the Role Group client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = roleClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel

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

	// Create the role group
	roleGroupRequest := &accessmanagement.CreateRoleGroupRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	roleGroup, err := r.client.CreateRoleGroup(ctx, orgID, roleGroupRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating role group",
			"Could not create role group: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	r.mapRoleGroupToState(roleGroup, &data, orgID)

	tflog.Trace(ctx, "created role group")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel

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
			// Role group was deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading role group",
			"Could not read role group ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	r.mapRoleGroupToState(roleGroup, &data, orgID)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RoleResourceModel

	// Read Terraform plan and state data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Build update request — always send both name and description
	updateRequest := &accessmanagement.UpdateRoleGroupRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	roleGroup, err := r.client.UpdateRoleGroup(ctx, orgID, state.ID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating role group",
			"Could not update role group: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	r.mapRoleGroupToState(roleGroup, &plan, orgID)

	tflog.Trace(ctx, "updated role group")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel

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

	err := r.client.DeleteRoleGroup(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted, nothing to do
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting role group",
			"Could not delete role group ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted role group")
}

// ImportState imports the resource by ID.
func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapRoleGroupToState maps an API RoleGroup response to the Terraform state model.
func (r *RoleResource) mapRoleGroupToState(roleGroup *accessmanagement.RoleGroup, data *RoleResourceModel, orgID string) {
	data.ID = types.StringValue(roleGroup.ID)
	data.Name = types.StringValue(roleGroup.Name)
	data.Description = types.StringValue(roleGroup.Description)
	data.OrganizationID = types.StringValue(orgID)
	data.Editable = types.BoolValue(roleGroup.Editable)
	data.CreatedAt = types.StringValue(roleGroup.CreatedAt)
	data.UpdatedAt = types.StringValue(roleGroup.UpdatedAt)

	// Map external_names list
	if len(roleGroup.ExternalNames) > 0 {
		elems := make([]attr.Value, len(roleGroup.ExternalNames))
		for i, name := range roleGroup.ExternalNames {
			elems[i] = types.StringValue(name)
		}
		data.ExternalNames = types.ListValueMust(types.StringType, elems)
	} else {
		data.ExternalNames = types.ListValueMust(types.StringType, []attr.Value{})
	}
}
