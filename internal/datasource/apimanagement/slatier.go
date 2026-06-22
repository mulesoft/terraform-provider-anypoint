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
	_ datasource.DataSource              = &SLATierDataSource{}
	_ datasource.DataSourceWithConfigure = &SLATierDataSource{}
)

// SLATierDataSource fetches a single SLA tier by ID.
type SLATierDataSource struct {
	client *apimanagement.SLATierClient
}

type SLATierDataSourceModel struct {
	ID             types.String    `tfsdk:"id"`
	OrganizationID types.String    `tfsdk:"organization_id"`
	EnvironmentID  types.String    `tfsdk:"environment_id"`
	APIInstanceID  types.String    `tfsdk:"api_instance_id"`
	TierID         types.String    `tfsdk:"tier_id"`
	Name           types.String    `tfsdk:"name"`
	Description    types.String    `tfsdk:"description"`
	AutoApprove    types.Bool      `tfsdk:"auto_approve"`
	Status         types.String    `tfsdk:"status"`
	Limits         []SLALimitModel `tfsdk:"limits"`
}

type SLALimitModel struct {
	TimePeriodInMilliseconds types.Int64 `tfsdk:"time_period_in_milliseconds"`
	MaximumRequests          types.Int64 `tfsdk:"maximum_requests"`
	Visible                  types.Bool  `tfsdk:"visible"`
}

func NewSLATierDataSource() datasource.DataSource {
	return &SLATierDataSource{}
}

func (d *SLATierDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_instance_sla_tier"
}

func (d *SLATierDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single SLA tier for an API instance in API Manager.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier: <organization_id>/<environment_id>/<api_instance_id>/<tier_id>.",
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
			"tier_id": schema.StringAttribute{
				Description: "The ID of the SLA tier to look up.",
				Required:    true,
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
	}
}

func (d *SLATierDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SLATierDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SLATierDataSourceModel

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
	tierIDStr := data.TierID.ValueString()

	// Convert string IDs to int
	apiInstanceID, err := strconv.Atoi(apiInstanceIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid API Instance ID",
			"Could not convert api_instance_id to integer: "+err.Error(),
		)
		return
	}

	tierID, err := strconv.Atoi(tierIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Tier ID",
			"Could not convert tier_id to integer: "+err.Error(),
		)
		return
	}

	tier, err := d.client.GetSLATier(ctx, orgID, envID, apiInstanceID, tierID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading SLA tier",
			"Could not read SLA tier "+tierIDStr+" for API instance "+apiInstanceIDStr+": "+err.Error(),
		)
		return
	}

	// Map the response to the model
	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s", orgID, envID, apiInstanceIDStr, tierIDStr))
	data.OrganizationID = types.StringValue(orgID)
	data.Name = types.StringValue(tier.Name)
	data.Description = types.StringValue(tier.Description)
	data.AutoApprove = types.BoolValue(tier.AutoApprove)
	data.Status = types.StringValue(tier.Status)

	// Map limits
	data.Limits = make([]SLALimitModel, 0, len(tier.Limits))
	for _, limit := range tier.Limits {
		data.Limits = append(data.Limits, SLALimitModel{
			TimePeriodInMilliseconds: types.Int64Value(int64(limit.TimePeriodInMilliseconds)),
			MaximumRequests:          types.Int64Value(int64(limit.MaximumRequests)),
			Visible:                  types.BoolValue(limit.Visible),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
