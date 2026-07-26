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
	_ datasource.DataSource              = &SelfManagedGatewaySingleDataSource{}
	_ datasource.DataSourceWithConfigure = &SelfManagedGatewaySingleDataSource{}
)

// SelfManagedGatewaySingleDataSource is the SINGULAR self-managed (connected-mode) Flex
// gateway data source. It is the read-only twin of the anypoint_self_managed_gateway
// resource: it looks up ONE registered gateway by its id and surfaces the full attribute
// set. Unlike the plural anypoint_self_managed_gateways data source — which lists every
// gateway and (by default) filters out soft-deleted tombstones — this singular data source
// returns exactly the gateway you ask for by id, INCLUDING one that has been soft-deleted
// (status DELETED lingers forever), so it can be used to inspect a tombstone. It also
// exposes the `versions` array, which the plural list does not surface.
type SelfManagedGatewaySingleDataSource struct {
	client *apimanagement.SelfManagedGatewayClient
}

// SelfManagedGatewaySingleDataSourceModel describes the singular data source data model.
// It reuses SelfManagedGatewayReplicaModel (defined alongside the plural data source) for
// the per-replica status buckets so both data sources report replicas identically.
type SelfManagedGatewaySingleDataSourceModel struct {
	ID             types.String                           `tfsdk:"id"`
	OrganizationID types.String                           `tfsdk:"organization_id"`
	EnvironmentID  types.String                           `tfsdk:"environment_id"`
	Name           types.String                           `tfsdk:"name"`
	Status         types.String                           `tfsdk:"status"`
	LastUpdate     types.String                           `tfsdk:"last_update"`
	Tags           []types.String                         `tfsdk:"tags"`
	Versions       []types.String                         `tfsdk:"versions"`
	Replicas       []SelfManagedGatewayReplicaModel       `tfsdk:"replicas"`
	ReplicaDetails []SelfManagedGatewayReplicaDetailModel `tfsdk:"replica_details"`
}

// SelfManagedGatewayReplicaDetailModel is the RICH per-replica detail surfaced by the
// singular data source — one entry per concrete runtime node, backing the Runtime Manager
// "Replicas" tab. It is distinct from SelfManagedGatewayReplicaModel (the coarse
// status-bucket summary in `replicas`).
type SelfManagedGatewayReplicaDetailModel struct {
	ID                        types.String `tfsdk:"id"`
	NodeID                    types.String `tfsdk:"node_id"`
	Name                      types.String `tfsdk:"name"`
	TargetID                  types.String `tfsdk:"target_id"`
	GatewayVersion            types.String `tfsdk:"gateway_version"`
	Status                    types.String `tfsdk:"status"`
	ConnectedAt               types.String `tfsdk:"connected_at"`
	DisconnectedAt            types.String `tfsdk:"disconnected_at"`
	ConfigurationStatus       types.String `tfsdk:"configuration_status"`
	ConfigurationMessage      types.String `tfsdk:"configuration_message"`
	CertificateExpirationDate types.String `tfsdk:"certificate_expiration_date"`
	Cid                       types.String `tfsdk:"cid"`
	Provider                  types.String `tfsdk:"provider"`
}

func NewSelfManagedGatewaySingleDataSource() datasource.DataSource {
	return &SelfManagedGatewaySingleDataSource{}
}

func (d *SelfManagedGatewaySingleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	// The SINGULAR data source shares the resource's type name; the plural list DS is
	// anypoint_self_managed_gateways (with the trailing 's').
	resp.TypeName = req.ProviderTypeName + "_self_managed_gateway"
}

func (d *SelfManagedGatewaySingleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single self-managed (connected-mode) Flex gateway by its ID. " +
			"Unlike the plural anypoint_self_managed_gateways data source — which lists every " +
			"gateway and by default hides soft-deleted (status DELETED) tombstones — this singular " +
			"data source returns exactly the gateway requested by id (including a tombstone) and " +
			"additionally surfaces the reported runtime `versions`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the self-managed gateway to fetch.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider credentials organization.",
				Optional:    true,
				Computed:    true,
			},
			"environment_id": schema.StringAttribute{
				Description: "The environment ID the gateway is registered in.",
				Required:    true,
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
			"versions": schema.ListAttribute{
				Description: "Runtime versions reported by the gateway's replicas. Empty until a " +
					"replica reports a version. Not exposed by the plural data source.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"replicas": schema.ListNestedAttribute{
				Description: "Replica (runtime instance) status buckets reported by the gateway. " +
					"The platform reports one entry per connectivity status with a running count. " +
					"This is the coarse summary; for per-node detail use `replica_details`.",
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
			"replica_details": schema.ListNestedAttribute{
				Description: "Rich per-replica detail — one entry per concrete Flex runtime node " +
					"registered against this gateway, as shown in the Runtime Manager \"Replicas\" tab. " +
					"Unlike the coarse `replicas` status buckets, each entry here identifies an individual " +
					"node with its version, connect/disconnect timestamps, per-node certificate expiry, " +
					"and configuration-sync status.",
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of this replica.",
							Computed:    true,
						},
						"node_id": schema.StringAttribute{
							Description: "The node identifier of this replica (typically equal to `id`).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The replica's reported name (e.g. \"d6c016e2693e.default\").",
							Computed:    true,
						},
						"target_id": schema.StringAttribute{
							Description: "The gateway (deployment target) id this replica belongs to.",
							Computed:    true,
						},
						"gateway_version": schema.StringAttribute{
							Description: "The Flex runtime version this replica is running.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The connectivity status of this replica (e.g. CONNECTED, DISCONNECTED).",
							Computed:    true,
						},
						"connected_at": schema.StringAttribute{
							Description: "Timestamp when this replica last connected (RFC 3339). Empty if never connected.",
							Computed:    true,
						},
						"disconnected_at": schema.StringAttribute{
							Description: "Timestamp when this replica last disconnected (RFC 3339). Empty while connected.",
							Computed:    true,
						},
						"configuration_status": schema.StringAttribute{
							Description: "The configuration-sync status of this replica (e.g. UP_TO_DATE).",
							Computed:    true,
						},
						"configuration_message": schema.StringAttribute{
							Description: "A human-readable message accompanying the configuration status. " +
								"Empty when the replica configuration is up to date.",
							Computed: true,
						},
						"certificate_expiration_date": schema.StringAttribute{
							Description: "This replica's client-certificate expiration timestamp (RFC 3339).",
							Computed:    true,
						},
						"cid": schema.StringAttribute{
							Description: "The internal connection identifier reported for this replica.",
							Computed:    true,
						},
						"provider": schema.StringAttribute{
							Description: "The runtime provider reported for this replica (e.g. RR).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SelfManagedGatewaySingleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SelfManagedGatewaySingleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SelfManagedGatewaySingleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	gatewayID := data.ID.ValueString()

	gw, err := d.client.GetSelfManagedGateway(ctx, orgID, envID, gatewayID)
	if err != nil {
		// A singular data source pointing at a non-existent id is a configuration error,
		// not empty state — surface it so the plan fails loudly rather than silently.
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				"Self-managed gateway not found",
				fmt.Sprintf("No self-managed gateway with id %q exists in environment %q.", gatewayID, envID),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading self-managed gateway",
			"Could not read self-managed gateway "+gatewayID+": "+err.Error(),
		)
		return
	}

	data.OrganizationID = types.StringValue(orgID)
	data.Name = types.StringValue(gw.Name)
	data.Status = types.StringValue(gw.Status)
	data.LastUpdate = types.StringValue(gw.LastUpdate)

	// Build with explicit non-nil slices so empty results serialize as [] not null.
	tags := make([]types.String, 0, len(gw.Tags))
	for _, t := range gw.Tags {
		tags = append(tags, types.StringValue(t))
	}
	data.Tags = tags

	versions := make([]types.String, 0, len(gw.Versions))
	for _, v := range gw.Versions {
		versions = append(versions, types.StringValue(v))
	}
	data.Versions = versions

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
	data.Replicas = replicas

	// Fetch the RICH per-replica detail from the dedicated /replicas endpoint (the data behind
	// the Runtime Manager "Replicas" tab). The gateway itself already resolved above, so a
	// NotFound here is treated as "no queryable replica detail" (e.g. a soft-deleted tombstone,
	// which this data source explicitly supports inspecting) rather than a hard error; any other
	// failure is surfaced so incomplete data never passes silently.
	details, err := d.client.GetSelfManagedGatewayReplicas(ctx, orgID, envID, gatewayID)
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error reading self-managed gateway replica details",
			"Could not read replica details for self-managed gateway "+gatewayID+": "+err.Error(),
		)
		return
	}
	replicaDetails := make([]SelfManagedGatewayReplicaDetailModel, 0, len(details))
	for _, rd := range details {
		replicaDetails = append(replicaDetails, SelfManagedGatewayReplicaDetailModel{
			ID:                        types.StringValue(rd.ID),
			NodeID:                    types.StringValue(rd.NodeID),
			Name:                      types.StringValue(rd.Name),
			TargetID:                  types.StringValue(rd.TargetID),
			GatewayVersion:            types.StringValue(rd.GatewayVersion),
			Status:                    types.StringValue(rd.Status),
			ConnectedAt:               stringOrEmpty(rd.ConnectedAt),
			DisconnectedAt:            stringOrEmpty(rd.DisconnectedAt),
			ConfigurationStatus:       types.StringValue(rd.ConfigurationStatus.Status),
			ConfigurationMessage:      stringOrEmpty(rd.ConfigurationStatus.Message),
			CertificateExpirationDate: stringOrEmpty(rd.CertificateExpirationDate),
			Cid:                       types.StringValue(rd.Cid),
			Provider:                  types.StringValue(rd.Provider),
		})
	}
	data.ReplicaDetails = replicaDetails

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// stringOrEmpty converts a nullable API string field to a types.String, mapping a nil pointer
// to the empty string so a Computed attribute is never left unknown/null in state.
func stringOrEmpty(s *string) types.String {
	if s == nil {
		return types.StringValue("")
	}
	return types.StringValue(*s)
}
