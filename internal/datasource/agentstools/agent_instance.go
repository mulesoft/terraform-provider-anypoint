package agentstools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

var (
	_ datasource.DataSource              = &AgentInstanceDataSource{}
	_ datasource.DataSourceWithConfigure = &AgentInstanceDataSource{}
)

// AgentInstanceDataSource lists all agent instances in an environment.
type AgentInstanceDataSource struct {
	client *agentstools.AgentInstanceClient
}

type AgentInstanceDataSourceModel struct {
	ID             types.String             `tfsdk:"id"`
	OrganizationID types.String             `tfsdk:"organization_id"`
	EnvironmentID  types.String             `tfsdk:"environment_id"`
	Instances      []AgentInstanceItemModel `tfsdk:"instances"`
}

type AgentInstanceItemModel struct {
	ID                        types.String `tfsdk:"id"`
	AssetID                   types.String `tfsdk:"asset_id"`
	AssetVersion              types.String `tfsdk:"asset_version"`
	ProductVersion            types.String `tfsdk:"product_version"`
	GroupID                   types.String `tfsdk:"group_id"`
	Technology                types.String `tfsdk:"technology"`
	InstanceLabel             types.String `tfsdk:"instance_label"`
	Status                    types.String `tfsdk:"status"`
	EndpointURI               types.String `tfsdk:"endpoint_uri"`
	AutodiscoveryInstanceName types.String `tfsdk:"autodiscovery_instance_name"`
}

func NewAgentInstanceDataSource() datasource.DataSource {
	return &AgentInstanceDataSource{}
}

func (d *AgentInstanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_instances"
}

func (d *AgentInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all agent instances registered in API Manager for the given environment.",
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
				Description: "The environment ID to list agent instances from.",
				Required:    true,
			},
			"instances": schema.ListNestedAttribute{
				Description: "List of agent instances.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The numeric ID of the agent instance.",
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
						"product_version": schema.StringAttribute{
							Description: "The product version.",
							Computed:    true,
						},
						"group_id": schema.StringAttribute{
							Description: "The Exchange group (organization) ID.",
							Computed:    true,
						},
						"technology": schema.StringAttribute{
							Description: "The gateway technology (flexGateway, mule4, etc.).",
							Computed:    true,
						},
						"instance_label": schema.StringAttribute{
							Description: "The label of the agent instance.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the agent instance.",
							Computed:    true,
						},
						"endpoint_uri": schema.StringAttribute{
							Description: "The endpoint URI for the agent instance.",
							Computed:    true,
						},
						"autodiscovery_instance_name": schema.StringAttribute{
							Description: "The autodiscovery instance name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *AgentInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	agentClient, err := agentstools.NewAgentInstanceClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Agent Instance Client",
			"An unexpected error occurred when creating the Agent Instance client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = agentClient
}

func (d *AgentInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentInstanceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	instances, err := d.client.ListAgentInstances(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing agent instances",
			"Could not list agent instances for environment "+envID+": "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(orgID + "/" + envID)
	data.OrganizationID = types.StringValue(orgID)
	data.Instances = make([]AgentInstanceItemModel, 0, len(instances))

	for _, inst := range instances {
		data.Instances = append(data.Instances, mapAgentInstanceToItemModel(inst))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapAgentInstanceToItemModel converts a client AgentInstance to the datasource item model.
func mapAgentInstanceToItemModel(inst agentstools.AgentInstance) AgentInstanceItemModel {
	endpointURI := types.StringNull()
	if inst.EndpointURI != "" {
		endpointURI = types.StringValue(inst.EndpointURI)
	} else if inst.Endpoint != nil && inst.Endpoint.URI != nil {
		endpointURI = types.StringValue(*inst.Endpoint.URI)
	}

	autodiscovery := types.StringNull()
	if inst.AutodiscoveryInstanceName != "" {
		autodiscovery = types.StringValue(inst.AutodiscoveryInstanceName)
	}

	instanceLabel := types.StringNull()
	if inst.InstanceLabel != "" {
		instanceLabel = types.StringValue(inst.InstanceLabel)
	}

	return AgentInstanceItemModel{
		ID:                        types.StringValue(strconv.Itoa(inst.ID)),
		AssetID:                   types.StringValue(inst.AssetID),
		AssetVersion:              types.StringValue(inst.AssetVersion),
		ProductVersion:            types.StringValue(inst.ProductVersion),
		GroupID:                   types.StringValue(inst.GroupID),
		Technology:                types.StringValue(inst.Technology),
		InstanceLabel:             instanceLabel,
		Status:                    types.StringValue(inst.Status),
		EndpointURI:               endpointURI,
		AutodiscoveryInstanceName: autodiscovery,
	}
}
