package apimanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &KnownPolicyResource{}
	_ resource.ResourceWithConfigure   = &KnownPolicyResource{}
	_ resource.ResourceWithImportState = &KnownPolicyResource{}
)

type KnownPolicyResource struct {
	client     *apimanagement.APIPolicyClient
	policyInfo apimanagement.PolicyInfo
	typeSuffix string
}

type KnownPolicyResourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	APIInstanceID    types.String `tfsdk:"api_instance_id"`
	Label            types.String `tfsdk:"label"`
	Configuration    types.Object `tfsdk:"configuration"`
	Order            types.Int64  `tfsdk:"order"`
	Disabled         types.Bool   `tfsdk:"disabled"`
	PolicyTemplateID types.String `tfsdk:"policy_template_id"`
	AssetVersion     types.String `tfsdk:"asset_version"`
	UpstreamIDs      types.List   `tfsdk:"upstream_ids"`
}

func NewKnownPolicyResourceFunc(policyType string) func() resource.Resource {
	info, ok := apimanagement.LookupPolicy(policyType)
	if !ok {
		panic(fmt.Sprintf("BUG: unknown policy type %q passed to NewKnownPolicyResourceFunc", policyType))
	}
	suffix := strings.ReplaceAll(policyType, "-", "_")
	return func() resource.Resource {
		return &KnownPolicyResource{
			policyInfo: info,
			typeSuffix: suffix,
		}
	}
}

func (r *KnownPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_policy_" + r.typeSuffix
}

// --- Schema Generation ---

// configAttrTypes returns the attr.Type map for the configuration object,
// derived from KnownPolicySchemas. This is used by both the schema and
// the expand/flatten logic to ensure consistency.
func (r *KnownPolicyResource) configAttrTypes() map[string]attr.Type {
	policySchema, ok := apimanagement.KnownPolicySchemas[r.policyInfo.AssetID]
	if !ok {
		return map[string]attr.Type{}
	}
	attrTypes := make(map[string]attr.Type, len(policySchema))
	for camelName, field := range policySchema {
		snakeName := apimanagement.CamelToSnake(camelName)
		attrTypes[snakeName] = fieldSchemaType(field.Type)
	}
	return attrTypes
}

func fieldSchemaType(t string) attr.Type {
	switch t {
	case "string":
		return types.StringType
	case "int":
		return types.NumberType
	case "bool":
		return types.BoolType
	case "string_array":
		return types.ListType{ElemType: types.StringType}
	default:
		return types.DynamicType
	}
}

func generateConfigurationSchema(assetID string) schema.SingleNestedAttribute {
	policySchema, ok := apimanagement.KnownPolicySchemas[assetID]
	if !ok {
		return schema.SingleNestedAttribute{
			Description: "Policy configuration.",
			Optional:    true,
			Attributes:  map[string]schema.Attribute{},
		}
	}

	attrs := make(map[string]schema.Attribute, len(policySchema))
	for camelName, field := range policySchema {
		snakeName := apimanagement.CamelToSnake(camelName)
		switch field.Type {
		case "string":
			attrs[snakeName] = schema.StringAttribute{
				Description: fmt.Sprintf("Policy field '%s'.", camelName),
				Required:    field.Required,
				Optional:    !field.Required,
			}
		case "int":
			var numValidators []validator.Number
			if field.Min != nil {
				numValidators = append(numValidators, numberAtLeastValidator{min: big.NewFloat(*field.Min)})
			}
			if field.Max != nil {
				numValidators = append(numValidators, numberAtMostValidator{max: big.NewFloat(*field.Max)})
			}
			attrs[snakeName] = schema.NumberAttribute{
				Description: fmt.Sprintf("Policy field '%s'.", camelName),
				Required:    field.Required,
				Optional:    !field.Required,
				Validators:  numValidators,
			}
		case "bool":
			attrs[snakeName] = schema.BoolAttribute{
				Description: fmt.Sprintf("Policy field '%s'.", camelName),
				Required:    field.Required,
				Optional:    !field.Required,
			}
		case "string_array":
			attrs[snakeName] = schema.ListAttribute{
				Description: fmt.Sprintf("Policy field '%s'. Must be a list of strings.", camelName),
				Required:    field.Required,
				Optional:    !field.Required,
				ElementType: types.StringType,
			}
		default:
			attrs[snakeName] = schema.DynamicAttribute{
				Description: fmt.Sprintf("Policy field '%s'. Accepts lists, maps, or any HCL value.", camelName),
				Required:    field.Required,
				Optional:    !field.Required,
			}
		}
	}

	return schema.SingleNestedAttribute{
		Description: "Policy configuration block with typed fields.",
		Required:    true,
		Attributes:  attrs,
	}
}

