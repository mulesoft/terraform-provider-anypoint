package agentstools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

var (
	_ resource.Resource                   = &MCPServerResource{}
	_ resource.ResourceWithConfigure      = &MCPServerResource{}
	_ resource.ResourceWithImportState    = &MCPServerResource{}
	_ resource.ResourceWithValidateConfig = &MCPServerResource{}
)

type MCPServerResource struct {
	client *agentstools.MCPServerClient
}

// --- Terraform State Models ---

type MCPServerResourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Technology       types.String `tfsdk:"technology"`
	ProviderID       types.String `tfsdk:"provider_id"`
	InstanceLabel    types.String `tfsdk:"instance_label"`
	ApprovalMethod   types.String `tfsdk:"approval_method"`
	Status           types.String `tfsdk:"status"`
	AssetID          types.String `tfsdk:"asset_id"`
	AssetVersion     types.String `tfsdk:"asset_version"`
	ProductVersion   types.String `tfsdk:"product_version"`
	ConsumerEndpoint types.String `tfsdk:"consumer_endpoint"`
	UpstreamURI      types.String `tfsdk:"upstream_uri"`
	UpstreamID       types.String `tfsdk:"upstream_id"`

	GatewayID types.String `tfsdk:"gateway_id"`

	Spec       *SpecModel   `tfsdk:"spec"`
	Endpoint   types.Object `tfsdk:"endpoint"`
	Deployment types.Object `tfsdk:"deployment"`
	Routing    types.List   `tfsdk:"routing"`
}

func NewMCPServerResource() resource.Resource {
	return &MCPServerResource{}
}

func (r *MCPServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_server"
}

