package apimanagement

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &SLATiersDataSource{}
	_ datasource.DataSourceWithConfigure = &SLATiersDataSource{}
)

// SLATiersDataSource lists all SLA tiers for an API instance.
type SLATiersDataSource struct {
	client *apimanagement.SLATierClient
}

type SLATiersDataSourceModel struct {
	ID             types.String         `tfsdk:"id"`
	OrganizationID types.String         `tfsdk:"organization_id"`
	EnvironmentID  types.String         `tfsdk:"environment_id"`
	APIInstanceID  types.String         `tfsdk:"api_instance_id"`
	Tiers          []SLATierItemModel `tfsdk:"tiers"`
}

type SLATierItemModel struct {
	ID          types.String    `tfsdk:"id"`
	Name        types.String    `tfsdk:"name"`
	Description types.String    `tfsdk:"description"`
	AutoApprove types.Bool      `tfsdk:"auto_approve"`
	Status      types.String    `tfsdk:"status"`
	Limits      []SLALimitModel `tfsdk:"limits"`
}

func NewSLATiersDataSource() datasource.DataSource {
	return &SLATiersDataSource{}
}

func (d *SLATiersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_instance_sla_tiers"
}

func (d *SLATiersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all SLA tiers for an API instance in API Manager.",
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
				Description: "The numeric ID of the API instance.",
				Required:    true,
			},
			"tiers": schema.ListNestedAttribute{
				Description: "List of SLA tiers for this API instance.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The numeric ID of the SLA tier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the SLA tier.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the SLA tier.",
							Computed:    true,
						},
						"auto_approve": schema.BoolAttribute{
							Description: "Whether requests for this tier are auto-approved.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the SLA tier.",
							Computed:    true,
						},
						"limits": schema.ListNestedAttribute{
							Description: "List of SLA limits for this tier.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"time_period_in_milliseconds": schema.Int64Attribute{
										Description: "The time period for the limit in milliseconds.",
										Computed:    true,
									},
									"maximum_requests": schema.Int64Attribute{
										Description: "The maximum number of requests allowed in the time period.",
										Computed:    true,
									},
									"visible": schema.BoolAttribute{
										Description: "Whether this limit is visible to API consumers.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *SLATiersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	slaClient, err := apimanagement.NewSLATierClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create SLA Tier Client",
			"An unexpected error occurred when creating the SLA Tier client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = slaClient
}

func (d *SLATiersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SLATiersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	apiInstanceIDStr := data.APIInstanceID.ValueString()

	// Convert string ID to int
	apiInstanceID, err := strconv.Atoi(apiInstanceIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid API Instance ID",
			"Could not convert api_instance_id to integer: "+err.Error(),
		)
		return
	}

	tiers, err := d.client.ListSLATiers(ctx, orgID, envID, apiInstanceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing SLA tiers",
			"Could not list SLA tiers for API instance "+apiInstanceIDStr+": "+err.Error(),
		)
		return
	}

	// Map the response to the model
	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", orgID, envID, apiInstanceIDStr))
	data.OrganizationID = types.StringValue(orgID)
	data.Tiers = make([]SLATierItemModel, 0, len(tiers))

	for _, tier := range tiers {
		limits := make([]SLALimitModel, 0, len(tier.Limits))
		for _, limit := range tier.Limits {
			limits = append(limits, SLALimitModel{
				TimePeriodInMilliseconds: types.Int64Value(int64(limit.TimePeriodInMilliseconds)),
				MaximumRequests:          types.Int64Value(int64(limit.MaximumRequests)),
				Visible:                  types.BoolValue(limit.Visible),
			})
		}

		data.Tiers = append(data.Tiers, SLATierItemModel{
			ID:          types.StringValue(strconv.Itoa(tier.ID)),
			Name:        types.StringValue(tier.Name),
			Description: types.StringValue(tier.Description),
			AutoApprove: types.BoolValue(tier.AutoApprove),
			Status:      types.StringValue(tier.Status),
			Limits:      limits,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
