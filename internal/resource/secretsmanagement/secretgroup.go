package secretsmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
)

var (
	_ resource.Resource                = &SecretGroupResource{}
	_ resource.ResourceWithConfigure   = &SecretGroupResource{}
	_ resource.ResourceWithImportState = &SecretGroupResource{}
)

type SecretGroupResource struct {
	client *secretsmanagement.SecretGroupClient
}

type SecretGroupResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	Name           types.String `tfsdk:"name"`
	Downloadable   types.Bool   `tfsdk:"downloadable"`
	CurrentState   types.String `tfsdk:"current_state"`
}

func NewSecretGroupResource() resource.Resource {
	return &SecretGroupResource{}
}

func (r *SecretGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group"
}

func (r *SecretGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a secret group in Anypoint Secrets Manager.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the secret group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: "Environment ID where the secret group is created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the secret group.",
				Required:    true,
			},
			"downloadable": schema.BoolAttribute{
				Description: "Whether the secrets in this group can be downloaded.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"current_state": schema.StringAttribute{
				Description: "Current state of the secret group.",
				Computed:    true,
			},
		},
	}
}

func (r *SecretGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	sgClient, err := secretsmanagement.NewSecretGroupClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Secret Group Client",
			"An unexpected error occurred when creating the Secret Group client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = sgClient
}

func (r *SecretGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecretGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	createReq := &secretsmanagement.CreateSecretGroupRequest{
		Name:         data.Name.ValueString(),
		Downloadable: data.Downloadable.ValueBool(),
	}

	plannedName := data.Name
	plannedDownloadable := data.Downloadable

	sg, err := r.client.CreateSecretGroup(ctx, orgID, envID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating secret group", "Could not create secret group: "+err.Error())
		return
	}

	r.flattenSecretGroup(sg, &data, orgID, envID)

	// Preserve planned values if the API didn't return them
	if data.Name.ValueString() == "" {
		data.Name = plannedName
	}
	if !plannedDownloadable.IsNull() && !plannedDownloadable.IsUnknown() {
		data.Downloadable = plannedDownloadable
	}

	tflog.Trace(ctx, "created secret group", map[string]interface{}{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecretGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	sg, err := r.client.GetSecretGroup(ctx, orgID, envID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading secret group", "Could not read secret group: "+err.Error())
		return
	}

	r.flattenSecretGroup(sg, &data, orgID, envID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SecretGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := state.EnvironmentID.ValueString()

	downloadable := plan.Downloadable.ValueBool()
	updateReq := &secretsmanagement.UpdateSecretGroupRequest{
		Name:         plan.Name.ValueString(),
		Downloadable: &downloadable,
	}

	sg, err := r.client.UpdateSecretGroup(ctx, orgID, envID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating secret group", "Could not update secret group: "+err.Error())
		return
	}

	r.flattenSecretGroup(sg, &plan, orgID, envID)
	tflog.Trace(ctx, "updated secret group", map[string]interface{}{"id": sg.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecretGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecretGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	if err := r.client.DeleteSecretGroup(ctx, orgID, envID, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting secret group", "Could not delete secret group: "+err.Error())
		return
	}

	tflog.Trace(ctx, "deleted secret group", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *SecretGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/secret_group_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// --- Flatten ---

func (r *SecretGroupResource) flattenSecretGroup(sg *secretsmanagement.SecretGroupResponse, data *SecretGroupResourceModel, orgID, envID string) {
	data.ID = types.StringValue(sg.Meta.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.Name = types.StringValue(sg.Name)
	data.Downloadable = types.BoolValue(sg.Downloadable)

	if sg.CurrentState != "" {
		data.CurrentState = types.StringValue(sg.CurrentState)
	} else {
		data.CurrentState = types.StringValue("active")
	}
}
