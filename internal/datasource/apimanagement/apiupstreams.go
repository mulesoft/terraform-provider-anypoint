package apimanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &APIUpstreamsDataSource{}
	_ datasource.DataSourceWithConfigure = &APIUpstreamsDataSource{}
)

// APIUpstreamsDataSource lists all upstreams for an API instance.
type APIUpstreamsDataSource struct {
	client *apimanagement.APIUpstreamsClient
}

type APIUpstreamsDataSourceModel struct {
	ID            types.String        `tfsdk:"id"`
	OrganizationID types.String       `tfsdk:"organization_id"`
	EnvironmentID  types.String       `tfsdk:"environment_id"`
	APIInstanceID  types.String       `tfsdk:"api_instance_id"`
	Total          types.Int64        `tfsdk:"total"`
	Upstreams      []APIUpstreamModel `tfsdk:"upstreams"`
}

type APIUpstreamModel struct {
	ID    types.String `tfsdk:"id"`
	Label types.String `tfsdk:"label"`
	URI   types.String `tfsdk:"uri"`
}

func NewAPIUpstreamsDataSource() datasource.DataSource {
	return &APIUpstreamsDataSource{}
}

func (d *APIUpstreamsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_upstreams"
}

func (d *APIUpstreamsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all upstreams registered for an API instance in API Manager.",
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
				Description: "The environment ID where the API instance lives.",
				Required:    true,
			},
			"api_instance_id": schema.StringAttribute{
				Description: "The numeric ID of the API instance.",
				Required:    true,
			},
			"total": schema.Int64Attribute{
				Description: "Total number of upstreams returned.",
				Computed:    true,
			},
			"upstreams": schema.ListNestedAttribute{
				Description: "List of upstreams for the API instance.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The upstream UUID.",
							Computed:    true,
						},
						"label": schema.StringAttribute{
							Description: "The upstream label (matches the label in the routing configuration).",
							Computed:    true,
						},
						"uri": schema.StringAttribute{
							Description: "The upstream URI.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *APIUpstreamsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	apiClient, err := apimanagement.NewAPIUpstreamsClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create API Upstreams Client",
			"An unexpected error occurred when creating the API Upstreams client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = apiClient
}

func (d *APIUpstreamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIUpstreamsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	apiID := data.APIInstanceID.ValueString()

	upstreams, err := d.client.ListUpstreams(ctx, orgID, envID, apiID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing API upstreams",
			fmt.Sprintf("Could not list upstreams for API instance %s: %s", apiID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(orgID + "/" + envID + "/" + apiID)
	data.OrganizationID = types.StringValue(orgID)
	data.Total = types.Int64Value(int64(len(upstreams)))
	data.Upstreams = make([]APIUpstreamModel, 0, len(upstreams))

	for _, u := range upstreams {
		data.Upstreams = append(data.Upstreams, APIUpstreamModel{
			ID:    types.StringValue(u.ID),
			Label: types.StringValue(u.Label),
			URI:   types.StringValue(u.URI),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