func (r *KnownPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	desc := fmt.Sprintf(
		"Manages the %s policy on an API instance. "+
			"Exchange coordinates (groupId, assetId) are handled automatically.",
		r.policyInfo.AssetID,
	)
	if len(r.policyInfo.SupportedTechnologies) > 0 {
		desc += fmt.Sprintf(
			" NOTE: This policy is only supported on %s API instances.",
			strings.Join(r.policyInfo.SupportedTechnologies, ", "),
		)
	}

	resp.Schema = schema.Schema{
		Description: desc,
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
			"label": schema.StringAttribute{
				Description: "A human-readable label for this policy instance.",
				Optional:    true,
			},
			"configuration": generateConfigurationSchema(r.policyInfo.AssetID),
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_version": schema.StringAttribute{
				Description: fmt.Sprintf(
					"Version of the policy asset. Defaults to %s. Override to pin a specific version.",
					r.policyInfo.DefaultVersion,
				),
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"upstream_ids": schema.ListAttribute{
				Description: "List of upstream IDs this policy applies to. Required for outbound policies.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *KnownPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// --- Expand / Flatten ---

// expandConfiguration converts the typed HCL configuration object into the
// map[string]interface{} that the API expects, mapping snake_case → camelCase.
// Uses the schema's original camelCase keys to avoid lossy round-trips (e.g. "URL").
func (r *KnownPolicyResource) expandConfiguration(_ context.Context, obj types.Object) map[string]interface{} {
	if obj.IsNull() || obj.IsUnknown() {
		return map[string]interface{}{}
	}

	result := make(map[string]interface{})
	attrs := obj.Attributes()

	policySchema := apimanagement.KnownPolicySchemas[r.policyInfo.AssetID]

	snakeToCamel := make(map[string]string, len(policySchema))
	fieldTypes := make(map[string]string, len(policySchema))
	for camelName, field := range policySchema {
		snake := apimanagement.CamelToSnake(camelName)
		snakeToCamel[snake] = camelName
		fieldTypes[snake] = field.Type
	}

	for snakeName, val := range attrs {
		if val.IsNull() || val.IsUnknown() {
			continue
		}
		camelName, ok := snakeToCamel[snakeName]
		if !ok {
			camelName = apimanagement.SnakeToCamel(snakeName)
		}

		switch fieldTypes[snakeName] {
		case "string":
			if sv, ok := val.(basetypes.StringValue); ok {
				result[camelName] = sv.ValueString()
			}
		case "int":
			if nv, ok := val.(basetypes.NumberValue); ok {
				f, _ := nv.ValueBigFloat().Float64()
				result[camelName] = int(f)
			}
		case "bool":
			if bv, ok := val.(basetypes.BoolValue); ok {
				result[camelName] = bv.ValueBool()
			}
		case "string_array":
			if lv, ok := val.(basetypes.ListValue); ok {
				elems := lv.Elements()
				strs := make([]string, 0, len(elems))
				for _, e := range elems {
					if sv, ok := e.(basetypes.StringValue); ok {
						strs = append(strs, sv.ValueString())
					}
				}
				result[camelName] = strs
			}
		default:
			if dv, ok := val.(basetypes.DynamicValue); ok {
				result[camelName] = dynamicToNative(dv.UnderlyingValue())
			}
		}
	}

	return result
}

// dynamicToNative recursively converts a Terraform attr.Value to a native Go value.
// Object/map keys are converted from snake_case to camelCase so the API receives
// its expected field names (users write snake_case in HCL).
func dynamicToNative(v attr.Value) interface{} {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return nil
	}
	switch tv := v.(type) {
	case basetypes.StringValue:
		return tv.ValueString()
	case basetypes.NumberValue:
		f, _ := tv.ValueBigFloat().Float64()
		if f == float64(int(f)) {
			return int(f)
		}
		return f
	case basetypes.BoolValue:
		return tv.ValueBool()
	case basetypes.ListValue:
		elems := tv.Elements()
		result := make([]interface{}, len(elems))
		for i, e := range elems {
			result[i] = dynamicToNative(e)
		}
		return result
	case basetypes.TupleValue:
		elems := tv.Elements()
		result := make([]interface{}, len(elems))
		for i, e := range elems {
			result[i] = dynamicToNative(e)
		}
		return result
	case basetypes.MapValue:
		elems := tv.Elements()
		result := make(map[string]interface{}, len(elems))
		for k, e := range elems {
			result[apimanagement.SnakeToCamel(k)] = dynamicToNative(e)
		}
		return result
	case basetypes.ObjectValue:
		attrs := tv.Attributes()
		result := make(map[string]interface{}, len(attrs))
		for k, a := range attrs {
			result[apimanagement.SnakeToCamel(k)] = dynamicToNative(a)
		}
		return result
	case basetypes.DynamicValue:
		return dynamicToNative(tv.UnderlyingValue())
	default:
		return nil
	}
}

// flattenConfiguration converts the API's map[string]interface{} response back
// into a types.Object matching the schema, mapping camelCase → snake_case.
func (r *KnownPolicyResource) flattenConfiguration(_ context.Context, configData map[string]interface{}) (types.Object, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	attrTypes := r.configAttrTypes()

	if len(attrTypes) == 0 {
		obj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{})
		allDiags.Append(diags...)
		return obj, allDiags
	}

	attrValues := make(map[string]attr.Value, len(attrTypes))

	// Initialize all attrs to null
	for name, t := range attrTypes {
		attrValues[name] = nullForType(t)
	}

	policySchema := apimanagement.KnownPolicySchemas[r.policyInfo.AssetID]

	for camelName, rawVal := range configData {
		snakeName := apimanagement.CamelToSnake(camelName)
		if _, exists := attrTypes[snakeName]; !exists {
			continue
		}

		fieldType := ""
		if f, ok := policySchema[camelName]; ok {
			fieldType = f.Type
		}

		switch fieldType {
		case "string":
			if s, ok := rawVal.(string); ok {
				attrValues[snakeName] = types.StringValue(s)
			}
		case "int":
			switch v := rawVal.(type) {
			case float64:
				attrValues[snakeName] = types.NumberValue(big.NewFloat(v))
			case int:
				attrValues[snakeName] = types.NumberValue(new(big.Float).SetInt64(int64(v)))
			case json.Number:
				f, _ := v.Float64()
				attrValues[snakeName] = types.NumberValue(big.NewFloat(f))
			}
		case "bool":
			if b, ok := rawVal.(bool); ok {
				attrValues[snakeName] = types.BoolValue(b)
			}
		case "string_array":
			var strs []attr.Value
			if arr, ok := rawVal.([]interface{}); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						strs = append(strs, types.StringValue(s))
					}
				}
			}
			if strs == nil {
				strs = []attr.Value{}
			}
			lv, _ := types.ListValue(types.StringType, strs)
			attrValues[snakeName] = lv
		default:
			tfVal := nativeToDynamic(rawVal)
			attrValues[snakeName] = types.DynamicValue(tfVal)
		}
	}

	obj, diags := types.ObjectValue(attrTypes, attrValues)
	allDiags.Append(diags...)
	return obj, allDiags
}

