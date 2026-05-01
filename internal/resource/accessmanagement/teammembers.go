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

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &TeamMembersResource{}
	_ resource.ResourceWithImportState = &TeamMembersResource{}
)

func NewTeamMembersResource() resource.Resource {
	return &TeamMembersResource{}
}

// TeamMembersResource defines the resource implementation.
type TeamMembersResource struct {
	client *accessmanagement.TeamMembersClient
}

// TeamMemberModel describes the team member data model for input.
type TeamMemberModel struct {
	ID             types.String `tfsdk:"id"`
	MembershipType types.String `tfsdk:"membership_type"`
}

// TeamMemberDetailsModel describes the team member details data model for computed output.
type TeamMemberDetailsModel struct {
	ID             types.String `tfsdk:"id"`
	Username       types.String `tfsdk:"username"`
	FirstName      types.String `tfsdk:"first_name"`
	LastName       types.String `tfsdk:"last_name"`
	Email          types.String `tfsdk:"email"`
	MembershipType types.String `tfsdk:"membership_type"`
}

// TeamMembersResourceModel describes the resource data model.
type TeamMembersResourceModel struct {
	TeamID         types.String `tfsdk:"team_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Members        types.List   `tfsdk:"members"`
	Users          types.List   `tfsdk:"users"` // Computed field with full user details
	ID             types.String `tfsdk:"id"`
}

func (r *TeamMembersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

func (r *TeamMembersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource to manage team member assignments in Anypoint Platform.",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the team to manage members for.",
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
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "List of team members with their membership types.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the user to add to the team.",
							Required:            true,
						},
						"membership_type": schema.StringAttribute{
							MarkdownDescription: "The membership type for the user (member or maintainer).",
							Required:            true,
						},
					},
				},
			},
			"users": schema.ListNestedAttribute{
				MarkdownDescription: "Computed list of team members with full user details.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the user.",
							Computed:            true,
						},
						"username": schema.StringAttribute{
							MarkdownDescription: "The username of the user.",
							Computed:            true,
						},
						"first_name": schema.StringAttribute{
							MarkdownDescription: "The first name of the user.",
							Computed:            true,
						},
						"last_name": schema.StringAttribute{
							MarkdownDescription: "The last name of the user.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "The email of the user.",
							Computed:            true,
						},
						"membership_type": schema.StringAttribute{
							MarkdownDescription: "The membership type of the user.",
							Computed:            true,
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Team members identifier (same as team_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

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

	teamMembersClient, err := accessmanagement.NewTeamMembersClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Members Client",
			"An unexpected error occurred when creating the team members client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = teamMembersClient
}

func (r *TeamMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TeamMembersResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build member assignments for the request
	members, err := r.buildMemberAssignments(ctx, data.Members)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building member assignments",
			"Could not build member assignments: "+err.Error(),
		)
		return
	}

	// Add the team members
	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.AnypointClient.OrgID
	}

	err = r.client.AddMembersToTeam(ctx, orgID, data.TeamID.ValueString(), members)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating team members",
			"Could not create team members: "+err.Error(),
		)
		return
	}

	// Set the ID to the team ID for Terraform state management
	data.ID = data.TeamID
	// Set the actual organization ID used
	data.OrganizationID = types.StringValue(orgID)

	// Refresh the member details to get complete user information
	err = r.refreshMembers(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing member details",
			"Members were created but could not retrieve member details: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "created team members")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamMembersResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Refresh the member details from the API
	err := r.refreshMembers(ctx, &data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading team members",
			"Could not read team members: "+err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *TeamMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamMembersResourceModel

	// Read Terraform plan and state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build member assignments for current state and planned state
	currentMembers, err := r.buildMemberAssignments(ctx, state.Members)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building current member assignments",
			"Could not build current member assignments: "+err.Error(),
		)
		return
	}

	plannedMembers, err := r.buildMemberAssignments(ctx, plan.Members)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building planned member assignments",
			"Could not build planned member assignments: "+err.Error(),
		)
		return
	}

	// Calculate members to remove (in current but not in planned)
	membersToRemove := r.calculateMembersToRemove(currentMembers, plannedMembers)

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.AnypointClient.OrgID
	}

	// First, remove members that are no longer in the list
	if len(membersToRemove) > 0 {
		err = r.client.RemoveMembersFromTeam(ctx, orgID, plan.TeamID.ValueString(), membersToRemove)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing members from team",
				"Could not remove members from team: "+err.Error(),
			)
			return
		}
		tflog.Trace(ctx, fmt.Sprintf("removed %d members from team", len(membersToRemove)))
	}

	// Then, add/update the new member list (this handles both additions and membership type changes)
	if len(plannedMembers) > 0 {
		err = r.client.AddMembersToTeam(ctx, orgID, plan.TeamID.ValueString(), plannedMembers)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating members for team",
				"Could not update members for team: "+err.Error(),
			)
			return
		}
	}

	// Refresh the member details to get complete user information
	err = r.refreshMembers(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error refreshing member details",
			"Members were updated but could not retrieve member details: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "updated team members")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TeamMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamMembersResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract member IDs for removal
	memberIDs, err := r.extractMemberIDs(ctx, data.Members)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error extracting member IDs",
			"Could not extract member IDs: "+err.Error(),
		)
		return
	}

	// Remove all members from the team
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.AnypointClient.OrgID
	}

	err = r.client.RemoveMembersFromTeam(ctx, orgID, data.TeamID.ValueString(), memberIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting team members",
			"Could not delete team members: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted team members")
}

func (r *TeamMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("team_id"), req, resp)
}

// Helper function to build member assignments from Terraform state
func (r *TeamMembersResource) buildMemberAssignments(ctx context.Context, membersList types.List) ([]accessmanagement.TeamMember, error) {
	if membersList.IsNull() || membersList.IsUnknown() {
		return []accessmanagement.TeamMember{}, nil
	}

	var memberModels []TeamMemberModel
	diags := membersList.ElementsAs(ctx, &memberModels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert members list")
	}

	var members []accessmanagement.TeamMember
	for _, memberModel := range memberModels {
		member := accessmanagement.TeamMember{
			ID:             memberModel.ID.ValueString(),
			MembershipType: memberModel.MembershipType.ValueString(),
		}
		members = append(members, member)
	}

	return members, nil
}

// Helper function to extract member IDs from Terraform state
func (r *TeamMembersResource) extractMemberIDs(ctx context.Context, membersList types.List) ([]string, error) {
	if membersList.IsNull() || membersList.IsUnknown() {
		return []string{}, nil
	}

	var memberModels []TeamMemberModel
	diags := membersList.ElementsAs(ctx, &memberModels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert members list")
	}

	var memberIDs []string
	for _, memberModel := range memberModels {
		memberIDs = append(memberIDs, memberModel.ID.ValueString())
	}

	return memberIDs, nil
}

// Helper function to refresh member details from the API
func (r *TeamMembersResource) refreshMembers(ctx context.Context, data *TeamMembersResourceModel) error {
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.AnypointClient.OrgID
	}

	members, err := r.client.GetTeamMembers(ctx, orgID, data.TeamID.ValueString())
	if err != nil {
		return err
	}

	// Convert members to state for both the simple members list and detailed users list
	membersState, err := r.convertMembersToState(members)
	if err != nil {
		return err
	}
	data.Members = membersState

	usersState, err := r.convertUsersToState(members)
	if err != nil {
		return err
	}
	data.Users = usersState

	return nil
}

// Helper function to convert members to simple Terraform state
func (r *TeamMembersResource) convertMembersToState(members []accessmanagement.TeamMemberDetails) (types.List, error) {
	var memberElements []attr.Value

	for _, member := range members {
		memberObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"id":              types.StringType,
				"membership_type": types.StringType,
			},
			map[string]attr.Value{
				"id":              types.StringValue(member.ID),
				"membership_type": types.StringValue(member.MembershipType),
			},
		)
		if objDiags.HasError() {
			return types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":              types.StringType,
					"membership_type": types.StringType,
				},
			}), fmt.Errorf("failed to create member object")
		}
		memberElements = append(memberElements, memberObj)
	}

	if len(memberElements) == 0 {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"membership_type": types.StringType,
			},
		}), nil
	}

	listValue, listDiags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"membership_type": types.StringType,
			},
		},
		memberElements,
	)
	if listDiags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"membership_type": types.StringType,
			},
		}), fmt.Errorf("failed to create list value")
	}

	return listValue, nil
}

// Helper function to convert members to detailed user Terraform state
func (r *TeamMembersResource) convertUsersToState(members []accessmanagement.TeamMemberDetails) (types.List, error) {
	var userElements []attr.Value

	for _, member := range members {
		userObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"id":              types.StringType,
				"username":        types.StringType,
				"first_name":      types.StringType,
				"last_name":       types.StringType,
				"email":           types.StringType,
				"membership_type": types.StringType,
			},
			map[string]attr.Value{
				"id":              types.StringValue(member.ID),
				"username":        types.StringValue(member.Username),
				"first_name":      types.StringValue(member.FirstName),
				"last_name":       types.StringValue(member.LastName),
				"email":           types.StringValue(member.Email),
				"membership_type": types.StringValue(member.MembershipType),
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
					"membership_type": types.StringType,
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
				"membership_type": types.StringType,
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
				"membership_type": types.StringType,
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
				"membership_type": types.StringType,
			},
		}), fmt.Errorf("failed to create list value")
	}

	return listValue, nil
}

// Helper function to calculate members to remove (members in current but not in planned)
func (r *TeamMembersResource) calculateMembersToRemove(currentMembers, plannedMembers []accessmanagement.TeamMember) []string {
	// Create a map of planned members for quick lookup
	// Key is user ID (since membership type changes are handled by PATCH)
	plannedSet := make(map[string]bool)
	for _, member := range plannedMembers {
		plannedSet[member.ID] = true
	}

	// Find members that are in current but not in planned (only user IDs needed for DELETE)
	var membersToRemove []string
	for _, member := range currentMembers {
		if !plannedSet[member.ID] {
			membersToRemove = append(membersToRemove, member.ID)
		}
	}

	return membersToRemove
}