func (r *MCPServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an MCP server in Anypoint API Manager. An MCP server represents " +
			"an MCP server specification deployed to a Flex Gateway target with routing rules and upstream backends.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric identifier of the MCP server (stored as string for Terraform compatibility).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: "The environment ID where the MCP server will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"technology": schema.StringAttribute{
				Description: "The gateway technology. Valid values: 'flexGateway', 'mule4', 'serviceMesh'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("flexGateway"),
				Validators: []validator.String{
					stringvalidator.OneOf("flexGateway", "mule4", "serviceMesh"),
				},
			},
			"provider_id": schema.StringAttribute{
				Description: "The identity provider ID for the MCP server.",
				Optional:    true,
			},
			"instance_label": schema.StringAttribute{
				Description: "A human-readable label for this MCP server.",
				Optional:    true,
			},
			"approval_method": schema.StringAttribute{
				Description: "Client approval method. Valid values: 'manual', 'automatic'. Defaults to null (no approval required).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("manual", "automatic"),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the MCP server.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_id": schema.StringAttribute{
				Description: "The Exchange asset ID (computed from MCP server response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_version": schema.StringAttribute{
				Description: "The Exchange asset version (computed from MCP server response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_version": schema.StringAttribute{
				Description: "The product version (computed from MCP server response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"spec": schema.SingleNestedAttribute{
				Description: "The Exchange asset specification backing this MCP server.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"asset_id": schema.StringAttribute{
						Description: "The Exchange asset ID.",
						Required:    true,
					},
					"group_id": schema.StringAttribute{
						Description: "The Exchange group (organization) ID.",
						Required:    true,
					},
					"version": schema.StringAttribute{
						Description: "The asset version.",
						Required:    true,
					},
				},
			},
			"endpoint": schema.SingleNestedAttribute{
				Description: "Endpoint / proxy configuration for the MCP server.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Description: "Deployment type. Valid values: 'HY' (hybrid), 'CH' (CloudHub), 'RF' (Runtime Fabric).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("HY"),
						Validators: []validator.String{
							stringvalidator.OneOf("HY", "CH", "CH2", "RF"),
						},
					},
					"type": schema.StringAttribute{
						Description: "Endpoint protocol type. For MCP servers, this is 'mcp'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("mcp"),
						Validators: []validator.String{
							stringvalidator.OneOf("mcp"),
						},
					},
					"base_path": schema.StringAttribute{
						Description: "MCP server base path for FlexGateway (e.g. 'my-mcp-server'). The provider constructs the full proxy URI as http://0.0.0.0:8081/<base_path>. Required when technology='flexGateway'. Mutually exclusive with 'uri'.",
						Optional:    true,
					},
					"uri": schema.StringAttribute{
						Description: "Direct implementation URI for Mule4 or other technologies (e.g. 'http://www.google.com'). Required when technology='mule4'. Mutually exclusive with 'base_path'.",
						Optional:    true,
					},
					"response_timeout": schema.Int64Attribute{
						Description: "Response timeout in milliseconds.",
						Optional:    true,
					},
				},
			},
			"consumer_endpoint": schema.StringAttribute{
				Description: "Consumer-facing endpoint URI (the public URL clients use to reach the MCP server). Maps to top-level endpointUri in the MCP server. For MCP, this is the proxy_uri that clients connect to.",
				Optional:    true,
			},
			"upstream_uri": schema.StringAttribute{
				Description: "Shorthand for a single-upstream routing configuration. " +
					"When set, the provider constructs routing as [{upstreams: [{weight: 100, uri: <value>}]}]. " +
					"Mutually exclusive with the 'routing' block. For MCP servers, this is typically the upstream MCP server URI that the proxy_uri forwards to.",
				Optional: true,
			},
			"upstream_id": schema.StringAttribute{
				Description: "The server-assigned upstream ID for the first upstream. " +
					"Populated automatically after creation. Use this to reference the upstream in outbound policy upstream_ids.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway_id": schema.StringAttribute{
				Description: "The Flex Gateway UUID. When provided, the deployment block is auto-populated " +
					"by fetching gateway details (target_id, target_name, gateway_version) from the Gateway Manager MCP server. " +
					"Mutually exclusive with specifying a full deployment block.",
				Optional: true,
			},
			"deployment": schema.SingleNestedAttribute{
				Description: "Deployment target configuration. Auto-populated when gateway_id is set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"environment_id": schema.StringAttribute{
						Description: "The environment ID for deployment (usually matches the top-level environment_id).",
						Optional:    true,
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Deployment type. Valid values: 'HY', 'CH', 'RF'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("HY"),
					},
					"expected_status": schema.StringAttribute{
						Description: "Expected deployment status. Valid values: 'deployed', 'undeployed'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("deployed"),
					},
					"overwrite": schema.BoolAttribute{
						Description: "Whether to overwrite an existing deployment.",
						Optional:    true,
					},
					"target_id": schema.StringAttribute{
						Description: "The target gateway ID to deploy to.",
						Optional:    true,
						Computed:    true,
					},
					"target_name": schema.StringAttribute{
						Description: "The target gateway name.",
						Optional:    true,
						Computed:    true,
					},
					"gateway_version": schema.StringAttribute{
						Description: "The Flex Gateway runtime version.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"routing": schema.ListNestedAttribute{
				Description: "Routing rules with weighted upstream backends. For MCP servers, upstreams typically point to the actual MCP server implementation URIs.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							Description: "A label for this route.",
							Optional:    true,
						},
						"rules": schema.SingleNestedAttribute{
							Description: "Match conditions for this route (methods, path, headers).",
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"methods": schema.StringAttribute{
									Description: "Pipe-separated HTTP methods (e.g. 'GET', 'POST|PUT').",
									Optional:    true,
								},
								"path": schema.StringAttribute{
									Description: "URL path pattern to match (e.g. '/api/*').",
									Optional:    true,
								},
								"host": schema.StringAttribute{
									Description: "Host header value to match.",
									Optional:    true,
								},
								"headers": schema.MapAttribute{
									Description: "Header key-value pairs to match.",
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
						"upstreams": schema.ListNestedAttribute{
							Description: "Weighted upstream backends for this route. For MCP servers, these are the actual MCP server implementation endpoints.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"weight": schema.Int64Attribute{
										Description: "Traffic weight percentage (0-100). Weights across upstreams should sum to 100. Defaults to 100.",
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(100),
									},
									"uri": schema.StringAttribute{
										Description: "The upstream backend URI. For MCP servers, this is the actual MCP server implementation URI that requests are forwarded to.",
										Required:    true,
									},
									"label": schema.StringAttribute{
										Description: "A label for this upstream.",
										Optional:    true,
									},
									"tls_context_id": schema.StringAttribute{
										Description: "TLS context for upstream connections. Format: 'secretGroupId/tlsContextId'.",
										Optional:    true,
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

func (r *MCPServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	mcpClient, err := agentstools.NewMCPServerClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create MCP Server Client",
			"An unexpected error occurred when creating the MCP Server client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = mcpClient
}

func (r *MCPServerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data MCPServerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasUpstreamURI := !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown()
	hasRouting := !data.Routing.IsNull() && !data.Routing.IsUnknown()

	if hasUpstreamURI && hasRouting {
		resp.Diagnostics.AddError(
			"Conflicting configuration",
			"'upstream_uri' and 'routing' are mutually exclusive. Use 'upstream_uri' for simple single-upstream routing, or 'routing' for advanced multi-route/multi-upstream configuration.",
		)
		return
	}

	if !hasRouting {
		return
	}

	var routes []RouteModel
	resp.Diagnostics.Append(data.Routing.ElementsAs(ctx, &routes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, route := range routes {
		if route.Upstreams.IsNull() || route.Upstreams.IsUnknown() {
			continue
		}

		var upstreams []UpstreamModel
		resp.Diagnostics.Append(route.Upstreams.ElementsAs(ctx, &upstreams, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// MCP servers only support a single upstream
		if len(upstreams) > 1 {
			routeLabel := fmt.Sprintf("routing[%d]", i)
			if !route.Label.IsNull() && !route.Label.IsUnknown() && route.Label.ValueString() != "" {
				routeLabel = fmt.Sprintf("routing[%d] (label=%q)", i, route.Label.ValueString())
			}
			resp.Diagnostics.AddError(
				"Multiple upstreams not supported",
				fmt.Sprintf("%s: mcp_server only supports a single upstream per route (got %d upstreams). "+
					"Multi-upstream routing with weighted traffic distribution is only available for api_instance resources. "+
					"For MCP servers, use 'upstream_uri' or a single upstream in the 'routing' block.",
					routeLabel, len(upstreams)),
			)
		}
	}
}

// --- CRUD ---

func (r *MCPServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MCPServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	if !data.GatewayID.IsNull() && !data.GatewayID.IsUnknown() && (data.Deployment.IsNull() || data.Deployment.IsUnknown()) {
		dep, err := r.resolveGatewayDeployment(ctx, orgID, envID, data.GatewayID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error resolving gateway_id", err.Error())
			return
		}
		data.Deployment = deploymentToObject(&DeploymentModel{
			EnvironmentID:  types.StringValue(dep.EnvironmentID),
			Type:           types.StringValue(dep.Type),
			ExpectedStatus: types.StringValue(dep.ExpectedStatus),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue(dep.TargetID),
			TargetName:     types.StringValue(dep.TargetName),
			GatewayVersion: types.StringValue(dep.GatewayVersion),
		})
	}

	createReq := r.expandCreateRequest(ctx, data)

	created, err := r.client.CreateMCPServer(ctx, orgID, envID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MCP server", "Could not create MCP server: "+err.Error())
		return
	}

	// POST response omits computed fields such as status. GET immediately so
	// flattenInstance receives a complete response and all computed attributes
	// are known in state — preventing Terraform from tainting the resource.
	instance, err := r.client.GetMCPServer(ctx, orgID, envID, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MCP server after create", "Could not read MCP server ID "+strconv.Itoa(created.ID)+": "+err.Error())
		return
	}

	gatewayID := data.GatewayID
	plannedRouting := data.Routing
	plannedEndpoint := data.Endpoint
	plannedDeployment := data.Deployment
	plannedConsumerEndpoint := data.ConsumerEndpoint
	plannedUpstreamURI := data.UpstreamURI
	r.flattenInstance(ctx, instance, &data, orgID, envID)
	data.GatewayID = gatewayID
	if !plannedUpstreamURI.IsNull() && !plannedUpstreamURI.IsUnknown() {
		data.UpstreamURI = plannedUpstreamURI
		data.Routing = types.ListNull(data.Routing.ElementType(ctx))
	} else if !plannedRouting.IsNull() {
		data.Routing = plannedRouting
	}
	if !plannedEndpoint.IsNull() && !plannedEndpoint.IsUnknown() {
		data.Endpoint = plannedEndpoint
	}
	data.Deployment = mergeDeploymentObjects(data.Deployment, plannedDeployment)
	if !plannedConsumerEndpoint.IsNull() && !plannedConsumerEndpoint.IsUnknown() {
		data.ConsumerEndpoint = plannedConsumerEndpoint
	}
	tflog.Trace(ctx, "created MCP server", map[string]interface{}{"id": instance.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MCPServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MCPServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	mcpID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Server ID", "Could not parse MCP server ID as integer: "+data.ID.ValueString())
		return
	}

	instance, err := r.client.GetMCPServer(ctx, orgID, envID, mcpID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading MCP server", "Could not read MCP server ID "+data.ID.ValueString()+": "+err.Error())
		return
	}

	// Snapshot user-managed fields before overwriting with MCP server response.
	// The Anypoint Platform may silently modify routing upstreams, endpoint
	// details, or deployment fields after creation. If we let those MCP server-side
	// changes propagate into state, Terraform will generate a spurious diff
	// and attempt an update that often fails (e.g. "upstream uri is mandatory").
	// User-managed fields are owned exclusively by Terraform config/state.
	gatewayID := data.GatewayID
	existingRouting := data.Routing
	existingEndpoint := data.Endpoint
	existingDeployment := data.Deployment

	r.flattenInstance(ctx, instance, &data, orgID, envID)

	// Restore user-managed fields from state.
	data.GatewayID = gatewayID
	if !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown() {
		data.Routing = types.ListNull(data.Routing.ElementType(ctx))
	} else if !existingRouting.IsNull() && !existingRouting.IsUnknown() {
		data.Routing = existingRouting
	}
	if !existingEndpoint.IsNull() && !existingEndpoint.IsUnknown() {
		data.Endpoint = existingEndpoint
	}
	data.Deployment = mergeDeploymentObjects(data.Deployment, existingDeployment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MCPServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MCPServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := state.EnvironmentID.ValueString()

	mcpID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Server ID", "Could not parse MCP server ID: "+state.ID.ValueString())
		return
	}

	if !plan.GatewayID.IsNull() && !plan.GatewayID.IsUnknown() && (plan.Deployment.IsNull() || plan.Deployment.IsUnknown()) {
		var dep *agentstools.MCPServerDeployment
		dep, err = r.resolveGatewayDeployment(ctx, orgID, envID, plan.GatewayID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error resolving gateway_id", err.Error())
			return
		}
		plan.Deployment = deploymentToObject(&DeploymentModel{
			EnvironmentID:  types.StringValue(dep.EnvironmentID),
			Type:           types.StringValue(dep.Type),
			ExpectedStatus: types.StringValue(dep.ExpectedStatus),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue(dep.TargetID),
			TargetName:     types.StringValue(dep.TargetName),
			GatewayVersion: types.StringValue(dep.GatewayVersion),
		})
	}

	updateReq := r.expandUpdateRequest(ctx, plan)

	if len(updateReq.Routing) > 0 {
		var current *agentstools.MCPServer
		current, err = r.client.GetMCPServer(ctx, orgID, envID, mcpID)
		if err == nil && len(current.Routing) > 0 {
			r.mergeUpstreamIDs(current.Routing, updateReq.Routing)
		}
	}

	var instance *agentstools.MCPServer
	instance, err = r.client.UpdateMCPServer(ctx, orgID, envID, mcpID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating MCP server", "Could not update MCP server: "+err.Error())
		return
	}

	gatewayID := plan.GatewayID
	plannedRouting := plan.Routing
	plannedEndpoint := plan.Endpoint
	plannedUpstreamURI := plan.UpstreamURI
	r.flattenInstance(ctx, instance, &plan, orgID, envID)
	plan.GatewayID = gatewayID
	if !plannedUpstreamURI.IsNull() && !plannedUpstreamURI.IsUnknown() {
		plan.UpstreamURI = plannedUpstreamURI
		plan.Routing = types.ListNull(plan.Routing.ElementType(ctx))
	} else if !plannedRouting.IsNull() {
		plan.Routing = plannedRouting
	}
	if !plannedEndpoint.IsNull() && !plannedEndpoint.IsUnknown() {
		plan.Endpoint = plannedEndpoint
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MCPServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MCPServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	mcpID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Server ID", "Could not parse MCP server ID: "+data.ID.ValueString())
		return
	}

	if err := r.client.DeleteMCPServer(ctx, orgID, envID, mcpID); err != nil {
		resp.Diagnostics.AddError("Error deleting MCP server", "Could not delete MCP server: "+err.Error())
	}
}

func (r *MCPServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Helpers ---

// resolveGatewayDeployment fetches gateway details and builds a deployment payload.
func (r *MCPServerResource) resolveGatewayDeployment(ctx context.Context, orgID, envID, gatewayID string) (*agentstools.MCPServerDeployment, error) {
	gw, err := r.client.GetGatewayInfo(ctx, orgID, envID, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve gateway_id %s: %w", gatewayID, err)
	}
	return &agentstools.MCPServerDeployment{
		EnvironmentID:  envID,
		Type:           "HY",
		ExpectedStatus: "deployed",
		TargetID:       gw.ID,
		TargetName:     gw.Name,
		GatewayVersion: gw.RuntimeVersion,
	}, nil
}

func (r *MCPServerResource) expandCreateRequest(ctx context.Context, data MCPServerResourceModel) *agentstools.CreateMCPServerRequest {
	req := &agentstools.CreateMCPServerRequest{
		Technology: data.Technology.ValueString(),
	}

	if !data.ProviderID.IsNull() && !data.ProviderID.IsUnknown() {
		pid := data.ProviderID.ValueString()
		req.ProviderID = &pid
	}
	if !data.InstanceLabel.IsNull() && !data.InstanceLabel.IsUnknown() {
		req.InstanceLabel = data.InstanceLabel.ValueString()
	}
	if !data.ApprovalMethod.IsNull() && !data.ApprovalMethod.IsUnknown() {
		am := data.ApprovalMethod.ValueString()
		req.ApprovalMethod = &am
	}

	if data.Spec != nil {
		req.Spec = &agentstools.MCPServerSpec{
			AssetID: data.Spec.AssetID.ValueString(),
			GroupID: data.Spec.GroupID.ValueString(),
			Version: data.Spec.Version.ValueString(),
		}
	}

	if ep := endpointFromObject(data.Endpoint); ep != nil {
		req.Endpoint = &agentstools.MCPServerEndpoint{
			DeploymentType: ep.DeploymentType.ValueString(),
			Type:           "mcp", // MCP Server type
		}

		technology := data.Technology.ValueString()
		switch technology {
		case "flexGateway", "":
			if !ep.BasePath.IsNull() && !ep.BasePath.IsUnknown() {
				basePath := strings.TrimPrefix(ep.BasePath.ValueString(), "/")
				proxyURI := "http://0.0.0.0:8081/" + basePath
				req.Endpoint.ProxyURI = &proxyURI
			} else {
				proxyURI := "http://0.0.0.0:8081/"
				req.Endpoint.ProxyURI = &proxyURI
			}

			req.Endpoint.TLSContexts = &agentstools.MCPServerTLSContexts{}
		case "mule4":
			mule4 := true
			req.Endpoint.MuleVersion4OrAbove = &mule4
			req.Endpoint.ProxyURI = nil

			if !ep.URI.IsNull() && !ep.URI.IsUnknown() {
				uri := ep.URI.ValueString()
				req.Endpoint.ProxyURI = &uri
			}

			req.Endpoint.IsCloudHub = nil
			req.Endpoint.ReferencesUserDomain = nil
		}

		if !ep.ResponseTimeout.IsNull() && !ep.ResponseTimeout.IsUnknown() {
			rt := int(ep.ResponseTimeout.ValueInt64())
			req.Endpoint.ResponseTimeout = &rt
		}
	}

	if !data.ConsumerEndpoint.IsNull() && !data.ConsumerEndpoint.IsUnknown() {
		ce := data.ConsumerEndpoint.ValueString()
		req.EndpointURI = &ce
	}

	if dep := deploymentFromObject(data.Deployment); dep != nil {
		req.Deployment = &agentstools.MCPServerDeployment{
			EnvironmentID:  dep.EnvironmentID.ValueString(),
			Type:           dep.Type.ValueString(),
			ExpectedStatus: dep.ExpectedStatus.ValueString(),
			Overwrite:      dep.Overwrite.ValueBool(),
			TargetID:       dep.TargetID.ValueString(),
			TargetName:     dep.TargetName.ValueString(),
			GatewayVersion: dep.GatewayVersion.ValueString(),
		}
	}

	if !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown() {
		req.Routing = []agentstools.MCPServerRoute{
			{
				Upstreams: []agentstools.MCPServerUpstream{
					{Weight: 100, URI: data.UpstreamURI.ValueString()},
				},
			},
		}
	} else {
		req.Routing = r.expandRouting(ctx, data.Routing)
	}

	return req
}

func (r *MCPServerResource) expandUpdateRequest(ctx context.Context, data MCPServerResourceModel) *agentstools.UpdateMCPServerRequest {
	req := &agentstools.UpdateMCPServerRequest{}

	tech := data.Technology.ValueString()
	req.Technology = &tech

	if !data.InstanceLabel.IsNull() && !data.InstanceLabel.IsUnknown() {
		il := data.InstanceLabel.ValueString()
		req.InstanceLabel = &il
	}

	if ep := endpointFromObject(data.Endpoint); ep != nil {
		req.Endpoint = &agentstools.MCPServerEndpoint{
			DeploymentType: ep.DeploymentType.ValueString(),
			Type:           "mcp", // MCP Server type
		}

		technology := data.Technology.ValueString()
		switch technology {
		case "flexGateway", "":
			if !ep.BasePath.IsNull() && !ep.BasePath.IsUnknown() {
				basePath := strings.TrimPrefix(ep.BasePath.ValueString(), "/")
				proxyURI := "http://0.0.0.0:8081/" + basePath
				req.Endpoint.ProxyURI = &proxyURI
			} else {
				proxyURI := "http://0.0.0.0:8081/"
				req.Endpoint.ProxyURI = &proxyURI
			}

			req.Endpoint.TLSContexts = &agentstools.MCPServerTLSContexts{}
		case "mule4":
			mule4 := true
			req.Endpoint.MuleVersion4OrAbove = &mule4
			req.Endpoint.ProxyURI = nil

			if !ep.URI.IsNull() && !ep.URI.IsUnknown() {
				uri := ep.URI.ValueString()
				req.Endpoint.ProxyURI = &uri
			}

			req.Endpoint.IsCloudHub = nil
			req.Endpoint.ReferencesUserDomain = nil
		}

		if !ep.ResponseTimeout.IsNull() && !ep.ResponseTimeout.IsUnknown() {
			rt := int(ep.ResponseTimeout.ValueInt64())
			req.Endpoint.ResponseTimeout = &rt
		}
	}

	if !data.ConsumerEndpoint.IsNull() && !data.ConsumerEndpoint.IsUnknown() {
		ce := data.ConsumerEndpoint.ValueString()
		req.EndpointURI = &ce
	}

	if data.Spec != nil {
		req.Spec = &agentstools.MCPServerSpec{
			AssetID: data.Spec.AssetID.ValueString(),
			GroupID: data.Spec.GroupID.ValueString(),
			Version: data.Spec.Version.ValueString(),
		}
	}

	if dep := deploymentFromObject(data.Deployment); dep != nil {
		req.Deployment = &agentstools.MCPServerDeployment{
			EnvironmentID:  dep.EnvironmentID.ValueString(),
			Type:           dep.Type.ValueString(),
			ExpectedStatus: dep.ExpectedStatus.ValueString(),
			Overwrite:      dep.Overwrite.ValueBool(),
			TargetID:       dep.TargetID.ValueString(),
			TargetName:     dep.TargetName.ValueString(),
			GatewayVersion: dep.GatewayVersion.ValueString(),
		}
	}

	if !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown() {
		req.Routing = []agentstools.MCPServerRoute{
			{
				Upstreams: []agentstools.MCPServerUpstream{
					{Weight: 100, URI: data.UpstreamURI.ValueString()},
				},
			},
		}
	} else {
		req.Routing = r.expandRouting(ctx, data.Routing)
	}

	return req
}

// mergeUpstreamIDs copies server-assigned upstream IDs from the current
// instance into the update payload. Matches by route index and upstream URI.
func (r *MCPServerResource) mergeUpstreamIDs(current, update []agentstools.MCPServerRoute) {
	for i := range update {
		if i >= len(current) {
			break
		}
		// Try URI-based matching first.
		currentByURI := make(map[string]string)
		for _, us := range current[i].Upstreams {
			if us.ID != "" && us.URI != "" {
				currentByURI[us.URI] = us.ID
			}
		}
		for j := range update[i].Upstreams {
			if id, ok := currentByURI[update[i].Upstreams[j].URI]; ok {
				update[i].Upstreams[j].ID = id
				update[i].Upstreams[j].URI = ""
			}
		}
		// Fallback: match by position for any upstreams that still lack an ID.
		for j := range update[i].Upstreams {
			if update[i].Upstreams[j].ID == "" && j < len(current[i].Upstreams) && current[i].Upstreams[j].ID != "" {
				update[i].Upstreams[j].ID = current[i].Upstreams[j].ID
				update[i].Upstreams[j].URI = ""
			}
		}
	}
}

func (r *MCPServerResource) expandRouting(ctx context.Context, routingList types.List) []agentstools.MCPServerRoute {
	if routingList.IsNull() || routingList.IsUnknown() {
		return nil
	}

	var routes []RouteModel
	routingList.ElementsAs(ctx, &routes, false)

	mcpRoutes := make([]agentstools.MCPServerRoute, 0, len(routes))
	for _, route := range routes {
		mcpRoute := agentstools.MCPServerRoute{
			Label: route.Label.ValueString(),
		}

		if route.Rules != nil {
			rules := &agentstools.MCPServerRules{}
			if !route.Rules.Methods.IsNull() && !route.Rules.Methods.IsUnknown() {
				rules.Methods = route.Rules.Methods.ValueString()
			}
			if !route.Rules.Path.IsNull() && !route.Rules.Path.IsUnknown() {
				rules.Path = route.Rules.Path.ValueString()
			}
			if !route.Rules.Host.IsNull() && !route.Rules.Host.IsUnknown() {
				rules.Host = route.Rules.Host.ValueString()
			}
			if !route.Rules.Headers.IsNull() && !route.Rules.Headers.IsUnknown() {
				hdrs := make(map[string]string)
				route.Rules.Headers.ElementsAs(ctx, &hdrs, false)
				rules.Headers = hdrs
			}
			mcpRoute.Rules = rules
		}

		var upstreams []UpstreamModel
		route.Upstreams.ElementsAs(ctx, &upstreams, false)

		for _, us := range upstreams {
			upstream := agentstools.MCPServerUpstream{
				Weight: int(us.Weight.ValueInt64()),
				URI:    us.URI.ValueString(),
				Label:  us.Label.ValueString(),
			}
			if !us.TLSContextID.IsNull() && !us.TLSContextID.IsUnknown() {
				parts := strings.Split(us.TLSContextID.ValueString(), "/")
				if len(parts) == 2 {
					upstream.TLSContext = &agentstools.MCPServerUpstreamTLS{
						SecretGroupID: parts[0],
						TLSContextID:  parts[1],
					}
				}
			}
			mcpRoute.Upstreams = append(mcpRoute.Upstreams, upstream)
		}

		mcpRoutes = append(mcpRoutes, mcpRoute)
	}

	return mcpRoutes
}

func (r *MCPServerResource) flattenInstance(_ context.Context, inst *agentstools.MCPServer, data *MCPServerResourceModel, orgID, envID string) {
	data.ID = types.StringValue(strconv.Itoa(inst.ID))
	if inst.Status != "" {
		data.Status = types.StringValue(inst.Status)
	}
	if inst.AssetID != "" {
		data.AssetID = types.StringValue(inst.AssetID)
	}
	if inst.AssetVersion != "" {
		data.AssetVersion = types.StringValue(inst.AssetVersion)
	}
	if inst.ProductVersion != "" {
		data.ProductVersion = types.StringValue(inst.ProductVersion)
	}
	if inst.Technology != "" {
		data.Technology = types.StringValue(inst.Technology)
	}

	if data.OrganizationID.IsNull() || data.OrganizationID.IsUnknown() || data.OrganizationID.ValueString() == "" {
		data.OrganizationID = types.StringValue(orgID)
	}
	data.EnvironmentID = types.StringValue(envID)

	if inst.ProviderID != nil {
		data.ProviderID = types.StringValue(*inst.ProviderID)
	}
	if inst.InstanceLabel != "" {
		data.InstanceLabel = types.StringValue(inst.InstanceLabel)
	}
	if inst.ApprovalMethod != "" {
		data.ApprovalMethod = types.StringValue(inst.ApprovalMethod)
	}

	if inst.Spec != nil {
		data.Spec = &SpecModel{
			AssetID: types.StringValue(inst.Spec.AssetID),
			GroupID: types.StringValue(inst.Spec.GroupID),
			Version: types.StringValue(inst.Spec.Version),
		}
	}

	if inst.Endpoint != nil {
		ep := &EndpointModel{
			DeploymentType: types.StringValue(inst.Endpoint.DeploymentType),
			Type:           types.StringValue(inst.Endpoint.Type),
		}

		technology := inst.Technology
		switch technology {
		case "flexGateway", "":
			if inst.Endpoint.ProxyURI != nil && *inst.Endpoint.ProxyURI != "" {
				ep.BasePath = types.StringValue(strings.TrimPrefix(*inst.Endpoint.ProxyURI, "http://0.0.0.0:8081/"))
			} else {
				ep.BasePath = types.StringNull()
			}
			ep.URI = types.StringNull()
		case "mule4":
			if inst.Endpoint.ProxyURI != nil && *inst.Endpoint.ProxyURI != "" {
				ep.URI = types.StringValue(*inst.Endpoint.ProxyURI)
			} else {
				ep.URI = types.StringNull()
			}
			ep.BasePath = types.StringNull()
		default:
			ep.BasePath = types.StringNull()
			ep.URI = types.StringNull()
		}

		if inst.Endpoint.ResponseTimeout != nil {
			ep.ResponseTimeout = types.Int64Value(int64(*inst.Endpoint.ResponseTimeout))
		} else {
			ep.ResponseTimeout = types.Int64Null()
		}
		data.Endpoint = endpointToObject(ep)
	} else {
		data.Endpoint = types.ObjectNull(endpointAttrTypes)
	}

	if inst.EndpointURI != "" {
		data.ConsumerEndpoint = types.StringValue(inst.EndpointURI)
	} else {
		data.ConsumerEndpoint = types.StringNull()
	}

	if inst.Deployment != nil {
		data.Deployment = deploymentToObject(&DeploymentModel{
			EnvironmentID:  types.StringValue(inst.Deployment.EnvironmentID),
			Type:           types.StringValue(inst.Deployment.Type),
			ExpectedStatus: types.StringValue(inst.Deployment.ExpectedStatus),
			Overwrite:      types.BoolValue(inst.Deployment.Overwrite),
			TargetID:       types.StringValue(inst.Deployment.TargetID),
			TargetName:     types.StringValue(inst.Deployment.TargetName),
			GatewayVersion: types.StringValue(inst.Deployment.GatewayVersion),
		})
	} else {
		data.Deployment = types.ObjectNull(deploymentAttrTypes)
	}

	// Flatten routing into types.List
	rulesObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"methods": types.StringType,
			"path":    types.StringType,
			"host":    types.StringType,
			"headers": types.MapType{ElemType: types.StringType},
		},
	}
	upstreamObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"weight":         types.Int64Type,
			"uri":            types.StringType,
			"label":          types.StringType,
			"tls_context_id": types.StringType,
		},
	}
	routeObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"label":     types.StringType,
			"rules":     rulesObjType,
			"upstreams": types.ListType{ElemType: upstreamObjType},
		},
	}

	// Extract the first upstream ID for the computed upstream_id attribute.
	data.UpstreamID = types.StringNull()
	if len(inst.Routing) > 0 && len(inst.Routing[0].Upstreams) > 0 && inst.Routing[0].Upstreams[0].ID != "" {
		data.UpstreamID = types.StringValue(inst.Routing[0].Upstreams[0].ID)
	}

	if len(inst.Routing) > 0 {
		routeElems := make([]attr.Value, 0, len(inst.Routing))
		for _, route := range inst.Routing {
			usElems := make([]attr.Value, 0, len(route.Upstreams))
			for _, us := range route.Upstreams {
				var tlsCtxVal attr.Value
				if us.TLSContext != nil && us.TLSContext.SecretGroupID != "" && us.TLSContext.TLSContextID != "" {
					tlsCtxVal = types.StringValue(fmt.Sprintf("%s/%s", us.TLSContext.SecretGroupID, us.TLSContext.TLSContextID))
				} else {
					tlsCtxVal = types.StringNull()
				}
				usObj, usDiags := types.ObjectValue(
					upstreamObjType.AttrTypes,
					map[string]attr.Value{
						"weight":         types.Int64Value(int64(us.Weight)),
						"uri":            types.StringValue(us.URI),
						"label":          types.StringValue(us.Label),
						"tls_context_id": tlsCtxVal,
					},
				)
				if usDiags.HasError() {
					continue
				}
				usElems = append(usElems, usObj)
			}

			var rulesVal attr.Value
			if route.Rules != nil {
				methods := types.StringNull()
				if route.Rules.Methods != "" {
					methods = types.StringValue(route.Rules.Methods)
				}
				pathVal := types.StringNull()
				if route.Rules.Path != "" {
					pathVal = types.StringValue(route.Rules.Path)
				}
				hostVal := types.StringNull()
				if route.Rules.Host != "" {
					hostVal = types.StringValue(route.Rules.Host)
				}
				var headersVal attr.Value
				if len(route.Rules.Headers) > 0 {
					hdrElems := make(map[string]attr.Value, len(route.Rules.Headers))
					for k, v := range route.Rules.Headers {
						hdrElems[k] = types.StringValue(v)
					}
					headersVal = types.MapValueMust(types.StringType, hdrElems)
				} else {
					headersVal = types.MapNull(types.StringType)
				}
				rulesObj, rulesDiags := types.ObjectValue(
					rulesObjType.AttrTypes,
					map[string]attr.Value{
						"methods": methods,
						"path":    pathVal,
						"host":    hostVal,
						"headers": headersVal,
					},
				)
				if rulesDiags.HasError() {
					rulesVal = types.ObjectNull(rulesObjType.AttrTypes)
				} else {
					rulesVal = rulesObj
				}
			} else {
				rulesVal = types.ObjectNull(rulesObjType.AttrTypes)
			}

			routeObj, routeDiags := types.ObjectValue(
				routeObjType.AttrTypes,
				map[string]attr.Value{
					"label":     types.StringValue(route.Label),
					"rules":     rulesVal,
					"upstreams": types.ListValueMust(upstreamObjType, usElems),
				},
			)
			if !routeDiags.HasError() {
				routeElems = append(routeElems, routeObj)
			}
		}

		data.Routing = types.ListValueMust(routeObjType, routeElems)
	} else {
		data.Routing = types.ListNull(routeObjType)
	}
}
