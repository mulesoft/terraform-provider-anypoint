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
	_ datasource.DataSource              = &APIPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &APIPolicyDataSource{}
)

// APIPolicyDataSource retrieves a single API policy by ID.
type APIPolicyDataSource struct {
	client *apimanagement.APIPolicyClient
}

type APIPolicyDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	EnvironmentID      types.String `tfsdk:"environment_id"`
	APIInstanceID      types.String `tfsdk:"api_instance_id"`
	PolicyID           types.String `tfsdk:"policy_id"`
	PolicyTemplateID   types.String `tfsdk:"policy_template_id"`
	GroupID            types.String `tfsdk:"group_id"`
	AssetID            types.String `tfsdk:"asset_id"`
	AssetVersion       types.String `tfsdk:"asset_version"`
	ConfigurationJSON  types.String `tfsdk:"configuration_json"`
	Order              types.Int64  `tfsdk:"order"`
	Disabled           types.Bool   `tfsdk:"disabled"`
	PointcutJSON       types.String `tfsdk:"pointcut_json"`
}

func NewAPIPolicyDataSource() datasource.DataSource {
	return &APIPolicyDataSource{}
}

func (d *APIPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_policy"
}

func (d *APIPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a single API policy by ID from API Manager.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier: <organization_id>/<environment_id>/<api_instance_id>/<policy_id>.",
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
			"policy_id": schema.StringAttribute{
				Description: "The ID of the policy to retrieve.",
				Required:    true,
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
	}
}

func (d *APIPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *APIPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIPolicyDataSourceModel

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
	policyIDStr := data.PolicyID.ValueString()

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid API Instance ID",
			"Could not convert api_instance_id to integer: "+err.Error(),
		)
		return
	}

	policyID, err := strconv.Atoi(policyIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Policy ID",
			"Could not convert policy_id to integer: "+err.Error(),
		)
		return
	}

	policy, err := d.client.GetAPIPolicy(ctx, orgID, envID, apiID, policyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error retrieving API policy",
			fmt.Sprintf("Could not retrieve policy %d for API instance %d: %s", policyID, apiID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s", orgID, envID, apiIDStr, policyIDStr))
	data.OrganizationID = types.StringValue(orgID)
	data.PolicyTemplateID = types.StringValue(policy.PolicyTemplateID)
	data.GroupID = types.StringValue(policy.GroupID)
	data.AssetID = types.StringValue(policy.AssetID)
	data.AssetVersion = types.StringValue(policy.AssetVersion)
	data.Order = types.Int64Value(int64(policy.Order))
	data.Disabled = types.BoolValue(policy.Disabled)

	if len(policy.ConfigurationData) > 0 {
		configBytes, err := json.Marshal(policy.ConfigurationData)
		if err == nil {
			data.ConfigurationJSON = types.StringValue(string(configBytes))
		} else {
			data.ConfigurationJSON = types.StringNull()
		}
	} else {
		data.ConfigurationJSON = types.StringNull()
	}

	if policy.PointcutData != nil {
		pointcutBytes, err := json.Marshal(policy.PointcutData)
		if err == nil {
			data.PointcutJSON = types.StringValue(string(pointcutBytes))
		} else {
			data.PointcutJSON = types.StringNull()
		}
	} else {
		data.PointcutJSON = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
