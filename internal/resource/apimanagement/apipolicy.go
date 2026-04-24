package apimanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &APIPolicyResource{}
	_ resource.ResourceWithConfigure   = &APIPolicyResource{}
	_ resource.ResourceWithImportState = &APIPolicyResource{}
)

type APIPolicyResource struct {
	client *apimanagement.APIPolicyClient
}

// --- Terraform State Model ---

type APIPolicyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	OrganizationID    types.String `tfsdk:"organization_id"`
	EnvironmentID     types.String `tfsdk:"environment_id"`
	APIInstanceID     types.String `tfsdk:"api_instance_id"`
	PolicyType        types.String `tfsdk:"policy_type"`
	GroupID           types.String `tfsdk:"group_id"`
	AssetID           types.String `tfsdk:"asset_id"`
	AssetVersion      types.String `tfsdk:"asset_version"`
	Label             types.String `tfsdk:"label"`
	ConfigurationData types.String `tfsdk:"configuration_data"`
	Order             types.Int64  `tfsdk:"order"`
	Disabled          types.Bool   `tfsdk:"disabled"`
	PolicyTemplateID  types.String `tfsdk:"policy_template_id"`
}

func NewAPIPolicyResource() resource.Resource {
	return &APIPolicyResource{}
}

func (r *APIPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_policy"
}

func (r *APIPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a policy applied to an API instance in Anypoint API Manager. " +
			"Use policy_type for known policies (auto-resolves group_id, asset_id, and default version), " +
			"or provide group_id + asset_id + asset_version directly for custom policies.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the applied policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID. Defaults to the provider's org ID if omitted.",
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
				Description: "Numeric ID of the API instance this policy is applied to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_type": schema.StringAttribute{
				Description: "Known policy type name (e.g. 'rate-limiting', 'cors', 'jwt-validation'). " +
					"When set, group_id, asset_id, and asset_version are auto-resolved from the built-in registry. " +
					"You can still override asset_version to pin a specific version. " +
					"For custom policies not in the registry, omit this and set group_id + asset_id + asset_version directly.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "Exchange group ID for the policy asset. Auto-resolved when policy_type is set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"asset_id": schema.StringAttribute{
				Description: "Exchange asset ID that identifies the policy type. Auto-resolved when policy_type is set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"asset_version": schema.StringAttribute{
				Description: "Version of the policy asset from Exchange. Auto-resolved to default when policy_type is set, but can be overridden.",
				Optional:    true,
				Computed:    true,
			},
			"label": schema.StringAttribute{
				Description: "A human-readable label for this policy instance.",
				Optional:    true,
			},
			"configuration_data": schema.StringAttribute{
				Description: "Policy configuration as a JSON string. Use jsonencode() to set this. " +
					"Fields vary by policy type; the provider validates known policies at plan time.",
				Required: true,
			},
			"order": schema.Int64Attribute{
				Description: "Execution order of the policy. Lower numbers execute first.",
				Optional:    true,
				Computed:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Whether the policy is disabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"policy_template_id": schema.StringAttribute{
				Description: "Policy template ID assigned by the server.",
				Computed:    true,
			},
		},
	}
}

func (r *APIPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	policyClient, err := apimanagement.NewAPIPolicyClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create API Policy Client",
			"An unexpected error occurred when creating the API Policy client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = policyClient
}

// --- Resolution ---

// resolvePolicyIdentifiers resolves group_id, asset_id, and asset_version from policy_type
// or validates that they are explicitly provided. Returns the resolved values.
func (r *APIPolicyResource) resolvePolicyIdentifiers(data *APIPolicyResourceModel) (groupID, assetID, assetVersion string, err error) {
	policyType := data.PolicyType.ValueString()

	if policyType != "" {
		info, ok := apimanagement.LookupPolicy(policyType)
		if !ok {
			return "", "", "", fmt.Errorf("unknown policy_type %q — use group_id + asset_id + asset_version for custom policies", policyType)
		}

		groupID = info.GroupID
		assetID = info.AssetID
		assetVersion = info.DefaultVersion

		// Allow user to override group_id, asset_id, asset_version
		if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() && data.GroupID.ValueString() != "" {
			groupID = data.GroupID.ValueString()
		}
		if !data.AssetID.IsNull() && !data.AssetID.IsUnknown() && data.AssetID.ValueString() != "" {
			assetID = data.AssetID.ValueString()
		}
		if !data.AssetVersion.IsNull() && !data.AssetVersion.IsUnknown() && data.AssetVersion.ValueString() != "" {
			assetVersion = data.AssetVersion.ValueString()
		}

		return groupID, assetID, assetVersion, nil
	}

	// No policy_type — require explicit values
	groupID = data.GroupID.ValueString()
	assetID = data.AssetID.ValueString()
	assetVersion = data.AssetVersion.ValueString()

	if groupID == "" || assetID == "" || assetVersion == "" {
		return "", "", "", fmt.Errorf("either policy_type must be set, or all of group_id, asset_id, and asset_version must be provided")
	}

	return groupID, assetID, assetVersion, nil
}

// --- Validation ---

func (r *APIPolicyResource) validateConfigurationData(assetID, configJSON string) []string {
	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configData); err != nil {
		return []string{fmt.Sprintf("configuration_data is not valid JSON: %s", err.Error())}
	}
	return apimanagement.ValidatePolicyConfiguration(assetID, configData)
}

// --- CRUD ---

