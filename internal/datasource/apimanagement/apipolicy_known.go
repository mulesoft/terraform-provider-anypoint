package apimanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &KnownPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &KnownPolicyDataSource{}
)

// KnownPolicyDataSource retrieves a specific known policy applied to an API instance.
// It looks up the policy either by policy_id or by matching the asset_id on the API instance.
type KnownPolicyDataSource struct {
	client     *apimanagement.APIPolicyClient
	policyInfo apimanagement.PolicyInfo
	typeSuffix string
}

// KnownPolicyDataSourceModel is the Terraform model for this data source.
type KnownPolicyDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	APIInstanceID    types.String `tfsdk:"api_instance_id"`
	PolicyID         types.String `tfsdk:"policy_id"`
	Label            types.String `tfsdk:"label"`
	Configuration    types.Object `tfsdk:"configuration"`
	PointcutData     types.String `tfsdk:"pointcut_data"`
	Order            types.Int64  `tfsdk:"order"`
	Disabled         types.Bool   `tfsdk:"disabled"`
	PolicyTemplateID types.String `tfsdk:"policy_template_id"`
	AssetVersion     types.String `tfsdk:"asset_version"`
	UpstreamIDs      types.List   `tfsdk:"upstream_ids"`
}

// NewKnownPolicyDataSourceFunc returns a factory function that creates a data source
// for a specific known policy type (e.g., "rate-limiting", "cors").
func NewKnownPolicyDataSourceFunc(policyType string) func() datasource.DataSource {
	info, ok := apimanagement.LookupPolicy(policyType)
	if !ok {
		panic(fmt.Sprintf("BUG: unknown policy type %q passed to NewKnownPolicyDataSourceFunc", policyType))
	}
	suffix := strings.ReplaceAll(policyType, "-", "_")
	return func() datasource.DataSource {
		return &KnownPolicyDataSource{
			policyInfo: info,
			typeSuffix: suffix,
		}
	}
}

func (d *KnownPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_policy_" + d.typeSuffix
}

// --- Schema Generation ---

// configAttrTypes returns the attr.Type map for the configuration object.
func (d *KnownPolicyDataSource) configAttrTypes() map[string]attr.Type {
	policySchema, ok := apimanagement.KnownPolicySchemas[d.policyInfo.AssetID]
	if !ok {
		return map[string]attr.Type{}
	}
	attrTypes := make(map[string]attr.Type, len(policySchema))
	for camelName, field := range policySchema {
		snakeName := apimanagement.CamelToSnake(camelName)
		attrTypes[snakeName] = dsFieldSchemaType(field.Type)
	}
	return attrTypes
}

func dsFieldSchemaType(t string) attr.Type {
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

// generateDSConfigurationSchema creates a read-only nested attribute for the policy configuration.
func generateDSConfigurationSchema(assetID string) schema.SingleNestedAttribute {
	policySchema, ok := apimanagement.KnownPolicySchemas[assetID]
	if !ok {
		return schema.SingleNestedAttribute{
			Description: "Policy configuration.",
			Computed:    true,
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
				Computed:    true,
			}
		case "int":
			attrs[snakeName] = schema.NumberAttribute{
				Description: fmt.Sprintf("Policy field '%s'.", camelName),
				Computed:    true,
			}
		case "bool":
			attrs[snakeName] = schema.BoolAttribute{
				Description: fmt.Sprintf("Policy field '%s'.", camelName),
				Computed:    true,
			}
		case "string_array":
			attrs[snakeName] = schema.ListAttribute{
				Description: fmt.Sprintf("Policy field '%s'.", camelName),
				Computed:    true,
				ElementType: types.StringType,
			}
		default:
			attrs[snakeName] = schema.DynamicAttribute{
				Description: fmt.Sprintf("Policy field '%s'.", camelName),
				Computed:    true,
			}
		}
	}

	return schema.SingleNestedAttribute{
		Description: "Policy configuration with typed fields.",
		Computed:    true,
		Attributes:  attrs,
	}
}

