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
	_ resource.Resource                = &RoleGroupUsersResource{}
	_ resource.ResourceWithConfigure   = &RoleGroupUsersResource{}
	_ resource.ResourceWithImportState = &RoleGroupUsersResource{}
)

// RoleGroupUsersResource is the resource implementation.
type RoleGroupUsersResource struct {
	client *accessmanagement.RoleGroupUsersClient
}

// UserModel represents a user in the resource model
type UserModel struct {
	ID             types.String `tfsdk:"id"`
	Username       types.String `tfsdk:"username"`
	FirstName      types.String `tfsdk:"first_name"`
	LastName       types.String `tfsdk:"last_name"`
	Email          types.String `tfsdk:"email"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	IDProviderID   types.String `tfsdk:"idprovider_id"`
}

// RoleGroupUsersResourceModel describes the resource data model.
type RoleGroupUsersResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RoleGroupID    types.String `tfsdk:"rolegroup_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	UserIDs        types.List   `tfsdk:"user_ids"`
	Users          types.List   `tfsdk:"users"`
}

func NewRoleGroupUsersResource() resource.Resource {
	return &RoleGroupUsersResource{}
}

// Metadata returns the resource type name.
func (r *RoleGroupUsersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rolegroup_users"
}

// Schema defines the schema for the resource.
func (r *RoleGroupUsersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages user assignments for an Anypoint Platform role group. This resource manages all users assigned to a role group as a single unit.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this resource (same as rolegroup_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rolegroup_id": schema.StringAttribute{
				Description: "The ID of the role group to assign users to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the role group is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_ids": schema.ListAttribute{
				Description: "List of user IDs to assign to the role group.",
				Required:    true,
				ElementType: types.StringType,
			},
			"users": schema.ListNestedAttribute{
				Description: "List of users assigned to the role group (computed).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The user ID.",
							Computed:    true,
						},
						"username": schema.StringAttribute{
							Description: "The username.",
							Computed:    true,
						},
						"first_name": schema.StringAttribute{
							Description: "The user's first name.",
							Computed:    true,
						},
						"last_name": schema.StringAttribute{
							Description: "The user's last name.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The user's email address.",
							Computed:    true,
						},
						"organization_id": schema.StringAttribute{
							Description: "The organization ID.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the user is enabled.",
							Computed:    true,
						},
						"idprovider_id": schema.StringAttribute{
							Description: "The identity provider ID.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RoleGroupUsersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	roleGroupUsersClient, err := accessmanagement.NewRoleGroupUsersClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Group Users Client",
			"An unexpected error occurred when creating the role group users client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = roleGroupUsersClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *RoleGroupUsersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleGroupUsersResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build user IDs list for the request
	userIDs, err := r.extractUserIDs(ctx, data.UserIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error extracting user IDs",
			"Could not extract user IDs: "+err.Error(),
		)
		return
	}

	// Assign users to the role group
	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	err = r.client.AssignUsersToRoleGroup(ctx, orgID, data.RoleGroupID.ValueString(), userIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning users to role group",
			"Could not assign users to role group: "+err.Error(),
		)
		return
	}

	// Set the ID (use rolegroup_id as the resource ID)
	data.ID = data.RoleGroupID
	// Set the actual organization ID used
	data.OrganizationID = types.StringValue(orgID)

	// Refresh the users list to get complete user details
	err = r.refreshUsers(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing user details",
			"Users were assigned but could not retrieve user details: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "assigned users to role group")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleGroupUsersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleGroupUsersResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Refresh the users list
	err := r.refreshUsers(ctx, &data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading role group users",
			"Could not read users for role group ID "+data.RoleGroupID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *RoleGroupUsersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RoleGroupUsersResourceModel

	// Read Terraform plan and state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build user IDs lists for current state and planned state
	currentUserIDs, err := r.extractUserIDs(ctx, state.UserIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error extracting current user IDs",
			"Could not extract current user IDs: "+err.Error(),
		)
		return
	}

	plannedUserIDs, err := r.extractUserIDs(ctx, plan.UserIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error extracting planned user IDs",
			"Could not extract planned user IDs: "+err.Error(),
		)
		return
	}

	// Calculate users to remove (in current but not in planned)
	usersToRemove := r.calculateUsersToRemove(currentUserIDs, plannedUserIDs)

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// First, remove users that are no longer in the list
	if len(usersToRemove) > 0 {
		err = r.client.RemoveUsersFromRoleGroup(ctx, orgID, plan.RoleGroupID.ValueString(), usersToRemove)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing users from role group",
				"Could not remove users from role group: "+err.Error(),
			)
			return
		}
		tflog.Trace(ctx, fmt.Sprintf("removed %d users from role group", len(usersToRemove)))
	}

	// Then, assign the new user list (this handles both additions and maintains existing users)
	if len(plannedUserIDs) > 0 {
		err = r.client.AssignUsersToRoleGroup(ctx, orgID, plan.RoleGroupID.ValueString(), plannedUserIDs)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating users for role group",
				"Could not update users for role group: "+err.Error(),
			)
			return
		}
	}

	// Refresh the users list to get complete user details
	err = r.refreshUsers(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing user details",
			"Users were updated but could not retrieve user details: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "updated users for role group")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RoleGroupUsersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleGroupUsersResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build user IDs list to remove (all current users)
	userIDs, err := r.extractUserIDs(ctx, data.UserIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error extracting user IDs for deletion",
			"Could not extract user IDs: "+err.Error(),
		)
		return
	}

	// Remove all users from the role group
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	err = r.client.RemoveUsersFromRoleGroup(ctx, orgID, data.RoleGroupID.ValueString(), userIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing users from role group",
			"Could not remove users from role group ID "+data.RoleGroupID.ValueString()+": "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "removed users from role group")
}

