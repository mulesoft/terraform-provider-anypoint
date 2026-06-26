package accessmanagement

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

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &RoleUsersResource{}
	_ resource.ResourceWithConfigure   = &RoleUsersResource{}
	_ resource.ResourceWithImportState = &RoleUsersResource{}
)

// RoleUsersResource is the resource implementation.
type RoleUsersResource struct {
	client *accessmanagement.RoleUsersClient
}

// RoleUsersResourceModel describes the resource data model.
type RoleUsersResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RoleGroupID    types.String `tfsdk:"role_group_id"`
	UserID         types.String `tfsdk:"user_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Username       types.String `tfsdk:"username"`
	FirstName      types.String `tfsdk:"first_name"`
	LastName       types.String `tfsdk:"last_name"`
	Email          types.String `tfsdk:"email"`
}

func NewRoleUsersResource() resource.Resource {
	return &RoleUsersResource{}
}

// Metadata returns the resource type name.
func (r *RoleUsersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_users"
}

// Schema defines the schema for the resource.
func (r *RoleUsersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a user to a role group. This creates a membership between a user " +
			"and a role group, granting the user all permissions assigned to that role group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for this user-role-group assignment (format: {role_group_id}:{user_id}).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_group_id": schema.StringAttribute{
				Description: "The ID of the role group to add the user to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The ID of the user to add to the role group. Use the anypoint_users data source to look up by username.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider's org.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The username of the assigned user (computed after creation).",
				Computed:    true,
			},
			"first_name": schema.StringAttribute{
				Description: "The first name of the assigned user.",
				Computed:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "The last name of the assigned user.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email of the assigned user.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RoleUsersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	usersClient, err := accessmanagement.NewRoleUsersClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Users Client",
			"An unexpected error occurred when creating the client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = usersClient
}

// Create adds a user to a role group.
func (r *RoleUsersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleUsersResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	roleGroupID := plan.RoleGroupID.ValueString()
	userID := plan.UserID.ValueString()

	// Add user to role group
	err := r.client.AddUserToRoleGroup(ctx, orgID, roleGroupID, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding user to role group",
			"Could not add user: "+err.Error(),
		)
		return
	}

	// Read back to get user details
	user, err := r.client.GetRoleGroupUser(ctx, orgID, roleGroupID, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user after assignment",
			"User was added but could not read back details: "+err.Error(),
		)
		return
	}

	// Set state
	plan.ID = types.StringValue(roleGroupID + ":" + userID)
	plan.OrganizationID = types.StringValue(orgID)
	plan.Username = types.StringValue(user.Username)
	plan.FirstName = types.StringValue(user.FirstName)
	plan.LastName = types.StringValue(user.LastName)
	plan.Email = types.StringValue(user.Email)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleUsersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleUsersResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	roleGroupID := state.RoleGroupID.ValueString()
	userID := state.UserID.ValueString()

	user, err := r.client.GetRoleGroupUser(ctx, orgID, roleGroupID, userID)
	if err != nil {
		if client.IsNotFound(err) {
			// User was removed from role group externally
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading role group user",
			"Could not read user assignment: "+err.Error(),
		)
		return
	}

	// Update state
	state.Username = types.StringValue(user.Username)
	state.FirstName = types.StringValue(user.FirstName)
	state.LastName = types.StringValue(user.LastName)
	state.Email = types.StringValue(user.Email)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op — all fields use RequiresReplace.
func (r *RoleUsersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No-op: all fields are ForceNew
}

// Delete removes a user from a role group.
func (r *RoleUsersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleUsersResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	roleGroupID := state.RoleGroupID.ValueString()
	userID := state.UserID.ValueString()

	err := r.client.RemoveUserFromRoleGroup(ctx, orgID, roleGroupID, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing user from role group",
			"Could not remove user: "+err.Error(),
		)
		return
	}
}

// ImportState imports a role-user assignment by composite ID (format: {role_group_id}:{user_id}).
func (r *RoleUsersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: {role_group_id}:{user_id}, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_group_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
}
