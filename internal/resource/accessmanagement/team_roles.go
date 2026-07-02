package accessmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &TeamRolesResource{}
	_ resource.ResourceWithConfigure   = &TeamRolesResource{}
	_ resource.ResourceWithImportState = &TeamRolesResource{}
)

// TeamRolesResource is the resource implementation.
type TeamRolesResource struct {
	client *accessmanagement.TeamRolesClient
}

// TeamRolesResourceModel describes the resource data model.
type TeamRolesResourceModel struct {
	ID             types.String `tfsdk:"id"`
	TeamID         types.String `tfsdk:"team_id"`
	RoleID         types.String `tfsdk:"role_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ContextParams  types.Map    `tfsdk:"context_params"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
}

func NewTeamRolesResource() resource.Resource {
	return &TeamRolesResource{}
}

// Metadata returns the resource type name.
func (r *TeamRolesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_roles"
}

// Schema defines the schema for the resource.
func (r *TeamRolesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a role (permission) to a team. This grants all team members " +
			"the specified permission scoped by the given context parameters (org, environment).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The role_group_assignment_id returned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the team to assign the role to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "The ID of the role (permission) to assign. Use the anypoint_available_roles data source to find valid role IDs.",
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
			"context_params": schema.MapAttribute{
				Description: "Context parameters that scope the permission. Common keys: " +
					"'org' (organization ID), 'envId' (environment ID). " +
					"Use the anypoint_available_roles data source to determine required context params for each role.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the assigned role (computed after creation).",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the assigned role (computed after creation).",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *TeamRolesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	rolesClient, err := accessmanagement.NewTeamRolesClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Roles Client",
			"An unexpected error occurred when creating the client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = rolesClient
}

// Create assigns a role to a team.
func (r *TeamRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamRolesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := plan.TeamID.ValueString()
	roleID := plan.RoleID.ValueString()

	// Convert context_params from types.Map to map[string]string
	contextParams := make(map[string]string)
	if !plan.ContextParams.IsNull() && !plan.ContextParams.IsUnknown() {
		elements := plan.ContextParams.Elements()
		for k, v := range elements {
			contextParams[k] = v.(types.String).ValueString()
		}
	}

	// Assign role to team
	assignReq := &accessmanagement.AssignTeamRoleRequest{
		RoleID:        roleID,
		ContextParams: contextParams,
	}

	_, err := r.client.AssignTeamRole(ctx, orgID, teamID, assignReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning role to team",
			"Could not assign role: "+err.Error(),
		)
		return
	}

	// Read back to get full details (name, description)
	fullAssignment, err := r.client.GetTeamRoleAssignment(ctx, orgID, teamID, roleID, contextParams)
	if err != nil {
		// Assignment succeeded but couldn't read details — set what we have
		plan.ID = types.StringValue(teamID + ":" + roleID)
		plan.OrganizationID = types.StringValue(orgID)
		plan.Name = types.StringValue("")
		plan.Description = types.StringValue("")
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// Set state — use team_id:role_id as composite ID since API doesn't return assignment_id
	assignmentID := fullAssignment.RoleGroupAssignmentID
	if assignmentID == "" {
		assignmentID = teamID + ":" + roleID
	}
	plan.ID = types.StringValue(assignmentID)
	plan.OrganizationID = types.StringValue(orgID)
	plan.Name = types.StringValue(fullAssignment.Name)
	plan.Description = types.StringValue(fullAssignment.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TeamRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamRolesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := state.TeamID.ValueString()
	roleID := state.RoleID.ValueString()

	// Convert context_params from types.Map to map[string]string
	contextParams := make(map[string]string)
	if !state.ContextParams.IsNull() && !state.ContextParams.IsUnknown() {
		elements := state.ContextParams.Elements()
		for k, v := range elements {
			contextParams[k] = v.(types.String).ValueString()
		}
	}

	assignment, err := r.client.GetTeamRoleAssignment(ctx, orgID, teamID, roleID, contextParams)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading team role assignment",
			"Could not read role assignment: "+err.Error(),
		)
		return
	}

	// Update computed fields
	state.Name = types.StringValue(assignment.Name)
	state.Description = types.StringValue(assignment.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op — all fields use RequiresReplace.
func (r *TeamRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No-op: all mutable fields use RequiresReplace
}

// Delete removes a role assignment from a team.
func (r *TeamRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamRolesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := state.TeamID.ValueString()
	roleID := state.RoleID.ValueString()

	// Convert context_params from types.Map to map[string]string
	contextParams := make(map[string]string)
	if !state.ContextParams.IsNull() && !state.ContextParams.IsUnknown() {
		elements := state.ContextParams.Elements()
		for k, v := range elements {
			contextParams[k] = v.(types.String).ValueString()
		}
	}

	unassignReq := &accessmanagement.AssignTeamRoleRequest{
		RoleID:        roleID,
		ContextParams: contextParams,
	}

	err := r.client.UnassignTeamRole(ctx, orgID, teamID, unassignReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing role from team",
			"Could not unassign role: "+err.Error(),
		)
		return
	}
}

// ImportState imports a team role assignment by composite ID.
// Format: {team_id}:{role_id}:{key1=val1,key2=val2}
// Example: abc123:d74ef94a-...:org=6c3c4eb3-...
// For environment-scoped roles: abc123:fa6b43ac-...:org=6c3c4eb3-...,envId=0f64e47f-...
func (r *TeamRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: {team_id}:{role_id}:{key1=val1,key2=val2}, got: %s", req.ID),
		)
		return
	}

	teamID := parts[0]
	roleID := parts[1]
	contextParamsStr := parts[2]

	// Parse context_params from "key1=val1,key2=val2" format
	contextParams := make(map[string]string)
	pairs := strings.Split(contextParamsStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 || kv[0] == "" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Invalid context_params format in import ID. Expected key=value pairs separated by commas, got: %s", contextParamsStr),
			)
			return
		}
		contextParams[kv[0]] = kv[1]
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), teamID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), roleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("context_params"), contextParams)...)
	// Set a composite ID for state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), teamID+":"+roleID)...)
}