func nullForType(t attr.Type) attr.Value {
	switch t {
	case types.StringType:
		return types.StringNull()
	case types.NumberType:
		return types.NumberNull()
	case types.BoolType:
		return types.BoolNull()
	default:
		// Handle typed list types (e.g. ListType{ElemType: StringType} for string_array).
		if lt, ok := t.(types.ListType); ok {
			return types.ListNull(lt.ElemType)
		}
		return types.DynamicNull()
	}
}

// nativeToDynamic converts a native Go value (from JSON) into a Terraform attr.Value
// suitable for wrapping in types.DynamicValue.
// Object/map keys are converted from the API's camelCase to snake_case so they
// match the HCL convention users write. Top-level schema keys are handled by
// flattenConfiguration.
func nativeToDynamic(v interface{}) attr.Value {
	if v == nil {
		return types.StringNull()
	}
	switch val := v.(type) {
	case string:
		return types.StringValue(val)
	case bool:
		return types.BoolValue(val)
	case float64:
		return types.NumberValue(big.NewFloat(val))
	case int:
		return types.NumberValue(new(big.Float).SetInt64(int64(val)))
	case json.Number:
		f, _ := val.Float64()
		return types.NumberValue(big.NewFloat(f))
	case []interface{}:
		if len(val) == 0 {
			tv, _ := types.TupleValue([]attr.Type{}, []attr.Value{})
			return tv
		}
		elems := make([]attr.Value, len(val))
		elemTypes := make([]attr.Type, len(val))
		for i, e := range val {
			ev := nativeToDynamic(e)
			elems[i] = ev
			elemTypes[i] = ev.Type(context.Background())
		}
		tv, _ := types.TupleValue(elemTypes, elems)
		return tv
	case map[string]interface{}:
		if len(val) == 0 {
			return types.ObjectNull(map[string]attr.Type{})
		}
		attrTypes := make(map[string]attr.Type, len(val))
		attrValues := make(map[string]attr.Value, len(val))
		for k, e := range val {
			snakeKey := apimanagement.CamelToSnake(k)
			ev := nativeToDynamic(e)
			attrTypes[snakeKey] = ev.Type(context.Background())
			attrValues[snakeKey] = ev
		}
		ov, _ := types.ObjectValue(attrTypes, attrValues)
		return ov
	default:
		return types.StringValue(fmt.Sprintf("%v", v))
	}
}