func (d *KnownPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	desc := fmt.Sprintf(
		"Reads the %s policy applied to an API instance. "+
			"If policy_id is omitted, the data source automatically finds the policy by its asset type.",
		d.policyInfo.AssetID,
	)

	resp.Schema = schema.Schema{
		Description: desc,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the applied policy.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID. Defaults to the provider's org ID if omitted.",
				Optional:    true,
				Computed:    true,
			},
			"environment_id": schema.StringAttribute{
				Description: "Environment ID where the API instance lives.",
				Required:    true,
			},
			"api_instance_id": schema.StringAttribute{
				Description: "Numeric ID of the API instance this policy is applied to.",
				Required:    true,
			},
			"policy_id": schema.StringAttribute{
				Description: "The ID of the policy. If omitted, the data source looks up the policy by asset type.",
				Optional:    true,
				Computed:    true,
			},
			"label": schema.StringAttribute{
				Description: "A human-readable label for this policy instance.",
				Computed:    true,
			},
			"configuration": generateDSConfigurationSchema(d.policyInfo.AssetID),
			"order": schema.Int64Attribute{
				Description: "Execution order of the policy.",
				Computed:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Whether the policy is disabled.",
				Computed:    true,
			},
			"policy_template_id": schema.StringAttribute{
				Description: "Policy template ID assigned by the server.",
				Computed:    true,
			},
			"asset_version": schema.StringAttribute{
				Description: "Version of the policy asset.",
				Computed:    true,
			},
			"upstream_ids": schema.ListAttribute{
				Description: "List of upstream IDs this policy applies to.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"pointcut_data": schema.StringAttribute{
				Description: "Pointcut definition as a JSON string.",
				Computed:    true,
			},
		},
	}
}

func (d *KnownPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	policyClient, err := apimanagement.NewAPIPolicyClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create API Policy Client",
			"An unexpected error occurred when creating the API Policy client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = policyClient
}