// ImportState imports the resource state.
func (r *RoleGroupUsersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID should be the rolegroup_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rolegroup_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper function to extract user IDs from Terraform state
func (r *RoleGroupUsersResource) extractUserIDs(ctx context.Context, userIDsState types.List) ([]string, error) {
	var userIDsList []string
	diags := userIDsState.ElementsAs(ctx, &userIDsList, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert user IDs from state")
	}
	return userIDsList, nil
}

// Helper function to refresh users from API and update state
func (r *RoleGroupUsersResource) refreshUsers(ctx context.Context, data *RoleGroupUsersResourceModel) error {
	// Get the users assigned to the role group
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	users, err := r.client.GetRoleGroupUsers(ctx, orgID, data.RoleGroupID.ValueString())
	if err != nil {
		return err
	}

	// Convert users to Terraform state format
	usersState, err := r.convertUsersToState(users)
	if err != nil {
		return err
	}

	data.Users = usersState

	// Update user_ids to match what's actually assigned
	var userIDs []attr.Value
	for _, user := range users {
		userIDs = append(userIDs, types.StringValue(user.ID))
	}

	if len(userIDs) > 0 {
		userIDsList, listDiags := types.ListValue(types.StringType, userIDs)
		if listDiags.HasError() {
			return fmt.Errorf("failed to create user IDs list")
		}
		data.UserIDs = userIDsList
	} else {
		data.UserIDs = types.ListNull(types.StringType)
	}

	return nil
}

// Helper function to convert users from API response to Terraform state
func (r *RoleGroupUsersResource) convertUsersToState(users []accessmanagement.UserAssignment) (types.List, error) {
	var userElements []attr.Value

	for _, user := range users {
		userObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"id":              types.StringType,
				"username":        types.StringType,
				"first_name":      types.StringType,
				"last_name":       types.StringType,
				"email":           types.StringType,
				"organization_id": types.StringType,
				"enabled":         types.BoolType,
				"idprovider_id":   types.StringType,
			},
			map[string]attr.Value{
				"id":              types.StringValue(user.ID),
				"username":        types.StringValue(user.Username),
				"first_name":      types.StringValue(user.FirstName),
				"last_name":       types.StringValue(user.LastName),
				"email":           types.StringValue(user.Email),
				"organization_id": types.StringValue(user.OrganizationID),
				"enabled":         types.BoolValue(user.Enabled),
				"idprovider_id":   types.StringValue(user.IDProviderID),
			},
		)
		if objDiags.HasError() {
			return types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":              types.StringType,
					"username":        types.StringType,
					"first_name":      types.StringType,
					"last_name":       types.StringType,
					"email":           types.StringType,
					"organization_id": types.StringType,
					"enabled":         types.BoolType,
					"idprovider_id":   types.StringType,
				},
			}), fmt.Errorf("failed to create user object")
		}
		userElements = append(userElements, userObj)
	}

	if len(userElements) == 0 {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"username":        types.StringType,
				"first_name":      types.StringType,
				"last_name":       types.StringType,
				"email":           types.StringType,
				"organization_id": types.StringType,
				"enabled":         types.BoolType,
				"idprovider_id":   types.StringType,
			},
		}), nil
	}

	listValue, listDiags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"username":        types.StringType,
				"first_name":      types.StringType,
				"last_name":       types.StringType,
				"email":           types.StringType,
				"organization_id": types.StringType,
				"enabled":         types.BoolType,
				"idprovider_id":   types.StringType,
			},
		},
		userElements,
	)
	if listDiags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"username":        types.StringType,
				"first_name":      types.StringType,
				"last_name":       types.StringType,
				"email":           types.StringType,
				"organization_id": types.StringType,
				"enabled":         types.BoolType,
				"idprovider_id":   types.StringType,
			},
		}), fmt.Errorf("failed to create list value")
	}

	return listValue, nil
}

// Helper function to calculate users to remove (users in current but not in planned)
func (r *RoleGroupUsersResource) calculateUsersToRemove(currentUserIDs, plannedUserIDs []string) []string {
	// Create a map of planned user IDs for quick lookup
	plannedSet := make(map[string]bool)
	for _, userID := range plannedUserIDs {
		plannedSet[userID] = true
	}

	// Find users that are in current but not in planned
	var usersToRemove []string
	for _, userID := range currentUserIDs {
		if !plannedSet[userID] {
			usersToRemove = append(usersToRemove, userID)
		}
	}

	return usersToRemove
}
