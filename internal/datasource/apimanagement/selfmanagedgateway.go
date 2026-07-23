package apimanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &SelfManagedGatewayDataSource{}
	_ datasource.DataSourceWithConfigure = &SelfManagedGatewayDataSource{}
)

// SelfManagedGatewayDataSource lists all self-managed (connected-mode) Flex gateways
// that have registered in an environment.
type SelfManagedGatewayDataSource struct {
	client *apimanagement.SelfManagedGatewayClient
}

type SelfManagedGatewayDataSourceModel struct {
	ID             types.String                  `tfsdk:"id"`
	OrganizationID types.String                  `tfsdk:"organization_id"`
	EnvironmentID  types.String                  `tfsdk:"environment_id"`
	IncludeDeleted types.Bool                    `tfsdk:"include_deleted"`
	Gateways       []SelfManagedGatewayItemModel `tfsdk:"gateways"`
}

type SelfManagedGatewayItemModel struct {
	ID         types.String                     `tfsdk:"id"`
	Name       types.String                     `tfsdk:"name"`
	Status     types.String                     `tfsdk:"status"`
	LastUpdate types.String                     `tfsdk:"last_update"`
	Tags       []types.String                   `tfsdk:"tags"`
	Replicas   []SelfManagedGatewayReplicaModel `tfsdk:"replicas"`
}

type SelfManagedGatewayReplicaModel struct {
	Status                     types.String   `tfsdk:"status"`
	Count                      types.Int64    `tfsdk:"count"`
	CertificateExpirationDates []types.String `tfsdk:"certificate_expiration_dates"`
}

func NewSelfManagedGatewayDataSource() datasource.DataSource {
	return &SelfManagedGatewayDataSource{}
}

func (d *SelfManagedGatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_self_managed_gateways"
}

func (d *SelfManagedGatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all self-managed (connected-mode) Flex gateways that have registered in the given environment.",
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
			"include_deleted": schema.BoolAttribute{
				Description: "Whether to include soft-deleted gateways. Deleting a self-managed " +
					"gateway is a soft-delete: the object lingers in the platform list forever with " +
					"status DELETED. By default these tombstones are filtered out; set this to `true` " +
					"to include them (e.g. for auditing).",
				Optional: true,
			},
			"gateways": schema.ListNestedAttribute{
				Description: "List of self-managed gateways that have registered. Soft-deleted " +
					"(status DELETED) gateways are excluded unless `include_deleted` is true.",
				Computed: true,
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
						"status": schema.StringAttribute{
							Description: "The current status of the gateway (e.g. CONNECTED, DISCONNECTED, DELETED).",
							Computed:    true,
						},
						"last_update": schema.StringAttribute{
							Description: "Timestamp of the gateway's last status update (RFC 3339).",
							Computed:    true,
						},
						"tags": schema.ListAttribute{
							Description: "Tags associated with the gateway.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"replicas": schema.ListNestedAttribute{
							Description: "Replica (runtime instance) status buckets reported by the gateway. " +
								"The platform reports one entry per connectivity status with a running count.",
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{
										Description: "The connectivity status of this replica bucket (e.g. CONNECTED, DISCONNECTED).",
										Computed:    true,
									},
									"count": schema.Int64Attribute{
										Description: "The number of replicas currently in this status.",
										Computed:    true,
									},
									"certificate_expiration_dates": schema.ListAttribute{
										Description: "Certificate expiration timestamps reported by replicas in this bucket.",
										Computed:    true,
										ElementType: types.StringType,
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

func (d *SelfManagedGatewayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	gwClient, err := apimanagement.NewSelfManagedGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Self-Managed Gateway Client",
			"An unexpected error occurred when creating the Self-Managed Gateway client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = gwClient
}

func (d *SelfManagedGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SelfManagedGatewayDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	gateways, err := d.client.ListSelfManagedGateways(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing self-managed gateways",
			"Could not list self-managed gateways for environment "+envID+": "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(orgID + "/" + envID)
	data.OrganizationID = types.StringValue(orgID)

	// Soft-delete tombstones (status DELETED) linger in the list forever. Filter them out by
	// default so callers see only live gateways; include them only when explicitly requested.
	includeDeleted := data.IncludeDeleted.ValueBool()

	// Build with an explicit non-nil slice so an empty result serializes as [] not null.
	data.Gateways = make([]SelfManagedGatewayItemModel, 0, len(gateways))
	for _, gw := range gateways {
		if !includeDeleted && strings.EqualFold(gw.Status, apimanagement.SelfManagedGatewayStatusDeleted) {
			continue
		}

		tags := make([]types.String, 0, len(gw.Tags))
		for _, t := range gw.Tags {
			tags = append(tags, types.StringValue(t))
		}

		replicas := make([]SelfManagedGatewayReplicaModel, 0, len(gw.Replicas))
		for _, rep := range gw.Replicas {
			certDates := make([]types.String, 0, len(rep.CertificateExpirationDates))
			for _, cd := range rep.CertificateExpirationDates {
				certDates = append(certDates, types.StringValue(cd))
			}
			replicas = append(replicas, SelfManagedGatewayReplicaModel{
				Status:                     types.StringValue(rep.Status),
				Count:                      types.Int64Value(rep.Count),
				CertificateExpirationDates: certDates,
			})
		}

		data.Gateways = append(data.Gateways, SelfManagedGatewayItemModel{
			ID:         types.StringValue(gw.ID),
			Name:       types.StringValue(gw.Name),
			Status:     types.StringValue(gw.Status),
			LastUpdate: types.StringValue(gw.LastUpdate),
			Tags:       tags,
			Replicas:   replicas,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