func (r *APIPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, assetID, assetVersion, err := r.resolvePolicyIdentifiers(&data)
	if err != nil {
		resp.Diagnostics.AddError("Policy resolution error", err.Error())
		return
	}

	// Store resolved values back into the model
	data.GroupID = types.StringValue(groupID)
	data.AssetID = types.StringValue(assetID)
	data.AssetVersion = types.StringValue(assetVersion)

	configJSON := data.ConfigurationData.ValueString()
	if validationErrs := r.validateConfigurationData(assetID, configJSON); len(validationErrs) > 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid configuration for policy %q", assetID),
			strings.Join(validationErrs, "\n"),
		)
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configData); err != nil {
		resp.Diagnostics.AddError("Invalid configuration_data", "Could not parse JSON: "+err.Error())
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

	createReq := &apimanagement.CreateAPIPolicyRequest{
		ConfigurationData: configData,
		APIVersionID:      apiID,
		GroupID:           groupID,
		AssetID:           assetID,
		AssetVersion:      assetVersion,
		PointcutData:      nil,
	}

	if !data.Label.IsNull() && !data.Label.IsUnknown() {
		createReq.Label = data.Label.ValueString()
	}
	if !data.Order.IsNull() && !data.Order.IsUnknown() {
		o := int(data.Order.ValueInt64())
		createReq.Order = &o
	}
	if !data.Disabled.IsNull() && !data.Disabled.IsUnknown() {
		d := data.Disabled.ValueBool()
		createReq.Disabled = &d
	}

	policy, err := r.client.CreateAPIPolicy(ctx, orgID, envID, apiID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating policy", "Could not create policy: "+err.Error())
		return
	}

	r.flattenPolicy(policy, &data, orgID, envID)
	tflog.Trace(ctx, "created API policy", map[string]interface{}{"id": policy.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIPolicyResourceModel
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
	policyID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid policy ID", "Could not parse policy ID: "+data.ID.ValueString())
		return
	}

	policy, err := r.client.GetAPIPolicy(ctx, orgID, envID, apiID, policyID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading policy", "Could not read policy: "+err.Error())
		return
	}

	r.flattenPolicy(policy, &data, orgID, envID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state APIPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, assetID, assetVersion, err := r.resolvePolicyIdentifiers(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Policy resolution error", err.Error())
		return
	}

	configJSON := plan.ConfigurationData.ValueString()
	if validationErrs := r.validateConfigurationData(assetID, configJSON); len(validationErrs) > 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid configuration for policy %q", assetID),
			strings.Join(validationErrs, "\n"),
		)
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configData); err != nil {
		resp.Diagnostics.AddError("Invalid configuration_data", "Could not parse JSON: "+err.Error())
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
	policyID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid policy ID", "Could not parse policy ID: "+state.ID.ValueString())
		return
	}

	updateReq := &apimanagement.UpdateAPIPolicyRequest{
		ConfigurationData: configData,
		AssetVersion:      assetVersion,
	}

	if !plan.Label.IsNull() && !plan.Label.IsUnknown() {
		updateReq.Label = plan.Label.ValueString()
	}
	if !plan.Order.IsNull() && !plan.Order.IsUnknown() {
		o := int(plan.Order.ValueInt64())
		updateReq.Order = &o
	}
	if !plan.Disabled.IsNull() && !plan.Disabled.IsUnknown() {
		d := plan.Disabled.ValueBool()
		updateReq.Disabled = &d
	}

	policy, err := r.client.UpdateAPIPolicy(ctx, orgID, envID, apiID, policyID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating policy", "Could not update policy: "+err.Error())
		return
	}

	r.flattenPolicy(policy, &plan, orgID, envID)
	tflog.Trace(ctx, "updated API policy", map[string]interface{}{"id": policy.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APIPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIPolicyResourceModel
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
	policyID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid policy ID", "Could not parse policy ID: "+data.ID.ValueString())
		return
	}

	if err := r.client.DeleteAPIPolicy(ctx, orgID, envID, apiID, policyID); err != nil {
		resp.Diagnostics.AddError("Error deleting policy", "Could not delete policy: "+err.Error())
		return
	}

	tflog.Trace(ctx, "deleted API policy", map[string]interface{}{"id": policyID})
}

func (r *APIPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/api_instance_id/policy_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("api_instance_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// --- Flatten ---

func (r *APIPolicyResource) flattenPolicy(policy *apimanagement.APIPolicy, data *APIPolicyResourceModel, orgID, envID string) {
	data.ID = types.StringValue(strconv.Itoa(policy.ID))
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)

	if policy.APIID != 0 {
		data.APIInstanceID = types.StringValue(strconv.Itoa(policy.APIID))
	}

	data.GroupID = types.StringValue(policy.GroupID)
	data.AssetID = types.StringValue(policy.AssetID)
	data.AssetVersion = types.StringValue(policy.AssetVersion)
	data.Order = types.Int64Value(int64(policy.Order))
	data.Disabled = types.BoolValue(policy.Disabled)
	data.PolicyTemplateID = types.StringValue(policy.PolicyTemplateID)

	// Reverse-lookup policy_type from the asset_id if it matches a known policy
	if _, ok := apimanagement.LookupPolicy(policy.AssetID); ok {
		data.PolicyType = types.StringValue(policy.AssetID)
	} else if data.PolicyType.IsNull() || data.PolicyType.IsUnknown() {
		data.PolicyType = types.StringValue("")
	}

	if policy.Label != "" {
		data.Label = types.StringValue(policy.Label)
	}

	if policy.ConfigurationData != nil {
		cfgJSON, err := json.Marshal(policy.ConfigurationData)
		if err == nil {
			data.ConfigurationData = types.StringValue(string(cfgJSON))
		}
	}
}
