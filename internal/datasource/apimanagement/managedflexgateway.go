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
	_ datasource.DataSource              = &ManagedFlexGatewayDataSource{}
	_ datasource.DataSourceWithConfigure = &ManagedFlexGatewayDataSource{}
)

// ManagedFlexGatewayDataSource lists all managed Flex Gateways in an environment.
type ManagedFlexGatewayDataSource struct {
	client *apimanagement.ManagedFlexGatewayClient
}

type ManagedFlexGatewayDataSourceModel struct {
	ID             types.String                  `tfsdk:"id"`
	OrganizationID types.String                  `tfsdk:"organization_id"`
	EnvironmentID  types.String                  `tfsdk:"environment_id"`
	Gateways       []ManagedFlexGatewayItemModel `tfsdk:"gateways"`
}

// ManagedFlexGatewayItemModel reflects the fields returned by the api/v1 list endpoint.
type ManagedFlexGatewayItemModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	TargetID    types.String `tfsdk:"target_id"`
	Status      types.String `tfsdk:"status"`
	DateCreated types.String `tfsdk:"date_created"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func NewManagedFlexGatewayDataSource() datasource.DataSource {
	return &ManagedFlexGatewayDataSource{}
}

func (d *ManagedFlexGatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_flexgateways"
}

func (d *ManagedFlexGatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all managed Flex Gateway instances in the given environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier: <organization_id>/<environment_id>.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider credentials organization.",
				Optional:    true,
				Computed:    true,
			},
			"environment_id": schema.StringAttribute{
				Description: "The environment ID to list gateways from.",
				Required:    true,
			},
			"gateways": schema.ListNestedAttribute{
				Description: "List of managed Flex Gateway instances.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the gateway.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the gateway.",
							Computed:    true,
						},
						"target_id": schema.StringAttribute{
							Description: "The target (private space) ID the gateway is deployed to.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the gateway (e.g. APPLIED, RUNNING).",
							Computed:    true,
						},
						"date_created": schema.StringAttribute{
							Description: "Timestamp when the gateway was created.",
							Computed:    true,
						},
						"last_updated": schema.StringAttribute{
							Description: "Timestamp of the last update to the gateway.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ManagedFlexGatewayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	gwClient, err := apimanagement.NewManagedFlexGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Managed Flex Gateway Client",
			"An unexpected error occurred when creating the Managed Flex Gateway client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = gwClient
}

func (d *ManagedFlexGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ManagedFlexGatewayDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	gateways, err := d.client.ListManagedFlexGateways(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing managed Flex Gateways",
			"Could not list managed Flex Gateways for environment "+envID+": "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(orgID + "/" + envID)
	data.OrganizationID = types.StringValue(orgID)
	data.Gateways = make([]ManagedFlexGatewayItemModel, len(gateways))

	for i, gw := range gateways {
		data.Gateways[i] = ManagedFlexGatewayItemModel{
			ID:          types.StringValue(gw.ID),
			Name:        types.StringValue(gw.Name),
			TargetID:    types.StringValue(gw.TargetID),
			Status:      types.StringValue(gw.Status),
			DateCreated: types.StringValue(gw.DateCreated),
			LastUpdated: types.StringValue(gw.LastUpdated),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
