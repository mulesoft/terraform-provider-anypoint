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

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TeamRolesResource{}
var _ resource.ResourceWithImportState = &TeamRolesResource{}

func NewTeamRolesResource() resource.Resource {
	return &TeamRolesResource{}
}

// TeamRolesResource defines the resource implementation.
type TeamRolesResource struct {
	client *accessmanagement.TeamRolesClient
}

// TeamRoleAssignmentModel describes the role assignment data model for teams.
type TeamRoleAssignmentModel struct {
	RoleID        types.String `tfsdk:"role_id"`
	ContextParams types.Map    `tfsdk:"context_params"`
}

// TeamRolesResourceModel describes the resource data model.
type TeamRolesResourceModel struct {
	TeamID         types.String `tfsdk:"team_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Roles          types.List   `tfsdk:"roles"`
	ID             types.String `tfsdk:"id"`
}

func (r *TeamRolesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_roles"
}

func (r *TeamRolesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource to manage role assignments for a team in Anypoint Platform.",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the team to assign roles to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the team is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"roles": schema.ListNestedAttribute{
				MarkdownDescription: "List of role assignments for the team.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the role to assign.",
							Required:            true,
						},
						"context_params": schema.MapAttribute{
							MarkdownDescription: "Context parameters for the role assignment.",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Team roles identifier (same as team_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TeamRolesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	teamRolesClient, err := accessmanagement.NewTeamRolesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Roles Client",
			"An unexpected error occurred when creating the team roles client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = teamRolesClient
}

func (r *TeamRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TeamRolesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
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

	// Create the team role assignments
	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.AnypointClient.OrgID
	}

	err = r.client.AssignRolesToTeam(ctx, orgID, data.TeamID.ValueString(), roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating team roles",
			"Could not create team roles: "+err.Error(),
		)
		return
	}

	// Set the ID to the team ID for Terraform state management
	data.ID = data.TeamID
	// Set the actual organization ID used
	data.OrganizationID = types.StringValue(orgID)

	tflog.Trace(ctx, "created team roles")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamRolesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the team roles from the API
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.AnypointClient.OrgID
	}

	roles, err := r.client.GetTeamRoles(ctx, orgID, data.TeamID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading team roles",
			"Could not read team roles: "+err.Error(),
		)
		return
	}

	// Convert roles to Terraform state
	rolesState, err := r.convertRolesToState(roles)
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
func (r *TeamRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamRolesResourceModel

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
		orgID = r.client.AnypointClient.OrgID
	}

	// First, remove roles that are no longer in the list
	if len(rolesToRemove) > 0 {
		err = r.client.RemoveRolesFromTeam(ctx, orgID, plan.TeamID.ValueString(), rolesToRemove)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing roles from team",
				"Could not remove roles from team: "+err.Error(),
			)
			return
		}
		tflog.Trace(ctx, fmt.Sprintf("removed %d roles from team", len(rolesToRemove)))
	}

	// Then, assign the new role list (this handles both additions and maintains existing roles)
	if len(plannedRoles) > 0 {
		err = r.client.AssignRolesToTeam(ctx, orgID, plan.TeamID.ValueString(), plannedRoles)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating roles for team",
				"Could not update roles for team: "+err.Error(),
			)
			return
		}
	}

	tflog.Trace(ctx, "updated roles for team")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TeamRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamRolesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build role assignments for removal
	roles, err := r.buildRoleAssignments(ctx, data.Roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building role assignments",
			"Could not build role assignments: "+err.Error(),
		)
		return
	}

	// Remove all roles from the team
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.AnypointClient.OrgID
	}

	err = r.client.RemoveRolesFromTeam(ctx, orgID, data.TeamID.ValueString(), roles)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting team roles",
			"Could not delete team roles: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted team roles")
}

func (r *TeamRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("team_id"), req, resp)
}

// Helper function to build role assignments from Terraform state
func (r *TeamRolesResource) buildRoleAssignments(ctx context.Context, rolesList types.List) ([]accessmanagement.TeamRoleAssignment, error) {
	if rolesList.IsNull() || rolesList.IsUnknown() {
		return []accessmanagement.TeamRoleAssignment{}, nil
	}

	var roleModels []TeamRoleAssignmentModel
	diags := rolesList.ElementsAs(ctx, &roleModels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert roles list")
	}

	var roles []accessmanagement.TeamRoleAssignment
	for _, roleModel := range roleModels {
		contextParams := make(map[string]interface{})
		if !roleModel.ContextParams.IsNull() && !roleModel.ContextParams.IsUnknown() {
			contextParamsMap := make(map[string]string)
			diags := roleModel.ContextParams.ElementsAs(ctx, &contextParamsMap, false)
			if diags.HasError() {
				return nil, fmt.Errorf("failed to convert context params")
			}
			for k, v := range contextParamsMap {
				contextParams[k] = v
			}
		}

		role := accessmanagement.TeamRoleAssignment{
			RoleID:        roleModel.RoleID.ValueString(),
			ContextParams: contextParams,
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// Helper function to convert roles to Terraform state
func (r *TeamRolesResource) convertRolesToState(roles []accessmanagement.TeamRoleAssignment) (types.List, error) {
	var roleElements []attr.Value

	for _, role := range roles {
		contextParamsMap := make(map[string]attr.Value)
		for key, value := range role.ContextParams {
			if strValue, ok := value.(string); ok {
				contextParamsMap[key] = types.StringValue(strValue)
			} else {
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
func (r *TeamRolesResource) calculateRolesToRemove(currentRoles, plannedRoles []accessmanagement.TeamRoleAssignment) []accessmanagement.TeamRoleAssignment {
	// Create a map of planned roles for quick lookup
	// Key is a combination of role_id and serialized context_params
	plannedSet := make(map[string]bool)
	for _, role := range plannedRoles {
		key := r.createRoleKey(role)
		plannedSet[key] = true
	}

	// Find roles that are in current but not in planned
	var rolesToRemove []accessmanagement.TeamRoleAssignment
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
func (r *TeamRolesResource) createRoleKey(role accessmanagement.TeamRoleAssignment) string {
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