// --- CRUD ---

func (r *KnownPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KnownPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assetVersion := r.resolveVersion(&data)
	configData := r.expandConfiguration(ctx, data.Configuration)

	if errs := apimanagement.ValidatePolicyConfiguration(r.policyInfo.AssetID, configData); len(errs) > 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid configuration for policy %q", r.policyInfo.AssetID),
			strings.Join(errs, "\n"),
		)
		return
	}

	orgID, envID, apiID, parseErr := r.parseIDs(&data)
	if parseErr != nil {
		resp.Diagnostics.AddError("Invalid IDs", parseErr.Error())
		return
	}

	var policy *apimanagement.APIPolicy
	var err error
	if r.policyInfo.OutboundPolicy {
		// Outbound policies (xapi/v1) must NOT include pointcutData, order, or disabled.
		outboundReq := &apimanagement.CreateOutboundAPIPolicyRequest{
			ConfigurationData: configData,
			GroupID:           r.policyInfo.GroupID,
			AssetID:           r.policyInfo.AssetID,
			AssetVersion:      assetVersion,
		}
		if !data.Label.IsNull() && !data.Label.IsUnknown() {
			outboundReq.Label = data.Label.ValueString()
		}
		if !data.UpstreamIDs.IsNull() && !data.UpstreamIDs.IsUnknown() {
			var ids []string
			data.UpstreamIDs.ElementsAs(ctx, &ids, false)
			outboundReq.UpstreamIDs = ids
		}
		policy, err = r.client.CreateOutboundAPIPolicy(ctx, orgID, envID, apiID, outboundReq)
	} else {
		createReq := &apimanagement.CreateAPIPolicyRequest{
			ConfigurationData: configData,
			APIVersionID:      apiID,
			GroupID:           r.policyInfo.GroupID,
			AssetID:           r.policyInfo.AssetID,
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
		if !data.UpstreamIDs.IsNull() && !data.UpstreamIDs.IsUnknown() {
			var ids []string
			data.UpstreamIDs.ElementsAs(ctx, &ids, false)
			createReq.UpstreamIDs = ids
		}
		policy, err = r.client.CreateAPIPolicy(ctx, orgID, envID, apiID, createReq)
	}
	if err != nil {
		errMsg := "Could not create policy: " + err.Error()
		if strings.Contains(err.Error(), "does not have an implementation") &&
			len(r.policyInfo.SupportedTechnologies) > 0 {
			errMsg += fmt.Sprintf(
				"\n\nHint: The %q policy only supports %s API instances. "+
					"Ensure the target API instance uses a supported technology.",
				r.policyInfo.AssetID,
				strings.Join(r.policyInfo.SupportedTechnologies, ", "),
			)
		}
		resp.Diagnostics.AddError("Error creating policy", errMsg)
		return
	}

	plannedConfig := data.Configuration
	r.flatten(ctx, policy, &data, orgID, envID)
	// Preserve the planned configuration to avoid type mismatches.
	// DynamicAttribute values (arrays, objects) may produce different tuple/object
	// types when round-tripped through the API, causing "tuple required" errors.
	if !plannedConfig.IsNull() && !plannedConfig.IsUnknown() {
		data.Configuration = plannedConfig
	}
	tflog.Trace(ctx, "created known API policy", map[string]interface{}{"id": policy.ID, "type": r.policyInfo.AssetID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KnownPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KnownPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID, envID, apiID, err := r.parseIDs(&data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid IDs", err.Error())
		return
	}
	policyID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid policy ID", "Could not parse policy ID: "+data.ID.ValueString())
		return
	}

	var policy *apimanagement.APIPolicy
	if r.policyInfo.OutboundPolicy {
		policy, err = r.client.GetOutboundAPIPolicy(ctx, orgID, envID, apiID, policyID)
	} else {
		policy, err = r.client.GetAPIPolicy(ctx, orgID, envID, apiID, policyID)
	}
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading policy", "Could not read policy: "+err.Error())
		return
	}

	// Preserve upstream_ids from state — the API echoes them but we own the source of truth.
	existingUpstreamIDs := data.UpstreamIDs

	// Preserve the existing configuration from state before flattening.
	// The API never returns write-only fields (passwords, tokens, secrets), so
	// flattenConfiguration would null them out, causing a perpetual diff where
	// Terraform sees config has a value but state has null.
	existingConfig := data.Configuration

	r.flatten(ctx, policy, &data, orgID, envID)

	// Re-merge: for any config field that the API didn't return (null in new state
	// but present in the prior state), restore the prior state value.
	if !existingConfig.IsNull() && !existingConfig.IsUnknown() &&
		!data.Configuration.IsNull() && !data.Configuration.IsUnknown() {
		merged := r.mergeConfigFromState(existingConfig, data.Configuration)
		data.Configuration = merged
	}

	// Restore upstream_ids from state if the API didn't echo them.
	if data.UpstreamIDs.IsNull() || data.UpstreamIDs.IsUnknown() {
		if !existingUpstreamIDs.IsNull() && !existingUpstreamIDs.IsUnknown() {
			data.UpstreamIDs = existingUpstreamIDs
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KnownPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state KnownPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assetVersion := r.resolveVersion(&plan)
	configData := r.expandConfiguration(ctx, plan.Configuration)

	if errs := apimanagement.ValidatePolicyConfiguration(r.policyInfo.AssetID, configData); len(errs) > 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid configuration for policy %q", r.policyInfo.AssetID),
			strings.Join(errs, "\n"),
		)
		return
	}

	orgID, envID, apiID, idErr := r.parseIDs(&state)
	if idErr != nil {
		resp.Diagnostics.AddError("Invalid IDs", idErr.Error())
		return
	}
	policyID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid policy ID", "Could not parse policy ID: "+state.ID.ValueString())
		return
	}

	var policy *apimanagement.APIPolicy
	if r.policyInfo.OutboundPolicy {
		// Outbound policies must NOT include pointcutData, order, or disabled.
		outboundUpdateReq := &apimanagement.UpdateOutboundAPIPolicyRequest{
			ConfigurationData: configData,
			AssetVersion:      assetVersion,
		}
		if !plan.Label.IsNull() && !plan.Label.IsUnknown() {
			outboundUpdateReq.Label = plan.Label.ValueString()
		}
		if !plan.UpstreamIDs.IsNull() && !plan.UpstreamIDs.IsUnknown() {
			var ids []string
			plan.UpstreamIDs.ElementsAs(ctx, &ids, false)
			outboundUpdateReq.UpstreamIDs = ids
		}
		policy, err = r.client.UpdateOutboundAPIPolicy(ctx, orgID, envID, apiID, policyID, outboundUpdateReq)
	} else {
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
		if !plan.UpstreamIDs.IsNull() && !plan.UpstreamIDs.IsUnknown() {
			var ids []string
			plan.UpstreamIDs.ElementsAs(ctx, &ids, false)
			updateReq.UpstreamIDs = ids
		}
		policy, err = r.client.UpdateAPIPolicy(ctx, orgID, envID, apiID, policyID, updateReq)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating policy", "Could not update policy: "+err.Error())
		return
	}

	plannedConfig := plan.Configuration
	r.flatten(ctx, policy, &plan, orgID, envID)
	if !plannedConfig.IsNull() && !plannedConfig.IsUnknown() {
		plan.Configuration = plannedConfig
	}
	tflog.Trace(ctx, "updated known API policy", map[string]interface{}{"id": policy.ID, "type": r.policyInfo.AssetID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KnownPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KnownPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID, envID, apiID, err := r.parseIDs(&data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid IDs", err.Error())
		return
	}
	policyID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid policy ID", "Could not parse policy ID: "+data.ID.ValueString())
		return
	}

	var deleteErr error
	if r.policyInfo.OutboundPolicy {
		deleteErr = r.client.DeleteOutboundAPIPolicy(ctx, orgID, envID, apiID, policyID)
	} else {
		deleteErr = r.client.DeleteAPIPolicy(ctx, orgID, envID, apiID, policyID)
	}
	if deleteErr != nil {
		resp.Diagnostics.AddError("Error deleting policy", "Could not delete policy: "+deleteErr.Error())
		return
	}

	tflog.Trace(ctx, "deleted known API policy", map[string]interface{}{"id": policyID, "type": r.policyInfo.AssetID})
}