func (d *KnownPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KnownPolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	apiID, err := strconv.Atoi(data.APIInstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid API Instance ID",
			"api_instance_id must be numeric: "+data.APIInstanceID.ValueString(),
		)
		return
	}

	var policy *apimanagement.APIPolicy

	// If policy_id is provided, fetch directly.
	if !data.PolicyID.IsNull() && !data.PolicyID.IsUnknown() && data.PolicyID.ValueString() != "" {
		policyID, err := strconv.Atoi(data.PolicyID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Policy ID",
				"policy_id must be numeric: "+data.PolicyID.ValueString(),
			)
			return
		}

		if d.policyInfo.OutboundPolicy {
			policy, err = d.client.GetOutboundAPIPolicy(ctx, orgID, envID, apiID, policyID)
		} else {
			policy, err = d.client.GetAPIPolicy(ctx, orgID, envID, apiID, policyID)
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading policy",
				fmt.Sprintf("Could not read policy %d: %s", policyID, err.Error()),
			)
			return
		}
	} else {
		// Auto-discover: list policies and find the one matching our asset ID.
		policies, err := d.client.ListAPIPolicies(ctx, orgID, envID, apiID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing policies",
				fmt.Sprintf("Could not list policies for API instance %d: %s", apiID, err.Error()),
			)
			return
		}

		var matches []apimanagement.APIPolicy
		for _, p := range policies {
			if p.AssetID == d.policyInfo.AssetID {
				matches = append(matches, p)
			}
		}

		if len(matches) == 0 {
			resp.Diagnostics.AddError(
				"Policy not found",
				fmt.Sprintf("No %s policy found on API instance %d. "+
					"Ensure the policy is applied before reading it.", d.policyInfo.AssetID, apiID),
			)
			return
		}

		if len(matches) > 1 {
			ids := make([]string, len(matches))
			for i, m := range matches {
				ids[i] = strconv.Itoa(m.ID)
			}
			resp.Diagnostics.AddError(
				"Multiple policies found",
				fmt.Sprintf("Found %d %s policies on API instance %d (IDs: %s). "+
					"Specify policy_id to select a specific one.",
					len(matches), d.policyInfo.AssetID, apiID, strings.Join(ids, ", ")),
			)
			return
		}

		// Single match — use it. For outbound policies, fetch the full record.
		if d.policyInfo.OutboundPolicy {
			policy, err = d.client.GetOutboundAPIPolicy(ctx, orgID, envID, apiID, matches[0].ID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error reading outbound policy",
					fmt.Sprintf("Could not read outbound policy %d: %s", matches[0].ID, err.Error()),
				)
				return
			}
		} else {
			found := matches[0]
			policy = &found
		}
	}

	// Flatten the API response into the data model.
	d.flatten(ctx, policy, &data, orgID, envID)

	tflog.Trace(ctx, "read known API policy data source", map[string]interface{}{
		"id":   policy.ID,
		"type": d.policyInfo.AssetID,
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// flatten maps an APIPolicy response into the data source model.
func (d *KnownPolicyDataSource) flatten(ctx context.Context, policy *apimanagement.APIPolicy, data *KnownPolicyDataSourceModel, orgID, envID string) {
	data.ID = types.StringValue(strconv.Itoa(policy.ID))
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.PolicyID = types.StringValue(strconv.Itoa(policy.ID))

	if policy.APIID != 0 {
		data.APIInstanceID = types.StringValue(strconv.Itoa(policy.APIID))
	}

	data.AssetVersion = types.StringValue(policy.AssetVersion)
	data.Order = types.Int64Value(int64(policy.Order))
	data.Disabled = types.BoolValue(policy.Disabled)
	data.PolicyTemplateID = types.StringValue(policy.PolicyTemplateID)

	if policy.Label != "" {
		data.Label = types.StringValue(policy.Label)
	} else {
		data.Label = types.StringNull()
	}

	if policy.ConfigurationData != nil {
		cfgObj := d.flattenConfiguration(ctx, policy.ConfigurationData)
		data.Configuration = cfgObj
	} else {
		// Null configuration
		attrTypes := d.configAttrTypes()
		data.Configuration = types.ObjectNull(attrTypes)
	}

	if len(policy.UpstreamIDs) > 0 {
		elems := make([]attr.Value, len(policy.UpstreamIDs))
		for i, id := range policy.UpstreamIDs {
			elems[i] = types.StringValue(id)
		}
		listVal, _ := types.ListValue(types.StringType, elems)
		data.UpstreamIDs = listVal
	} else {
		data.UpstreamIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if policy.PointcutData != nil {
		pcJSON, err := json.Marshal(policy.PointcutData)
		if err == nil {
			data.PointcutData = types.StringValue(string(pcJSON))
		} else {
			data.PointcutData = types.StringNull()
		}
	} else {
		data.PointcutData = types.StringNull()
	}
}

// flattenConfiguration converts the API's map[string]interface{} response into a types.Object.
func (d *KnownPolicyDataSource) flattenConfiguration(_ context.Context, configData map[string]interface{}) types.Object {
	attrTypes := d.configAttrTypes()

	if len(attrTypes) == 0 {
		obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{})
		return obj
	}

	attrValues := make(map[string]attr.Value, len(attrTypes))

	// Initialize all attrs to null
	for name, t := range attrTypes {
		attrValues[name] = dsNullForType(t)
	}

	policySchema := apimanagement.KnownPolicySchemas[d.policyInfo.AssetID]

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
			tfVal := dsNativeToDynamic(rawVal)
			attrValues[snakeName] = types.DynamicValue(tfVal)
		}
	}

	obj, _ := types.ObjectValue(attrTypes, attrValues)
	return obj
}

func dsNullForType(t attr.Type) attr.Value {
	switch t {
	case types.StringType:
		return types.StringNull()
	case types.NumberType:
		return types.NumberNull()
	case types.BoolType:
		return types.BoolNull()
	default:
		if lt, ok := t.(types.ListType); ok {
			return types.ListNull(lt.ElemType)
		}
		return types.DynamicNull()
	}
}

// dsNativeToDynamic converts a native Go value (from JSON) into a Terraform attr.Value.
func dsNativeToDynamic(v interface{}) attr.Value {
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
			ev := dsNativeToDynamic(e)
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
			ev := dsNativeToDynamic(e)
			attrTypes[snakeKey] = ev.Type(context.Background())
			attrValues[snakeKey] = ev
		}
		ov, _ := types.ObjectValue(attrTypes, attrValues)
		return ov
	default:
		return types.StringValue(fmt.Sprintf("%v", v))
	}
}

// KnownPolicyDataSourceTypes returns the list of all known policy type names,
// used by provider.go to register each one as a dedicated data source.
func KnownPolicyDataSourceTypes() []string {
	result := make([]string, 0, len(apimanagement.KnownPolicies))
	for k := range apimanagement.KnownPolicies {
		result = append(result, k)
	}
	return result
}
