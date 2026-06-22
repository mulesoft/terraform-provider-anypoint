package apimanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &APIPoliciesDataSource{}
	_ datasource.DataSourceWithConfigure = &APIPoliciesDataSource{}
)

// APIPoliciesDataSource lists all API policies for an API instance.
type APIPoliciesDataSource struct {
	client *apimanagement.APIPolicyClient
}

type APIPoliciesDataSourceModel struct {
	ID             types.String         `tfsdk:"id"`
	OrganizationID types.String         `tfsdk:"organization_id"`
	EnvironmentID  types.String         `tfsdk:"environment_id"`
	APIInstanceID  types.String         `tfsdk:"api_instance_id"`
	Policies       []APIPolicyItemModel `tfsdk:"policies"`
}

type APIPolicyItemModel struct {
	ID                types.String `tfsdk:"id"`
	PolicyTemplateID  types.String `tfsdk:"policy_template_id"`
	GroupID           types.String `tfsdk:"group_id"`
	AssetID           types.String `tfsdk:"asset_id"`
	AssetVersion      types.String `tfsdk:"asset_version"`
	ConfigurationJSON types.String `tfsdk:"configuration_json"`
	Order             types.Int64  `tfsdk:"order"`
	Disabled          types.Bool   `tfsdk:"disabled"`
	PointcutJSON      types.String `tfsdk:"pointcut_json"`
}

func NewAPIPoliciesDataSource() datasource.DataSource {
	return &APIPoliciesDataSource{}
}

func (d *APIPoliciesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_policies"
}

func (d *APIPoliciesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all API policies for an API instance in API Manager.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier: <organization_id>/<environment_id>/<api_instance_id>.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider credentials organization.",
				Optional:    true,
				Computed:    true,
			},
			"environment_id": schema.StringAttribute{
				Description: "The environment ID where the API instance exists.",
				Required:    true,
			},
			"api_instance_id": schema.StringAttribute{
				Description: "The API instance ID.",
				Required:    true,
			},
			"policies": schema.ListNestedAttribute{
				Description: "List of API policies.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The numeric ID of the policy.",
							Computed:    true,
						},
						"policy_template_id": schema.StringAttribute{
							Description: "The policy template ID.",
							Computed:    true,
						},
						"group_id": schema.StringAttribute{
							Description: "The Exchange group (organization) ID.",
							Computed:    true,
						},
						"asset_id": schema.StringAttribute{
							Description: "The Exchange asset ID.",
							Computed:    true,
						},
						"asset_version": schema.StringAttribute{
							Description: "The Exchange asset version.",
							Computed:    true,
						},
						"configuration_json": schema.StringAttribute{
							Description: "JSON-encoded policy configuration.",
							Computed:    true,
						},
						"order": schema.Int64Attribute{
							Description: "The execution order of the policy.",
							Computed:    true,
						},
						"disabled": schema.BoolAttribute{
							Description: "Whether the policy is disabled.",
							Computed:    true,
						},
						"pointcut_json": schema.StringAttribute{
							Description: "JSON-encoded pointcut data.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *APIPoliciesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	apiClient, err := apimanagement.NewAPIPolicyClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create API Policy Client",
			"An unexpected error occurred when creating the API Policy client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = apiClient
}

func (d *APIPoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIPoliciesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	apiIDStr := data.APIInstanceID.ValueString()

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid API Instance ID",
			"Could not convert api_instance_id to integer: "+err.Error(),
		)
		return
	}

	policies, err := d.client.ListAPIPolicies(ctx, orgID, envID, apiID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing API policies",
			fmt.Sprintf("Could not list policies for API instance %d: %s", apiID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", orgID, envID, apiIDStr))
	data.OrganizationID = types.StringValue(orgID)
	data.Policies = make([]APIPolicyItemModel, 0, len(policies))

	for _, policy := range policies {
		data.Policies = append(data.Policies, mapAPIPolicyToItemModel(policy))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapAPIPolicyToItemModel converts a client APIPolicy to the datasource item model.
func mapAPIPolicyToItemModel(policy apimanagement.APIPolicy) APIPolicyItemModel {
	configJSON := types.StringNull()
	if len(policy.ConfigurationData) > 0 {
		configBytes, err := json.Marshal(policy.ConfigurationData)
		if err == nil {
			configJSON = types.StringValue(string(configBytes))
		}
	}

	pointcutJSON := types.StringNull()
	if policy.PointcutData != nil {
		pointcutBytes, err := json.Marshal(policy.PointcutData)
		if err == nil {
			pointcutJSON = types.StringValue(string(pointcutBytes))
		}
	}

	return APIPolicyItemModel{
		ID:                types.StringValue(strconv.Itoa(policy.ID)),
		PolicyTemplateID:  types.StringValue(policy.PolicyTemplateID),
		GroupID:           types.StringValue(policy.GroupID),
		AssetID:           types.StringValue(policy.AssetID),
		AssetVersion:      types.StringValue(policy.AssetVersion),
		ConfigurationJSON: configJSON,
		Order:             types.Int64Value(int64(policy.Order)),
		Disabled:          types.BoolValue(policy.Disabled),
		PointcutJSON:      pointcutJSON,
	}
}