func (r *KnownPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

// --- Helpers ---

func (r *KnownPolicyResource) resolveVersion(data *KnownPolicyResourceModel) string {
	if !data.AssetVersion.IsNull() && !data.AssetVersion.IsUnknown() && data.AssetVersion.ValueString() != "" {
		return data.AssetVersion.ValueString()
	}
	return r.policyInfo.DefaultVersion
}

func (r *KnownPolicyResource) parseIDs(data *KnownPolicyResourceModel) (orgID, envID string, apiID int, err error) {
	orgID = data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID = data.EnvironmentID.ValueString()

	apiID, err = strconv.Atoi(data.APIInstanceID.ValueString())
	if err != nil {
		return "", "", 0, fmt.Errorf("api_instance_id must be numeric: %s", data.APIInstanceID.ValueString())
	}
	return orgID, envID, apiID, nil
}

// mergeConfigFromState merges the existing state configuration into the freshly
// flattened API configuration. For each attribute that the API returned as null
// (write-only fields like passwords, tokens, secrets), the prior state value is
// restored so Terraform does not generate a spurious diff.
func (r *KnownPolicyResource) mergeConfigFromState(stateConfig, apiConfig types.Object) types.Object {
	stateAttrs := stateConfig.Attributes()
	apiAttrs := apiConfig.Attributes()

	merged := make(map[string]attr.Value, len(apiAttrs))
	for k, apiVal := range apiAttrs {
		// If the API returned null/unknown for this field but state has a real
		// value, keep the state value (the field is write-only / not echoed by API).
		if apiVal.IsNull() || apiVal.IsUnknown() {
			if stateVal, ok := stateAttrs[k]; ok && !stateVal.IsNull() && !stateVal.IsUnknown() {
				merged[k] = stateVal
				continue
			}
		}
		merged[k] = apiVal
	}

	obj, _ := types.ObjectValue(apiConfig.AttributeTypes(context.Background()), merged)
	return obj
}

func (r *KnownPolicyResource) flatten(ctx context.Context, policy *apimanagement.APIPolicy, data *KnownPolicyResourceModel, orgID, envID string) {
	data.ID = types.StringValue(strconv.Itoa(policy.ID))
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)

	if policy.APIID != 0 {
		data.APIInstanceID = types.StringValue(strconv.Itoa(policy.APIID))
	}

	data.AssetVersion = types.StringValue(policy.AssetVersion)
	data.Order = types.Int64Value(int64(policy.Order))
	data.Disabled = types.BoolValue(policy.Disabled)
	data.PolicyTemplateID = types.StringValue(policy.PolicyTemplateID)

	if policy.Label != "" {
		data.Label = types.StringValue(policy.Label)
	}

	if policy.ConfigurationData != nil {
		cfgObj, _ := r.flattenConfiguration(ctx, policy.ConfigurationData)
		data.Configuration = cfgObj
	}

	if len(policy.UpstreamIDs) > 0 {
		elems := make([]attr.Value, len(policy.UpstreamIDs))
		for i, id := range policy.UpstreamIDs {
			elems[i] = types.StringValue(id)
		}
		listVal, _ := types.ListValue(types.StringType, elems)
		data.UpstreamIDs = listVal
	} else if data.UpstreamIDs.IsNull() || data.UpstreamIDs.IsUnknown() {
		data.UpstreamIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}
}

