package apimanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &ManagedFlexGatewaySingleDataSource{}
	_ datasource.DataSourceWithConfigure = &ManagedFlexGatewaySingleDataSource{}
)

// ManagedFlexGatewaySingleDataSource fetches a single managed Omni Gateway by ID.
type ManagedFlexGatewaySingleDataSource struct {
	client *apimanagement.ManagedFlexGatewayClient
}

type ManagedFlexGatewaySingleDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	Name           types.String `tfsdk:"name"`
	TargetID       types.String `tfsdk:"target_id"`
	TargetName     types.String `tfsdk:"target_name"`
	TargetType     types.String `tfsdk:"target_type"`
	RuntimeVersion types.String `tfsdk:"runtime_version"`
	ReleaseChannel types.String `tfsdk:"release_channel"`
	Size           types.String `tfsdk:"size"`
	Status         types.String `tfsdk:"status"`
	DesiredStatus  types.String `tfsdk:"desired_status"`
	StatusMessage  types.String `tfsdk:"status_message"`
	DateCreated    types.String `tfsdk:"date_created"`
	LastUpdated    types.String `tfsdk:"last_updated"`
	APILimit       types.Int64  `tfsdk:"api_limit"`
	Ingress        types.Object `tfsdk:"ingress"`
	Properties     types.Object `tfsdk:"properties"`
	Logging        types.Object `tfsdk:"logging"`
	PortConfig     types.Object `tfsdk:"port_configuration"`
}

var (
	dsSingleIngressAttrTypes = map[string]attr.Type{
		"public_url":          types.StringType,
		"internal_urls":       types.ListType{ElemType: types.StringType},
		"forward_ssl_session": types.BoolType,
		"last_mile_security":  types.BoolType,
	}
	dsSinglePropertiesAttrTypes = map[string]attr.Type{
		"upstream_response_timeout": types.Int64Type,
		"connection_idle_timeout":   types.Int64Type,
	}
	dsSingleLoggingAttrTypes = map[string]attr.Type{
		"level":        types.StringType,
		"forward_logs": types.BoolType,
	}
	dsSinglePortEntryAttrTypes = map[string]attr.Type{
		"port":     types.Int64Type,
		"protocol": types.StringType,
	}
	dsSinglePortConfigAttrTypes = map[string]attr.Type{
		"ingress": types.ObjectType{AttrTypes: dsSinglePortEntryAttrTypes},
		"egress":  types.ObjectType{AttrTypes: dsSinglePortEntryAttrTypes},
	}
)

func NewManagedFlexGatewaySingleDataSource() datasource.DataSource {
	return &ManagedFlexGatewaySingleDataSource{}
}

func (d *ManagedFlexGatewaySingleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_omnigateway"
}

func (d *ManagedFlexGatewaySingleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	portEntryAttrs := map[string]schema.Attribute{
		"port": schema.Int64Attribute{
			Computed:    true,
			Description: "The port number.",
		},
		"protocol": schema.StringAttribute{
			Computed:    true,
			Description: "The protocol (e.g. TCP).",
		},
	}

	resp.Schema = schema.Schema{
		Description: "Fetches the full details of a single managed Omni Gateway by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The managed Omni Gateway ID.",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. Defaults to the provider credentials organization.",
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "The environment ID where the gateway is deployed.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the gateway.",
			},
			"target_id": schema.StringAttribute{
				Computed:    true,
				Description: "The target (private space) ID.",
			},
			"target_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the target (private space).",
			},
			"target_type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the target (e.g. private-space).",
			},
			"runtime_version": schema.StringAttribute{
				Computed:    true,
				Description: "The runtime version of the gateway.",
			},
			"release_channel": schema.StringAttribute{
				Computed:    true,
				Description: "The release channel (lts or edge).",
			},
			"size": schema.StringAttribute{
				Computed:    true,
				Description: "The gateway size (small, large).",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the gateway (e.g. APPLIED).",
			},
			"desired_status": schema.StringAttribute{
				Computed:    true,
				Description: "The desired status of the gateway (e.g. STARTED).",
			},
			"status_message": schema.StringAttribute{
				Computed:    true,
				Description: "Additional status message from the gateway.",
			},
			"date_created": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the gateway was created.",
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last update to the gateway.",
			},
			"api_limit": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum number of APIs that can be deployed to this gateway.",
			},
			"ingress": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Ingress network configuration.",
				Attributes: map[string]schema.Attribute{
					"public_url": schema.StringAttribute{
						Computed:    true,
						Description: "The primary public URL.",
					},
					"internal_urls": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "All internal URLs (the API returns these as comma-separated).",
					},
					"forward_ssl_session": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether SSL session forwarding is enabled.",
					},
					"last_mile_security": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether last-mile security (TLS to upstream) is enabled.",
					},
				},
			},
			"properties": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Runtime properties.",
				Attributes: map[string]schema.Attribute{
					"upstream_response_timeout": schema.Int64Attribute{
						Computed:    true,
						Description: "Upstream response timeout in seconds.",
					},
					"connection_idle_timeout": schema.Int64Attribute{
						Computed:    true,
						Description: "Connection idle timeout in seconds.",
					},
				},
			},
			"logging": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Logging configuration.",
				Attributes: map[string]schema.Attribute{
					"level": schema.StringAttribute{
						Computed:    true,
						Description: "Log level (debug, info, warn, error).",
					},
					"forward_logs": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether logs are forwarded to Anypoint Monitoring.",
					},
				},
			},
			"port_configuration": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Port configuration for ingress and egress traffic.",
				Attributes: map[string]schema.Attribute{
					"ingress": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Ingress port settings.",
						Attributes:  portEntryAttrs,
					},
					"egress": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Egress port settings.",
						Attributes:  portEntryAttrs,
					},
				},
			},
		},
	}
}

func (d *ManagedFlexGatewaySingleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	gwClient, err := apimanagement.NewManagedFlexGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Managed Omni Gateway Client", err.Error())
		return
	}
	d.client = gwClient
}

func (d *ManagedFlexGatewaySingleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ManagedFlexGatewaySingleDataSourceModel
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

	gw, err := d.client.GetManagedFlexGateway(ctx, orgID, envID, gatewayID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading managed Omni Gateway", err.Error())
		return
	}

	data.OrganizationID = types.StringValue(orgID)
	data.Name = types.StringValue(gw.Name)
	data.TargetID = types.StringValue(gw.TargetID)
	data.TargetName = types.StringValue(gw.TargetName)
	data.TargetType = types.StringValue(gw.TargetType)
	data.RuntimeVersion = types.StringValue(gw.RuntimeVersion)
	data.ReleaseChannel = types.StringValue(gw.ReleaseChannel)
	data.Size = types.StringValue(gw.Size)
	data.Status = types.StringValue(gw.Status)
	data.DesiredStatus = types.StringValue(gw.DesiredStatus)
	data.StatusMessage = types.StringValue(gw.StatusMessage)
	data.DateCreated = types.StringValue(gw.DateCreated)
	data.LastUpdated = types.StringValue(gw.LastUpdated)
	data.APILimit = types.Int64Value(int64(gw.APILimit))

	// ingress — internalUrl from the API is comma-separated
	internalRaw := gw.Configuration.Ingress.InternalURL
	var internalElems []attr.Value
	for _, u := range strings.Split(internalRaw, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			internalElems = append(internalElems, types.StringValue(u))
		}
	}
	internalURLsList, _ := types.ListValue(types.StringType, internalElems)
	ingressObj, _ := types.ObjectValue(dsSingleIngressAttrTypes, map[string]attr.Value{
		"public_url":          types.StringValue(gw.Configuration.Ingress.PublicURL),
		"internal_urls":       internalURLsList,
		"forward_ssl_session": types.BoolValue(gw.Configuration.Ingress.ForwardSSLSession),
		"last_mile_security":  types.BoolValue(gw.Configuration.Ingress.LastMileSecurity),
	})
	data.Ingress = ingressObj

	// properties
	propertiesObj, _ := types.ObjectValue(dsSinglePropertiesAttrTypes, map[string]attr.Value{
		"upstream_response_timeout": types.Int64Value(int64(gw.Configuration.Properties.UpstreamResponseTimeout)),
		"connection_idle_timeout":   types.Int64Value(int64(gw.Configuration.Properties.ConnectionIdleTimeout)),
	})
	data.Properties = propertiesObj

	// logging
	loggingObj, _ := types.ObjectValue(dsSingleLoggingAttrTypes, map[string]attr.Value{
		"level":        types.StringValue(gw.Configuration.Logging.Level),
		"forward_logs": types.BoolValue(gw.Configuration.Logging.ForwardLogs),
	})
	data.Logging = loggingObj

	// port_configuration
	if gw.PortConfig != nil {
		ingressPortObj, _ := types.ObjectValue(dsSinglePortEntryAttrTypes, map[string]attr.Value{
			"port":     types.Int64Value(int64(gw.PortConfig.Ingress.Port)),
			"protocol": types.StringValue(gw.PortConfig.Ingress.Protocol),
		})
		egressPortObj, _ := types.ObjectValue(dsSinglePortEntryAttrTypes, map[string]attr.Value{
			"port":     types.Int64Value(int64(gw.PortConfig.Egress.Port)),
			"protocol": types.StringValue(gw.PortConfig.Egress.Protocol),
		})
		portConfigObj, _ := types.ObjectValue(dsSinglePortConfigAttrTypes, map[string]attr.Value{
			"ingress": ingressPortObj,
			"egress":  egressPortObj,
		})
		data.PortConfig = portConfigObj
	} else {
		data.PortConfig = types.ObjectNull(dsSinglePortConfigAttrTypes)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
