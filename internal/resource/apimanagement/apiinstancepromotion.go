package apimanagement

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &APIInstancePromotionResource{}
	_ resource.ResourceWithConfigure   = &APIInstancePromotionResource{}
	_ resource.ResourceWithImportState = &APIInstancePromotionResource{}
)

type APIInstancePromotionResource struct {
	client *apimanagement.APIInstanceClient
}

type APIInstancePromotionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	SourceApiID    types.Int64  `tfsdk:"source_api_id"`
	InstanceLabel  types.String `tfsdk:"instance_label"`

	IncludeAlerts   types.Bool `tfsdk:"include_alerts"`
	IncludePolicies types.Bool `tfsdk:"include_policies"`
	IncludeTiers    types.Bool `tfsdk:"include_tiers"`

	// Computed from the API response
	AssetID        types.String `tfsdk:"asset_id"`
	AssetVersion   types.String `tfsdk:"asset_version"`
	ProductVersion types.String `tfsdk:"product_version"`
	GroupID        types.String `tfsdk:"group_id"`
	Technology     types.String `tfsdk:"technology"`
	Status         types.String `tfsdk:"status"`
	AutodiscoveryInstanceName types.String `tfsdk:"autodiscovery_instance_name"`
}

func NewAPIInstancePromotionResource() resource.Resource {
	return &APIInstancePromotionResource{}
}

func (r *APIInstancePromotionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_instance_promotion"
}

func (r *APIInstancePromotionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Promotes an API instance from one environment to another. " +
			"This copies the API definition (and optionally its policies, SLA tiers, and alerts) " +
			"into the target environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the promoted API instance in the target environment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: "The target environment ID where the API instance will be promoted to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_api_id": schema.Int64Attribute{
				Description: "The numeric ID of the source API instance to promote from.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"instance_label": schema.StringAttribute{
				Description: "Optional label for the promoted API instance.",
				Optional:    true,
			},
			"include_alerts": schema.BoolAttribute{
				Description: "Whether to copy alerts from the source API instance. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"include_policies": schema.BoolAttribute{
				Description: "Whether to copy policies from the source API instance. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"include_tiers": schema.BoolAttribute{
				Description: "Whether to copy SLA tiers from the source API instance. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"asset_id": schema.StringAttribute{
				Description: "The Exchange asset ID of the promoted API instance.",
				Computed:    true,
			},
			"asset_version": schema.StringAttribute{
				Description: "The Exchange asset version of the promoted API instance.",
				Computed:    true,
			},
			"product_version": schema.StringAttribute{
				Description: "The product version of the promoted API instance.",
				Computed:    true,
			},
			"group_id": schema.StringAttribute{
				Description: "The Exchange group (organization) ID of the promoted API instance.",
				Computed:    true,
			},
			"technology": schema.StringAttribute{
				Description: "The gateway technology of the promoted API instance.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the promoted API instance.",
				Computed:    true,
			},
			"autodiscovery_instance_name": schema.StringAttribute{
				Description: "The autodiscovery instance name of the promoted API instance.",
				Computed:    true,
			},
		},
	}
}

func (r *APIInstancePromotionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T.", req.ProviderData),
		)
		return
	}

	apiClient, err := apimanagement.NewAPIInstanceClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create API Instance Client",
			"An unexpected error occurred when creating the API Instance client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = apiClient
}

// --- CRUD ---

func (r *APIInstancePromotionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIInstancePromotionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	promoteReq := &apimanagement.PromoteAPIInstanceRequest{
		Promote: apimanagement.PromoteConfig{
			OriginApiID: int(data.SourceApiID.ValueInt64()),
		},
	}

	if !data.InstanceLabel.IsNull() && !data.InstanceLabel.IsUnknown() {
		label := data.InstanceLabel.ValueString()
		promoteReq.InstanceLabel = &label
	}

	if data.IncludeAlerts.ValueBool() {
		promoteReq.Promote.Alerts = &apimanagement.PromoteEntities{AllEntities: true}
	}
	if data.IncludePolicies.ValueBool() {
		promoteReq.Promote.Policies = &apimanagement.PromoteEntities{AllEntities: true}
	}
	if data.IncludeTiers.ValueBool() {
		promoteReq.Promote.Tiers = &apimanagement.PromoteEntities{AllEntities: true}
	}

	instance, err := r.client.PromoteAPIInstance(ctx, orgID, envID, promoteReq)
	if err != nil {
		resp.Diagnostics.AddError("Error promoting API instance", "Could not promote API instance: "+err.Error())
		return
	}

	r.flattenPromotion(instance, &data, orgID)
	tflog.Trace(ctx, "promoted API instance", map[string]interface{}{"id": instance.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIInstancePromotionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIInstancePromotionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	apiID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Instance ID", "Could not parse promoted API instance ID: "+data.ID.ValueString())
		return
	}

	instance, err := r.client.GetAPIInstance(ctx, orgID, envID, apiID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading promoted API instance",
			"Could not read promoted API instance ID "+data.ID.ValueString()+": "+err.Error())
		return
	}

	sourceApiID := data.SourceApiID
	includeAlerts := data.IncludeAlerts
	includePolicies := data.IncludePolicies
	includeTiers := data.IncludeTiers
	instanceLabel := data.InstanceLabel

	r.flattenPromotion(instance, &data, orgID)

	data.SourceApiID = sourceApiID
	data.IncludeAlerts = includeAlerts
	data.IncludePolicies = includePolicies
	data.IncludeTiers = includeTiers
	data.InstanceLabel = instanceLabel

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIInstancePromotionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state APIInstancePromotionResourceModel
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

	apiID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Instance ID", "Could not parse promoted API instance ID: "+state.ID.ValueString())
		return
	}

	updateReq := &apimanagement.UpdateAPIInstanceRequest{}
	if !plan.InstanceLabel.IsNull() && !plan.InstanceLabel.IsUnknown() {
		il := plan.InstanceLabel.ValueString()
		updateReq.InstanceLabel = &il
	}

	instance, err := r.client.UpdateAPIInstance(ctx, orgID, envID, apiID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating promoted API instance", "Could not update promoted API instance: "+err.Error())
		return
	}

	r.flattenPromotion(instance, &plan, orgID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APIInstancePromotionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIInstancePromotionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	apiID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Instance ID", "Could not parse promoted API instance ID: "+data.ID.ValueString())
		return
	}

	if err := r.client.DeleteAPIInstance(ctx, orgID, envID, apiID); err != nil {
		resp.Diagnostics.AddError("Error deleting promoted API instance", "Could not delete promoted API instance: "+err.Error())
	}
}

func (r *APIInstancePromotionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Helpers ---

func (r *APIInstancePromotionResource) flattenPromotion(inst *apimanagement.APIInstance, data *APIInstancePromotionResourceModel, orgID string) {
	data.ID = types.StringValue(strconv.Itoa(inst.ID))
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(inst.EnvironmentID)
	data.AssetID = types.StringValue(inst.AssetID)
	data.AssetVersion = types.StringValue(inst.AssetVersion)
	data.ProductVersion = types.StringValue(inst.ProductVersion)
	data.GroupID = types.StringValue(inst.GroupID)
	data.Technology = types.StringValue(inst.Technology)
	data.Status = types.StringValue(inst.Status)

	if inst.AutodiscoveryInstanceName != "" {
		data.AutodiscoveryInstanceName = types.StringValue(inst.AutodiscoveryInstanceName)
	} else {
		data.AutodiscoveryInstanceName = types.StringNull()
	}

	if inst.InstanceLabel != "" {
		data.InstanceLabel = types.StringValue(inst.InstanceLabel)
	}
}
