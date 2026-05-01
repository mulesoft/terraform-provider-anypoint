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
	_ resource.Resource                = &SLATierResource{}
	_ resource.ResourceWithConfigure   = &SLATierResource{}
	_ resource.ResourceWithImportState = &SLATierResource{}
)

type SLATierResource struct {
	client *apimanagement.SLATierClient
}

type SLATierResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	APIInstanceID  types.String `tfsdk:"api_instance_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	AutoApprove    types.Bool   `tfsdk:"auto_approve"`
	Status         types.String `tfsdk:"status"`
	Limits         types.List   `tfsdk:"limits"`
}

type SLALimitModel struct {
	TimePeriodInMilliseconds types.Int64 `tfsdk:"time_period_in_milliseconds"`
	MaximumRequests          types.Int64 `tfsdk:"maximum_requests"`
	Visible                  types.Bool  `tfsdk:"visible"`
}

var slaLimitObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"time_period_in_milliseconds": types.Int64Type,
		"maximum_requests":            types.Int64Type,
		"visible":                     types.BoolType,
	},
}

func NewSLATierResource() resource.Resource {
	return &SLATierResource{}
}

func (r *SLATierResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_instance_sla_tier"
}

func (r *SLATierResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an SLA tier for an API instance in Anypoint API Manager.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the SLA tier.",
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
				Description: "Environment ID where the API instance lives.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_instance_id": schema.StringAttribute{
				Description: "Numeric ID of the API instance.",
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
				Description: "Description of the SLA tier.",
				Optional:    true,
			},
			"auto_approve": schema.BoolAttribute{
				Description: "Whether requests for this SLA tier are auto-approved.",
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
			"limits": schema.ListNestedAttribute{
				Description: "Rate limits for this SLA tier.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"time_period_in_milliseconds": schema.Int64Attribute{
							Description: "Time period for the rate limit in milliseconds.",
							Required:    true,
						},
						"maximum_requests": schema.Int64Attribute{
							Description: "Maximum number of requests allowed in the time period.",
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

func (r *SLATierResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	slaClient, err := apimanagement.NewSLATierClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create SLA Tier Client",
			"An unexpected error occurred when creating the SLA Tier client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = slaClient
}

func (r *SLATierResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SLATierResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	apiID, err := strconv.Atoi(data.APIInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid api_instance_id", "Must be a numeric ID: "+data.APIInstanceID.ValueString())
		return
	}

	limits, diags := r.expandLimits(ctx, data.Limits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &apimanagement.CreateSLATierRequest{
		Name:         data.Name.ValueString(),
		AutoApprove:  data.AutoApprove.ValueBool(),
		Limits:       limits,
		APIVersionID: data.APIInstanceID.ValueString(),
		Status:       "ACTIVE",
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		createReq.Description = data.Description.ValueString()
	}
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		createReq.Status = data.Status.ValueString()
	}

	tier, err := r.client.CreateSLATier(ctx, orgID, envID, apiID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SLA tier", "Could not create SLA tier: "+err.Error())
		return
	}

	flatDiags := r.flattenTier(ctx, tier, &data, orgID, envID)
	resp.Diagnostics.Append(flatDiags...)
	tflog.Trace(ctx, "created SLA tier", map[string]interface{}{"id": tier.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SLATierResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SLATierResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	apiID, err := strconv.Atoi(data.APIInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid api_instance_id", "Must be a numeric ID: "+data.APIInstanceID.ValueString())
		return
	}
	tierID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid tier ID", "Could not parse tier ID: "+data.ID.ValueString())
		return
	}

	tier, err := r.client.GetSLATier(ctx, orgID, envID, apiID, tierID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading SLA tier", "Could not read SLA tier: "+err.Error())
		return
	}

	flatDiags := r.flattenTier(ctx, tier, &data, orgID, envID)
	resp.Diagnostics.Append(flatDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SLATierResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SLATierResourceModel
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
	apiID, err := strconv.Atoi(state.APIInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid api_instance_id", "Must be a numeric ID: "+state.APIInstanceID.ValueString())
		return
	}
	tierID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid tier ID", "Could not parse tier ID: "+state.ID.ValueString())
		return
	}

	limits, diags := r.expandLimits(ctx, plan.Limits)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	autoApprove := plan.AutoApprove.ValueBool()
	updateReq := &apimanagement.UpdateSLATierRequest{
		Name:        plan.Name.ValueString(),
		AutoApprove: &autoApprove,
		Limits:      limits,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		updateReq.Status = plan.Status.ValueString()
	}

	tier, err := r.client.UpdateSLATier(ctx, orgID, envID, apiID, tierID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating SLA tier", "Could not update SLA tier: "+err.Error())
		return
	}

	flatDiags := r.flattenTier(ctx, tier, &plan, orgID, envID)
	resp.Diagnostics.Append(flatDiags...)
	tflog.Trace(ctx, "updated SLA tier", map[string]interface{}{"id": tier.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SLATierResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SLATierResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	apiID, err := strconv.Atoi(data.APIInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid api_instance_id", "Must be a numeric ID: "+data.APIInstanceID.ValueString())
		return
	}
	tierID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid tier ID", "Could not parse tier ID: "+data.ID.ValueString())
		return
	}

	if err := r.client.DeleteSLATier(ctx, orgID, envID, apiID, tierID); err != nil {
		resp.Diagnostics.AddError("Error deleting SLA tier", "Could not delete SLA tier: "+err.Error())
		return
	}

	tflog.Trace(ctx, "deleted SLA tier", map[string]interface{}{"id": tierID})
}

func (r *SLATierResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/api_instance_id/tier_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("api_instance_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// --- Helpers ---

func (r *SLATierResource) expandLimits(ctx context.Context, limitsList types.List) ([]apimanagement.SLALimit, diag.Diagnostics) {
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

func (r *SLATierResource) flattenTier(ctx context.Context, tier *apimanagement.SLATier, data *SLATierResourceModel, orgID, envID string) diag.Diagnostics {
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

	limitModels := make([]SLALimitModel, len(tier.Limits))
	for i, l := range tier.Limits {
		limitModels[i] = SLALimitModel{
			TimePeriodInMilliseconds: types.Int64Value(int64(l.TimePeriodInMilliseconds)),
			MaximumRequests:          types.Int64Value(int64(l.MaximumRequests)),
			Visible:                  types.BoolValue(l.Visible),
		}
	}

	limitListVal, diags := types.ListValueFrom(ctx, slaLimitObjectType, limitModels)
	data.Limits = limitListVal
	return diags
}
