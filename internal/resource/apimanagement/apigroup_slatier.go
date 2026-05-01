package apimanagement

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &APIGroupSLATierResource{}
	_ resource.ResourceWithConfigure   = &APIGroupSLATierResource{}
	_ resource.ResourceWithImportState = &APIGroupSLATierResource{}
)

// APIGroupSLATierResource manages SLA tiers on API Group instances.
type APIGroupSLATierResource struct {
	client *apimanagement.GroupSLATierClient
}

// --- Terraform State Model ---

type APIGroupSLATierResourceModel struct {
	ID              types.String `tfsdk:"id"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	EnvironmentID   types.String `tfsdk:"environment_id"`
	GroupInstanceID types.String `tfsdk:"group_instance_id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	AutoApprove     types.Bool   `tfsdk:"auto_approve"`
	Status          types.String `tfsdk:"status"`
	DefaultLimits   types.List   `tfsdk:"default_limits"`
}

var groupSLALimitObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"time_period_in_milliseconds": types.Int64Type,
		"maximum_requests":            types.Int64Type,
		"visible":                     types.BoolType,
	},
}

func NewAPIGroupSLATierResource() resource.Resource {
	return &APIGroupSLATierResource{}
}

func (r *APIGroupSLATierResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_group_sla_tier"
}

func (r *APIGroupSLATierResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an SLA tier for an API Group instance in Anypoint API Manager. " +
			"The group_instance_id is the numeric ID of the group instance " +
			"(visible in the groupInstances URL path in API Manager).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the SLA tier (assigned by the platform).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID. Defaults to the provider's org if omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: "Environment ID where the API Group instance lives.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_instance_id": schema.StringAttribute{
				Description: "Numeric ID of the API Group instance to attach this SLA tier to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the SLA tier.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the SLA tier.",
				Optional:    true,
			},
			"auto_approve": schema.BoolAttribute{
				Description: "Whether subscription requests for this tier are auto-approved.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"status": schema.StringAttribute{
				Description: "Status of the SLA tier (ACTIVE or INACTIVE).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ACTIVE", "INACTIVE"),
				},
			},
			"default_limits": schema.ListNestedAttribute{
				Description: "Rate limits for this SLA tier. Maps to the 'defaultLimits' field in the API.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"time_period_in_milliseconds": schema.Int64Attribute{
							Description: "Duration of the rate-limit window in milliseconds.",
							Required:    true,
						},
						"maximum_requests": schema.Int64Attribute{
							Description: "Maximum number of requests allowed within the window.",
							Required:    true,
						},
						"visible": schema.BoolAttribute{
							Description: "Whether this limit is visible to API consumers.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
					},
				},
			},
		},
	}
}

func (r *APIGroupSLATierResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cfg, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	c, err := apimanagement.NewGroupSLATierClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Group SLA Tier Client",
			"An unexpected error occurred when creating the Group SLA Tier client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}
	r.client = c
}

// --- CRUD ---

