package accessmanagement

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &TeamResource{}
	_ resource.ResourceWithConfigure   = &TeamResource{}
	_ resource.ResourceWithImportState = &TeamResource{}
)

// TeamResource is the resource implementation.
type TeamResource struct {
	client *accessmanagement.TeamClient
}

// TeamResourceModel describes the resource data model.
type TeamResourceModel struct {
	ID             types.String `tfsdk:"id"`
	TeamName       types.String `tfsdk:"team_name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ParentTeamID   types.String `tfsdk:"parent_team_id"`
	TeamType       types.String `tfsdk:"team_type"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

// Metadata returns the resource type name.
func (r *TeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

// Schema defines the schema for the resource.
func (r *TeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Platform team.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the team.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_name": schema.StringAttribute{
				Description: "The name of the team.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the team will be created. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_team_id": schema.StringAttribute{
				Description: "The ID of the parent team.",
				Required:    true,
			},
			"team_type": schema.StringAttribute{
				Description: "The type of the team.",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the team was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the team was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *TeamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	teamClient, err := accessmanagement.NewTeamClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Client",
			"An unexpected error occurred when creating the Team client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = teamClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TeamResourceModel

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

	// Create the team
	teamRequest := &accessmanagement.CreateTeamRequest{
		TeamName:     data.TeamName.ValueString(),
		ParentTeamID: data.ParentTeamID.ValueString(),
		TeamType:     data.TeamType.ValueString(),
	}

	team, err := r.client.CreateTeam(ctx, orgID, teamRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating team",
			"Could not create team: "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(team.ID)
	data.TeamName = types.StringValue(team.TeamName)
	data.TeamType = types.StringValue(team.TeamType)
	// Set the actual organization ID used (may be from API response or our determined orgID)
	data.OrganizationID = types.StringValue(orgID)
	data.CreatedAt = types.StringValue(team.CreatedAt)
	data.UpdatedAt = types.StringValue(team.UpdatedAt)

	tflog.Trace(ctx, "created team")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamResourceModel

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

	// Get the team from the API
	team, err := r.client.GetTeam(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading team",
			"Could not read team ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(team.ID)
	data.TeamName = types.StringValue(team.TeamName)
	data.TeamType = types.StringValue(team.TeamType)
	data.OrganizationID = types.StringValue(team.OrgID)
	data.CreatedAt = types.StringValue(team.CreatedAt)
	data.UpdatedAt = types.StringValue(team.UpdatedAt)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamResourceModel

	// Read Terraform plan data into the model
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

	// Handle parent_team_id changes first (requires separate API call)
	if !plan.ParentTeamID.Equal(state.ParentTeamID) {
		parentUpdateRequest := &accessmanagement.UpdateTeamParentRequest{
			ParentTeamID: plan.ParentTeamID.ValueString(),
		}

		err := r.client.UpdateTeamParent(ctx, orgID, plan.ID.ValueString(), parentUpdateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating team parent",
				"Could not update team parent: "+err.Error(),
			)
			return
		}
	}

	// Build update request for other fields
	updateRequest := &accessmanagement.UpdateTeamRequest{}
	hasChanges := false

	if !plan.TeamName.Equal(state.TeamName) {
		teamName := plan.TeamName.ValueString()
		updateRequest.TeamName = &teamName
		hasChanges = true
	}

	// Note: team_type cannot be updated via the API, so we ignore changes to it

	if hasChanges {
		team, err := r.client.UpdateTeam(ctx, orgID, plan.ID.ValueString(), updateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating team",
				"Could not update team: "+err.Error(),
			)
			return
		}

		// Map response body to schema
		plan.ID = types.StringValue(team.ID)
		plan.TeamName = types.StringValue(team.TeamName)
		plan.TeamType = types.StringValue(team.TeamType)
		plan.OrganizationID = types.StringValue(team.OrgID)
		plan.CreatedAt = types.StringValue(team.CreatedAt)
		plan.UpdatedAt = types.StringValue(team.UpdatedAt)
	} else if !plan.ParentTeamID.Equal(state.ParentTeamID) {
		// If only parent_team_id was changed, we need to read the updated team
		team, err := r.client.GetTeam(ctx, orgID, plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading team after parent update",
				"Could not read team after updating parent: "+err.Error(),
			)
			return
		}

		// Map response body to schema
		plan.ID = types.StringValue(team.ID)
		plan.TeamName = types.StringValue(team.TeamName)
		plan.TeamType = types.StringValue(team.TeamType)
		plan.OrganizationID = types.StringValue(team.OrgID)
		plan.CreatedAt = types.StringValue(team.CreatedAt)
		plan.UpdatedAt = types.StringValue(team.UpdatedAt)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamResourceModel

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

	// Delete the team
	err := r.client.DeleteTeam(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting team",
			"Could not delete team: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
