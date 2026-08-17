package apimanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &ManagedOmniGatewayDataSource{}
	_ datasource.DataSourceWithConfigure = &ManagedOmniGatewayDataSource{}
)

// ManagedOmniGatewayDataSource lists all managed Omni Gateways in an environment.
type ManagedOmniGatewayDataSource struct {
	client *apimanagement.ManagedOmniGatewayClient
}

type ManagedOmniGatewayDataSourceModel struct {
	ID             types.String                  `tfsdk:"id"`
	OrganizationID types.String                  `tfsdk:"organization_id"`
	EnvironmentID  types.String                  `tfsdk:"environment_id"`
	Gateways       []ManagedOmniGatewayItemModel `tfsdk:"gateways"`
}

// ManagedOmniGatewayItemModel reflects the fields returned by the api/v1 list endpoint.
// target_type is not returned by the list endpoint; the data source enriches it
// from the deployment-targets list (see Read).
type ManagedOmniGatewayItemModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	TargetID    types.String `tfsdk:"target_id"`
	TargetType  types.String `tfsdk:"target_type"`
	Status      types.String `tfsdk:"status"`
	DateCreated types.String `tfsdk:"date_created"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func NewManagedOmniGatewayDataSource() datasource.DataSource {
	return &ManagedOmniGatewayDataSource{}
}

func (d *ManagedOmniGatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_omni_gateways"
}

func (d *ManagedOmniGatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all managed Omni Gateway instances in the given environment.",
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
				Description: "List of managed Omni Gateway instances.",
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
							Description: "The target ID the gateway is deployed to. A private space UUID " +
								"or, for a CloudHub 2.0 shared space, a region slug (e.g. 'cloudhub-us-east-1').",
							Computed: true,
						},
						"target_type": schema.StringAttribute{
							Description: "The type of the deployment target: 'private-space' or 'shared-space'. " +
								"Resolved from the deployment-targets list; empty if the target could not be resolved.",
							Computed: true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the gateway (e.g. APPLYING while provisioning, RUNNING once ready).",
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

func (d *ManagedOmniGatewayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	gwClient, err := apimanagement.NewManagedOmniGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Managed Omni Gateway Client",
			"An unexpected error occurred when creating the Managed Omni Gateway client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = gwClient
}

func (d *ManagedOmniGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ManagedOmniGatewayDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	gateways, err := d.client.ListManagedOmniGateways(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing managed Omni Gateways",
			"Could not list managed Omni Gateways for environment "+envID+": "+err.Error(),
		)
		return
	}

	// The list endpoint does not return target_type. Enrich it with a single
	// deployment-targets lookup (targetId -> type). Best-effort: if the lookup
	// fails, leave target_type empty rather than failing the whole data source.
	targetTypeByID := map[string]string{}
	if targets, terr := d.client.ListTargets(ctx, orgID); terr != nil {
		tflog.Warn(ctx, "Could not list deployment targets to resolve target_type; leaving it empty",
			map[string]interface{}{"error": terr.Error()})
	} else {
		for _, t := range targets {
			targetTypeByID[t.ID] = t.Type
		}
	}

	data.ID = types.StringValue(orgID + "/" + envID)
	data.OrganizationID = types.StringValue(orgID)
	data.Gateways = make([]ManagedOmniGatewayItemModel, len(gateways))

	for i, gw := range gateways {
		data.Gateways[i] = ManagedOmniGatewayItemModel{
			ID:          types.StringValue(gw.ID),
			Name:        types.StringValue(gw.Name),
			TargetID:    types.StringValue(gw.TargetID),
			TargetType:  types.StringValue(targetTypeByID[gw.TargetID]),
			Status:      types.StringValue(gw.Status),
			DateCreated: types.StringValue(gw.DateCreated),
			LastUpdated: types.StringValue(gw.LastUpdated),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