// KnownPolicyTypes returns the list of all known policy type names,
// used by provider.go to register each one as a dedicated resource.
func KnownPolicyTypes() []string {
	result := make([]string, 0, len(apimanagement.KnownPolicies))
	for k := range apimanagement.KnownPolicies {
		result = append(result, k)
	}
	return result
}

// numberAtLeastValidator validates that a Number value is >= min.
type numberAtLeastValidator struct{ min *big.Float }

func (v numberAtLeastValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be at least %s", v.min.Text('f', 0))
}
func (v numberAtLeastValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v numberAtLeastValidator) ValidateNumber(_ context.Context, req validator.NumberRequest, resp *validator.NumberResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueBigFloat()
	if val.Cmp(v.min) < 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Value too small",
			fmt.Sprintf("must be >= %s, got %s", v.min.Text('f', 0), val.Text('f', 0)))
	}
}

// numberAtMostValidator validates that a Number value is <= max.
type numberAtMostValidator struct{ max *big.Float }

func (v numberAtMostValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be at most %s", v.max.Text('f', 0))
}
func (v numberAtMostValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v numberAtMostValidator) ValidateNumber(_ context.Context, req validator.NumberRequest, resp *validator.NumberResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueBigFloat()
	if val.Cmp(v.max) > 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Value too large",
			fmt.Sprintf("must be <= %s, got %s", v.max.Text('f', 0), val.Text('f', 0)))
	}
}
