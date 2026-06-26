package accessmanagement

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	_ resource.Resource                = &TeamMembersResource{}
	_ resource.ResourceWithConfigure   = &TeamMembersResource{}
	_ resource.ResourceWithImportState = &TeamMembersResource{}
)

// TeamMembersResource is the resource implementation.
type TeamMembersResource struct {
	client *accessmanagement.TeamMembersClient
}

// TeamMembersResourceModel describes the resource data model.
type TeamMembersResourceModel struct {
	ID             types.String `tfsdk:"id"`
	TeamID         types.String `tfsdk:"team_id"`
	UserID         types.String `tfsdk:"user_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	MembershipType types.String `tfsdk:"membership_type"`
}

func NewTeamMembersResource() resource.Resource {
	return &TeamMembersResource{}
}

// Metadata returns the resource type name.
func (r *TeamMembersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

// Schema defines the schema for the resource.
func (r *TeamMembersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a user to a team. This creates a membership between a user " +
			"and a team, granting the user access to resources visible to that team.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for this team membership (format: {team_id}:{user_id}).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the team to add the user to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The ID of the user to add to the team. Use the anypoint_users data source to look up by username.",
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
			"membership_type": schema.StringAttribute{
				Description: "The membership type. Valid values: 'member' (default), 'maintainer'. " +
					"Members inherit permissions from the team. " +
					"Maintainers can additionally manage team membership and child teams.",
				Optional: true,
				Computed: true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *TeamMembersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	membersClient, err := accessmanagement.NewTeamMembersClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Members Client",
			"An unexpected error occurred when creating the client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = membersClient
}

// Create adds a user to a team.
func (r *TeamMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamMembersResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := plan.TeamID.ValueString()
	userID := plan.UserID.ValueString()
	membershipType := plan.MembershipType.ValueString()

	// Add user to team
	err := r.client.AddTeamMember(ctx, orgID, teamID, userID, membershipType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding user to team",
			"Could not add user: "+err.Error(),
		)
		return
	}

	// Read back to get membership details.
	// The API has eventual consistency — the member may not appear immediately
	// in the list after PUT returns 204. Retry a few times with a short delay.
	var member *accessmanagement.TeamMember
	for attempt := 0; attempt < 5; attempt++ {
		member, err = r.client.GetTeamMember(ctx, orgID, teamID, userID)
		if err == nil {
			break
		}
		if !client.IsNotFound(err) {
			// Real error, not just eventual consistency
			resp.Diagnostics.AddError(
				"Error reading member after assignment",
				"User was added but could not read back details: "+err.Error(),
			)
			return
		}
		// Not found yet — wait and retry
		time.Sleep(2 * time.Second)
	}
	if member == nil {
		resp.Diagnostics.AddError(
			"Error reading member after assignment",
			"User was added (PUT returned success) but the member did not appear in the team members list after retries. This may be a platform consistency delay.",
		)
		return
	}

	// Set state
	plan.ID = types.StringValue(teamID + ":" + userID)
	plan.OrganizationID = types.StringValue(orgID)
	if member.MembershipType != "" {
		plan.MembershipType = types.StringValue(member.MembershipType)
	} else {
		plan.MembershipType = types.StringValue("member")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TeamMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamMembersResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := state.TeamID.ValueString()
	userID := state.UserID.ValueString()

	member, err := r.client.GetTeamMember(ctx, orgID, teamID, userID)
	if err != nil {
		if client.IsNotFound(err) {
			// Member was removed externally
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading team member",
			"Could not read team membership: "+err.Error(),
		)
		return
	}

	// Update membership_type from API
	if member.MembershipType != "" {
		state.MembershipType = types.StringValue(member.MembershipType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update handles in-place changes (e.g., membership_type from "member" to "maintainer").
func (r *TeamMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamMembersResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := state.TeamID.ValueString()
	userID := state.UserID.ValueString()
	newMembershipType := plan.MembershipType.ValueString()

	// PUT with the new membership_type (same endpoint as Create)
	err := r.client.AddTeamMember(ctx, orgID, teamID, userID, newMembershipType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating team membership",
			"Could not update membership type: "+err.Error(),
		)
		return
	}

	// Update state
	plan.ID = state.ID
	plan.OrganizationID = state.OrganizationID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a user from a team.
func (r *TeamMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamMembersResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := state.TeamID.ValueString()
	userID := state.UserID.ValueString()

	err := r.client.RemoveTeamMember(ctx, orgID, teamID, userID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing user from team",
			"Could not remove user: "+err.Error(),
		)
		return
	}
}

// ImportState imports a team membership by composite ID (format: {team_id}:{user_id}).
func (r *TeamMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: {team_id}:{user_id}, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
}