func (r *APIGroupSLATierResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIGroupSLATierResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	groupInstanceID, err := strconv.Atoi(data.GroupInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid group_instance_id", "Must be a numeric ID: "+data.GroupInstanceID.ValueString())
		return
	}

	limits, diags := r.expandLimits(ctx, data.DefaultLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &apimanagement.CreateGroupSLATierRequest{
		Name:          data.Name.ValueString(),
		AutoApprove:   data.AutoApprove.ValueBool(),
		DefaultLimits: limits,
		Status:        "ACTIVE",
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		createReq.Description = data.Description.ValueString()
	}
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		createReq.Status = data.Status.ValueString()
	}

	tier, err := r.client.CreateGroupSLATier(ctx, orgID, envID, groupInstanceID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating group SLA tier", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, tier, &data, orgID, envID)...)
	tflog.Trace(ctx, "created group SLA tier", map[string]interface{}{"id": tier.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIGroupSLATierResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIGroupSLATierResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	groupInstanceID, err := strconv.Atoi(data.GroupInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid group_instance_id", "Must be a numeric ID: "+data.GroupInstanceID.ValueString())
		return
	}
	tierID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid tier ID", "Could not parse tier ID: "+data.ID.ValueString())
		return
	}

	tier, err := r.client.GetGroupSLATier(ctx, orgID, envID, groupInstanceID, tierID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading group SLA tier", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, tier, &data, orgID, envID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIGroupSLATierResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state APIGroupSLATierResourceModel
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

	groupInstanceID, err := strconv.Atoi(state.GroupInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid group_instance_id", "Must be a numeric ID: "+state.GroupInstanceID.ValueString())
		return
	}
	tierID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid tier ID", "Could not parse tier ID: "+state.ID.ValueString())
		return
	}

	limits, diags := r.expandLimits(ctx, plan.DefaultLimits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	autoApprove := plan.AutoApprove.ValueBool()
	updateReq := &apimanagement.UpdateGroupSLATierRequest{
		Name:          plan.Name.ValueString(),
		AutoApprove:   &autoApprove,
		DefaultLimits: limits,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		updateReq.Status = plan.Status.ValueString()
	}

	tier, err := r.client.UpdateGroupSLATier(ctx, orgID, envID, groupInstanceID, tierID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating group SLA tier", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, tier, &plan, orgID, envID)...)
	tflog.Trace(ctx, "updated group SLA tier", map[string]interface{}{"id": tier.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APIGroupSLATierResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIGroupSLATierResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	groupInstanceID, err := strconv.Atoi(data.GroupInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid group_instance_id", "Must be a numeric ID: "+data.GroupInstanceID.ValueString())
		return
	}
	tierID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid tier ID", "Could not parse tier ID: "+data.ID.ValueString())
		return
	}

	if err := r.client.DeleteGroupSLATier(ctx, orgID, envID, groupInstanceID, tierID); err != nil {
		resp.Diagnostics.AddError("Error deleting group SLA tier", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted group SLA tier", map[string]interface{}{"id": tierID})
}

func (r *APIGroupSLATierResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Format: organization_id/environment_id/group_instance_id/tier_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/group_instance_id/tier_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_instance_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// --- Helpers ---

func (r *APIGroupSLATierResource) expandLimits(ctx context.Context, limitsList types.List) ([]apimanagement.SLALimit, diag.Diagnostics) {
	var models []SLALimitModel
	diags := limitsList.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	limits := make([]apimanagement.SLALimit, len(models))
	for i, m := range models {
		limits[i] = apimanagement.SLALimit{
			TimePeriodInMilliseconds: int(m.TimePeriodInMilliseconds.ValueInt64()),
			MaximumRequests:          int(m.MaximumRequests.ValueInt64()),
			Visible:                  m.Visible.ValueBool(),
		}
	}
	return limits, nil
}

func (r *APIGroupSLATierResource) flatten(ctx context.Context, tier *apimanagement.GroupSLATier, data *APIGroupSLATierResourceModel, orgID, envID string) diag.Diagnostics {
	data.ID = types.StringValue(strconv.Itoa(tier.ID))
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.Name = types.StringValue(tier.Name)
	data.AutoApprove = types.BoolValue(tier.AutoApprove)

	if tier.Description != "" {
		data.Description = types.StringValue(tier.Description)
	}
	if tier.Status != "" {
		data.Status = types.StringValue(tier.Status)
	}

	limitModels := make([]SLALimitModel, len(tier.DefaultLimits))
	for i, l := range tier.DefaultLimits {
		limitModels[i] = SLALimitModel{
			TimePeriodInMilliseconds: types.Int64Value(int64(l.TimePeriodInMilliseconds)),
			MaximumRequests:          types.Int64Value(int64(l.MaximumRequests)),
			Visible:                  types.BoolValue(l.Visible),
		}
	}

	limitListVal, diags := types.ListValueFrom(ctx, groupSLALimitObjectType, limitModels)
	data.DefaultLimits = limitListVal
	return diags
}
