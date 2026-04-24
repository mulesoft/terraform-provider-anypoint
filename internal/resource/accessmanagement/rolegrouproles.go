package accessmanagement

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	_ resource.Resource                = &RoleGroupRolesResource{}
	_ resource.ResourceWithConfigure   = &RoleGroupRolesResource{}
	_ resource.ResourceWithImportState = &RoleGroupRolesResource{}
)

// RoleGroupRolesResource is the resource implementation.
type RoleGroupRolesResource struct {
	client *accessmanagement.RoleGroupRolesClient
}

// RoleAssignmentModel represents a role assignment in the resource model
type RoleAssignmentModel struct {
	RoleID        types.String `tfsdk:"role_id"`
	ContextParams types.Map    `tfsdk:"context_params"`
}

// RoleGroupRolesResourceModel describes the resource data model.
type RoleGroupRolesResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RoleGroupID    types.String `tfsdk:"rolegroup_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Roles          types.List   `tfsdk:"roles"`
}

func NewRoleGroupRolesResource() resource.Resource {
	return &RoleGroupRolesResource{}
}

// Metadata returns the resource type name.
func (r *RoleGroupRolesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rolegroup_roles"
}

// Schema defines the schema for the resource.
func (r *RoleGroupRolesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages role assignments for an Anypoint Platform role group. This resource manages all roles assigned to a role group as a single unit.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this resource (same as rolegroup_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rolegroup_id": schema.StringAttribute{
				Description: "The ID of the role group to assign roles to.",
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
			"roles": schema.ListNestedAttribute{
				Description: "List of roles to assign to the role group.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							Description: "The ID of the role to assign.",
							Required:    true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the role assignment (e.g., organization, environment).",
							Required:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RoleGroupRolesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	roleGroupRolesClient, err := accessmanagement.NewRoleGroupRolesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Group Roles Client",
			"An unexpected error occurred when creating the role group roles client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = roleGroupRolesClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *RoleGroupRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleGroupRolesResourceModel

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

	// Build role assignments for the request
	roles, err := r.buildRoleAssignments(ctx, data.Roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building role assignments",
			"Could not build role assignments: "+err.Error(),
		)
		return
	}

	// Assign roles to the role group
	err = r.client.AssignRolesToRoleGroup(ctx, orgID, data.RoleGroupID.ValueString(), roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning roles to role group",
			"Could not assign roles to role group: "+err.Error(),
		)
		return
	}

	// Set the ID (use rolegroup_id as the resource ID)
	data.ID = data.RoleGroupID
	// Set the actual organization ID used
	data.OrganizationID = types.StringValue(orgID)

	tflog.Trace(ctx, "assigned roles to role group")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleGroupRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleGroupRolesResourceModel

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

	// Get the roles assigned to the role group
	roles, err := r.client.GetRoleGroupRoles(ctx, orgID, data.RoleGroupID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading role group roles",
			"Could not read roles for role group ID "+data.RoleGroupID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Convert roles to Terraform state format
	rolesState, err := r.convertRolesToState(ctx, roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting roles to state",
			"Could not convert roles to state: "+err.Error(),
		)
		return
	}

	data.Roles = rolesState

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *RoleGroupRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RoleGroupRolesResourceModel

	// Read Terraform plan and state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build role assignments for current state and planned state
	currentRoles, err := r.buildRoleAssignments(ctx, state.Roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building current role assignments",
			"Could not build current role assignments: "+err.Error(),
		)
		return
	}

	plannedRoles, err := r.buildRoleAssignments(ctx, plan.Roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building planned role assignments",
			"Could not build planned role assignments: "+err.Error(),
		)
		return
	}

	// Calculate roles to remove (in current but not in planned)
	rolesToRemove := r.calculateRolesToRemove(currentRoles, plannedRoles)

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// First, remove roles that are no longer in the list
	if len(rolesToRemove) > 0 {
		err = r.client.RemoveRolesFromRoleGroup(ctx, orgID, plan.RoleGroupID.ValueString(), rolesToRemove)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing roles from role group",
				"Could not remove roles from role group: "+err.Error(),
			)
			return
		}
		tflog.Trace(ctx, fmt.Sprintf("removed %d roles from role group", len(rolesToRemove)))
	}

	// Then, assign the new role list (this handles both additions and maintains existing roles)
	if len(plannedRoles) > 0 {
		err = r.client.AssignRolesToRoleGroup(ctx, orgID, plan.RoleGroupID.ValueString(), plannedRoles)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating roles for role group",
				"Could not update roles for role group: "+err.Error(),
			)
			return
		}
	}

	tflog.Trace(ctx, "updated roles for role group")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RoleGroupRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleGroupRolesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build role assignments to remove (all current roles)
	roles, err := r.buildRoleAssignments(ctx, data.Roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building role assignments for deletion",
			"Could not build role assignments: "+err.Error(),
		)
		return
	}

	// Remove all roles from the role group
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	err = r.client.RemoveRolesFromRoleGroup(ctx, orgID, data.RoleGroupID.ValueString(), roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing roles from role group",
			"Could not remove roles from role group ID "+data.RoleGroupID.ValueString()+": "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "removed roles from role group")
}

// ImportState imports the resource state.
func (r *RoleGroupRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID should be the rolegroup_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rolegroup_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper function to build role assignments from Terraform state
func (r *RoleGroupRolesResource) buildRoleAssignments(ctx context.Context, rolesState types.List) ([]accessmanagement.RoleAssignment, error) {
	var roleModels []RoleAssignmentModel
	diags := rolesState.ElementsAs(ctx, &roleModels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert roles from state")
	}

	var roles []accessmanagement.RoleAssignment
	for _, roleModel := range roleModels {
		// Convert context_params from types.Map to map[string]interface{}
		contextParams := make(map[string]interface{})
		for key, value := range roleModel.ContextParams.Elements() {
			if strValue, ok := value.(types.String); ok {
				contextParams[key] = strValue.ValueString()
			}
		}

		roles = append(roles, accessmanagement.RoleAssignment{
			RoleID:        roleModel.RoleID.ValueString(),
			ContextParams: contextParams,
		})
	}

	return roles, nil
}

// Helper function to convert roles from API response to Terraform state
func (r *RoleGroupRolesResource) convertRolesToState(ctx context.Context, roles []accessmanagement.RoleAssignment) (types.List, error) {
	var roleElements []attr.Value

	for _, role := range roles {
		// Convert context_params from map[string]interface{} to types.Map
		contextParamsMap := make(map[string]attr.Value)
		for key, value := range role.ContextParams {
			if strValue, ok := value.(string); ok {
				contextParamsMap[key] = types.StringValue(strValue)
			} else {
				// Convert other types to string
				contextParamsMap[key] = types.StringValue(fmt.Sprintf("%v", value))
			}
		}

		contextParamsState, mapDiags := types.MapValue(types.StringType, contextParamsMap)
		if mapDiags.HasError() {
			return types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"role_id":        types.StringType,
					"context_params": types.MapType{ElemType: types.StringType},
				},
			}), fmt.Errorf("failed to create context params map")
		}

		roleObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"role_id":        types.StringType,
				"context_params": types.MapType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"role_id":        types.StringValue(role.RoleID),
				"context_params": contextParamsState,
			},
		)
		if objDiags.HasError() {
			return types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"role_id":        types.StringType,
					"context_params": types.MapType{ElemType: types.StringType},
				},
			}), fmt.Errorf("failed to create role object")
		}
		roleElements = append(roleElements, roleObj)
	}

	if len(roleElements) == 0 {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"role_id":        types.StringType,
				"context_params": types.MapType{ElemType: types.StringType},
			},
		}), nil
	}

	listValue, listDiags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"role_id":        types.StringType,
				"context_params": types.MapType{ElemType: types.StringType},
			},
		},
		roleElements,
	)
	if listDiags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"role_id":        types.StringType,
				"context_params": types.MapType{ElemType: types.StringType},
			},
		}), fmt.Errorf("failed to create list value")
	}

	return listValue, nil
}

// Helper function to calculate roles to remove (roles in current but not in planned)
func (r *RoleGroupRolesResource) calculateRolesToRemove(currentRoles, plannedRoles []accessmanagement.RoleAssignment) []accessmanagement.RoleAssignment {
	// Create a map of planned roles for quick lookup
	// Key is a combination of role_id and serialized context_params
	plannedSet := make(map[string]bool)
	for _, role := range plannedRoles {
		key := r.createRoleKey(role)
		plannedSet[key] = true
	}

	// Find roles that are in current but not in planned
	var rolesToRemove []accessmanagement.RoleAssignment
	for _, role := range currentRoles {
		key := r.createRoleKey(role)
		if !plannedSet[key] {
			rolesToRemove = append(rolesToRemove, role)
		}
	}

	return rolesToRemove
}

// Helper function to create a unique key for a role assignment
// This combines role_id with sorted context_params to ensure proper comparison
func (r *RoleGroupRolesResource) createRoleKey(role accessmanagement.RoleAssignment) string {
	// Start with role_id
	key := role.RoleID

	// Add context_params in a deterministic way
	if role.ContextParams != nil {
		// Convert map to sorted key=value pairs for consistent comparison
		var pairs []string
		for k, v := range role.ContextParams {
			pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
		}
		// Sort to ensure consistent ordering
		sort.Strings(pairs)
		if len(pairs) > 0 {
			key += "|" + strings.Join(pairs, "&")
		}
	}

	return key
}
