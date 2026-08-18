package agentstools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                   = &MCPBridgeResource{}
	_ resource.ResourceWithConfigure      = &MCPBridgeResource{}
	_ resource.ResourceWithImportState    = &MCPBridgeResource{}
	_ resource.ResourceWithValidateConfig = &MCPBridgeResource{}
	_ resource.ResourceWithModifyPlan     = &MCPBridgeResource{}
)

// MCPBridgeResource manages an MCP bridge: a one-click server generated from N source
// REST APIs. Unlike anypoint_mcp_server (user supplies an existing Exchange asset),
// the bridge GENERATES its Exchange asset from explicit tool declarations and attaches
// the MCP transcoding policies that turn tool calls into REST calls. See the live
// contract in .agents/artifacts/mcp-bridge-onefile-capture.md.
type MCPBridgeResource struct {
	client *agentstools.MCPBridgeClient
}

// --- Terraform State Models ---

type MCPBridgeResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	GatewayID      types.String `tfsdk:"gateway_id"`
	MCPAssetName   types.String `tfsdk:"mcp_asset_name"`
	Port           types.Int64  `tfsdk:"port"`
	BasePath       types.String `tfsdk:"base_path"`

	AssetID          types.String `tfsdk:"asset_id"`
	AssetVersion     types.String `tfsdk:"asset_version"`
	ProductVersion   types.String `tfsdk:"product_version"`
	ConsumerEndpoint types.String `tfsdk:"consumer_endpoint"`
	Status           types.String `tfsdk:"status"`
	Technology       types.String `tfsdk:"technology"`

	Deployment types.Object `tfsdk:"deployment"`
	SourceAPIs types.List   `tfsdk:"source_apis"`
}

type BridgeSourceAPIModel struct {
	Label       types.String `tfsdk:"label"`
	UpstreamURI types.String `tfsdk:"upstream_uri"`
	AssetID     types.String `tfsdk:"asset_id"`
	GroupID     types.String `tfsdk:"group_id"`
	Version     types.String `tfsdk:"version"`
	Tools       types.List   `tfsdk:"tools"`
}

type BridgeToolModel struct {
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Method       types.String `tfsdk:"method"`
	Path         types.String `tfsdk:"path"`
	QueryParams  types.List   `tfsdk:"query_params"`
	HeaderParams types.List   `tfsdk:"header_params"`
	HasBody      types.Bool   `tfsdk:"has_body"`
}

var bridgeToolAttrTypes = map[string]attr.Type{
	"name":          types.StringType,
	"description":   types.StringType,
	"method":        types.StringType,
	"path":          types.StringType,
	"query_params":  types.ListType{ElemType: types.StringType},
	"header_params": types.ListType{ElemType: types.StringType},
	"has_body":      types.BoolType,
}

var bridgeSourceAttrTypes = map[string]attr.Type{
	"label":        types.StringType,
	"upstream_uri": types.StringType,
	"asset_id":     types.StringType,
	"group_id":     types.StringType,
	"version":      types.StringType,
	"tools":        types.ListType{ElemType: types.ObjectType{AttrTypes: bridgeToolAttrTypes}},
}

func NewMCPBridgeResource() resource.Resource {
	return &MCPBridgeResource{}
}

func (r *MCPBridgeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_bridge"
}

func (r *MCPBridgeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an MCP bridge in Anypoint API Manager. An MCP bridge generates an MCP server " +
			"from one or more source REST APIs: it publishes a generated Exchange asset (mcp-metadata.json), " +
			"creates the Flex/Omni Gateway instance with one route per source API, and attaches the MCP " +
			"transcoding policies that map MCP tool calls to REST calls. Tools are declared explicitly.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric identifier of the MCP bridge instance (stored as string).",
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
				Description: "REQUIRED. The environment ID where the MCP bridge is created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gateway_id": schema.StringAttribute{
				Description: "REQUIRED. The Omni/Self-Managed (Flex) Gateway UUID to deploy the bridge to. " +
					"The deployment block is auto-populated by fetching gateway details.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mcp_asset_name": schema.StringAttribute{
				Description: "REQUIRED. The MCP asset name (UI: \"MCP asset name\"). Becomes the generated Exchange asset name and (sanitized) asset ID, and is shown as the instance name on the MCP server summary.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"port": schema.Int64Attribute{
				Description: "The listener port for the bridge on the gateway. Defaults to 8081. The proxy URI is http://0.0.0.0:<port>/<base_path>.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(8081),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"base_path": schema.StringAttribute{
				Description: "The base path for the bridge proxy URI (default empty). The proxy URI is http://0.0.0.0:<port>/<base_path>.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"asset_id": schema.StringAttribute{
				Description: "The generated Exchange asset ID (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_version": schema.StringAttribute{
				Description: "The generated Exchange asset version (computed; starts at 1.0.0 and bumps on tool updates).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_version": schema.StringAttribute{
				Description: "The product version (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"consumer_endpoint": schema.StringAttribute{
				Description: "The consumer-facing MCP endpoint URI (UI: \"Consumer Endpoint\"; computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the MCP bridge (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"technology": schema.StringAttribute{
				Description: "The gateway technology (computed; always 'flexGateway' for a bridge).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment": schema.SingleNestedAttribute{
				Description: "Deployment target details, auto-populated from gateway_id (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"environment_id":  schema.StringAttribute{Description: "Deployment environment ID.", Computed: true},
					"type":            schema.StringAttribute{Description: "Deployment type (HY).", Computed: true},
					"expected_status": schema.StringAttribute{Description: "Expected deployment status.", Computed: true},
					"overwrite":       schema.BoolAttribute{Description: "Whether an existing deployment is overwritten.", Computed: true},
					"target_id":       schema.StringAttribute{Description: "The target gateway ID.", Computed: true},
					"target_name":     schema.StringAttribute{Description: "The target gateway name.", Computed: true},
					"gateway_version": schema.StringAttribute{Description: "The gateway runtime version.", Computed: true},
				},
			},
			"source_apis": schema.ListNestedAttribute{
				Description: "REQUIRED. One block per source REST API. Each becomes a route (matched by the X-UPSTREAM-NAME header) " +
					"and an upstream backend, and its tools are transcoded to REST calls.",
				Required: true,
				PlanModifiers: []planmodifier.List{
					requiresReplaceOnStructuralChange(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							Description: "REQUIRED. The source API label; used as the route label and the X-UPSTREAM-NAME header value. Must be unique within the bridge.",
							Required:    true,
						},
						"upstream_uri": schema.StringAttribute{
							Description: "REQUIRED. The real backend base URI that tool calls are forwarded to.",
							Required:    true,
						},
						"asset_id": schema.StringAttribute{
							Description: "REQUIRED. The source REST API's Exchange asset ID (the connection link).",
							Required:    true,
						},
						"group_id": schema.StringAttribute{
							Description: "The source asset group (organization) ID. Defaults to organization_id.",
							Optional:    true,
						},
						"version": schema.StringAttribute{
							Description: "REQUIRED. The source asset version.",
							Required:    true,
						},
						"tools": schema.ListNestedAttribute{
							Description: "REQUIRED. The explicit tools exposed for this source API. Each maps one REST operation to an MCP tool.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"method": schema.StringAttribute{
										Description: "REQUIRED. The HTTP method (GET, POST, PUT, PATCH, DELETE).",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOf("GET", "POST", "PUT", "PATCH", "DELETE", "get", "post", "put", "patch", "delete"),
										},
									},
									"path": schema.StringAttribute{
										Description: "REQUIRED. The REST operation path, e.g. /pets/{petId}. Path parameters ({...}) become required tool inputs automatically.",
										Required:    true,
									},
									"name": schema.StringAttribute{
										Description: "The tool name. Defaults to <method>_<slug(path)> (e.g. get_pets_petid).",
										Optional:    true,
									},
									"description": schema.StringAttribute{
										Description: "The tool description shown to MCP clients. Defaults to the tool name.",
										Optional:    true,
									},
									"query_params": schema.ListAttribute{
										Description: "Query parameter names exposed as tool inputs.",
										Optional:    true,
										ElementType: types.StringType,
									},
									"header_params": schema.ListAttribute{
										Description: "Header parameter names exposed as tool inputs.",
										Optional:    true,
										ElementType: types.StringType,
									},
									"has_body": schema.BoolAttribute{
										Description: "Whether the operation takes a request body (typically for POST/PUT/PATCH). Adds a required 'body' tool input.",
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
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

func (r *MCPBridgeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	bridgeClient, err := agentstools.NewMCPBridgeClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create MCP Bridge Client",
			"An unexpected error occurred when creating the MCP Bridge client.\n\nAnypoint Client Error: "+err.Error(),
		)
		return
	}
	r.client = bridgeClient
}

func (r *MCPBridgeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data MCPBridgeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.SourceAPIs.IsNull() || data.SourceAPIs.IsUnknown() {
		return
	}
	var sources []BridgeSourceAPIModel
	resp.Diagnostics.Append(data.SourceAPIs.ElementsAs(ctx, &sources, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(sources) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("source_apis"), "At least one source API required",
			"An MCP bridge must declare at least one source_api block.")
		return
	}
	seen := map[string]bool{}
	for i, s := range sources {
		if !s.Label.IsNull() && !s.Label.IsUnknown() {
			lbl := s.Label.ValueString()
			if seen[lbl] {
				resp.Diagnostics.AddAttributeError(path.Root("source_apis").AtListIndex(i).AtName("label"),
					"Duplicate source API label", fmt.Sprintf("The label %q is used by more than one source_api; labels must be unique.", lbl))
			}
			seen[lbl] = true
		}
		if s.Tools.IsNull() || s.Tools.IsUnknown() {
			continue
		}
		var tools []BridgeToolModel
		resp.Diagnostics.Append(s.Tools.ElementsAs(ctx, &tools, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(tools) == 0 {
			resp.Diagnostics.AddAttributeError(path.Root("source_apis").AtListIndex(i).AtName("tools"),
				"At least one tool required", "Each source_api must declare at least one tool.")
		}
		for j, t := range tools {
			if !t.Path.IsNull() && !t.Path.IsUnknown() && !strings.HasPrefix(t.Path.ValueString(), "/") {
				resp.Diagnostics.AddAttributeError(path.Root("source_apis").AtListIndex(i).AtName("tools").AtListIndex(j).AtName("path"),
					"Invalid tool path", "The tool path must start with '/'.")
			}
		}
	}
}

// ModifyPlan marks the computed platform-managed fields as unknown when the tool set
// changes on an update. Update republishes the generated asset at a bumped patch version
// and re-syncs the transcoding policies, then reads the instance back. The prior state
// values (which UseStateForUnknown would otherwise carry into the plan) cannot be
// predicted for asset_version (it is bumped) and can transiently differ for the other
// platform-managed fields while the gateway re-registers the new version — leaving any of
// them in the plan yields a "provider produced inconsistent result after apply" error.
// Structural source-API changes require replacement (handled in Update) and are not
// affected here.
func (r *MCPBridgeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only relevant on update: create (state null) and destroy (plan null) are skipped.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state MCPBridgeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	planSources := r.toBridgeSources(ctx, plan.SourceAPIs, "")
	stateSources := r.toBridgeSources(ctx, state.SourceAPIs, "")
	if bridgeToolSignature(planSources) != bridgeToolSignature(stateSources) {
		resp.Plan.SetAttribute(ctx, path.Root("asset_version"), types.StringUnknown())
		// These are read back from the live instance after the version move / policy
		// resync and can flap while the gateway re-registers; mark them unknown so the
		// readback value is always accepted.
		resp.Plan.SetAttribute(ctx, path.Root("product_version"), types.StringUnknown())
		resp.Plan.SetAttribute(ctx, path.Root("consumer_endpoint"), types.StringUnknown())
		resp.Plan.SetAttribute(ctx, path.Root("status"), types.StringUnknown())
	}
}

// --- CRUD ---

func (r *MCPBridgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MCPBridgeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	sources := r.toBridgeSources(ctx, data.SourceAPIs, orgID)

	deployment, err := r.resolveDeployment(ctx, orgID, envID, data.GatewayID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error resolving gateway_id", err.Error())
		return
	}

	assetID := bridgeAssetID(data.MCPAssetName.ValueString())
	version := "1.0.0"
	proxyURI := bridgeProxyURI(data.Port.ValueInt64(), data.BasePath.ValueString())

	metaJSON, err := json.Marshal(buildBridgeMetadata(proxyURI, sources))
	if err != nil {
		resp.Diagnostics.AddError("Error building MCP metadata", err.Error())
		return
	}

	publishInput := &agentstools.PublishBridgeAssetInput{
		OrganizationID: orgID,
		GroupID:        orgID,
		AssetID:        assetID,
		Version:        version,
		Name:           data.MCPAssetName.ValueString(),
		MetadataJSON:   metaJSON,
	}
	if err = r.publishBridgeAsset(ctx, orgID, publishInput); err != nil {
		resp.Diagnostics.AddError("Error publishing generated MCP asset", err.Error())
		return
	}

	createReq := &agentstools.CreateBridgeInstanceRequest{
		Spec:       &agentstools.MCPBridgeSpec{AssetID: assetID, GroupID: orgID, Version: version},
		Endpoint:   &agentstools.MCPBridgeEndpoint{Type: "mcp", ProxyURI: &proxyURI, DeploymentType: "HY"},
		Technology: "flexGateway",
		Routing:    buildBridgeRouting(sources),
		Deployment: deployment,
		Metadata:   map[string]string{"generatedBy": "mcp_bridge"},
	}

	bridge, err := r.client.CreateBridgeInstance(ctx, orgID, envID, createReq)
	if err != nil {
		// The asset was published but the instance was not created. Roll the asset
		// back so a retry doesn't 409 on a leftover orphan (Class J: never orphan).
		r.cleanupPartialBridge(ctx, orgID, envID, 0, assetID, version)
		resp.Diagnostics.AddError("Error creating MCP bridge instance", err.Error())
		return
	}

	ups, err := r.client.GetBridgeUpstreams(ctx, orgID, envID, bridge.ID)
	if err != nil {
		r.cleanupPartialBridge(ctx, orgID, envID, bridge.ID, assetID, version)
		resp.Diagnostics.AddError("Error reading MCP bridge upstreams after create", err.Error())
		return
	}

	if err = r.attachBridgePolicies(ctx, orgID, envID, bridge.ID, sources, ups); err != nil {
		r.cleanupPartialBridge(ctx, orgID, envID, bridge.ID, assetID, version)
		resp.Diagnostics.AddError("Error attaching MCP bridge policies", err.Error())
		return
	}

	instance, err := r.client.GetBridge(ctx, orgID, envID, bridge.ID)
	if err != nil {
		// The bridge was fully created (instance + upstreams + policies); a failure on
		// this final readback would otherwise strand it with no TF state. Roll it back
		// like every other post-publish failure branch (Class J: never orphan).
		r.cleanupPartialBridge(ctx, orgID, envID, bridge.ID, assetID, version)
		resp.Diagnostics.AddError("Error reading MCP bridge after create", err.Error())
		return
	}

	plannedSources := data.SourceAPIs
	r.flattenBridge(instance, &data, orgID, envID)
	data.SourceAPIs = plannedSources
	tflog.Trace(ctx, "created MCP bridge", map[string]interface{}{"id": instance.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// publishBridgeAsset publishes the generated MCP asset, self-healing a leftover
// orphan from a prior failed create: Exchange rejects a re-publish of an existing
// GAV with a 409 (ASSET_PRE_CONDITIONS_FAILED). Because a genuinely in-use bridge
// asset would be tracked in Terraform state (and thus never re-created here), a 409 on
// CREATE means the previous version is orphaned — hard-delete it and republish once.
func (r *MCPBridgeResource) publishBridgeAsset(ctx context.Context, orgID string, in *agentstools.PublishBridgeAssetInput) error {
	_, err := r.client.PublishBridgeAsset(ctx, in)
	if err == nil {
		return nil
	}
	if !isAssetConflict(err) {
		return err
	}
	tflog.Warn(ctx, "generated MCP asset already exists (orphan from a prior failed create); hard-deleting and republishing", map[string]interface{}{
		"asset_id": in.AssetID, "version": in.Version,
	})
	// Best-effort delete; ignore its error and surface the republish result.
	_ = r.client.DeleteBridgeAssetVersion(ctx, orgID, in.AssetID, in.Version)
	_, err = r.client.PublishBridgeAsset(ctx, in)
	return err
}

// cleanupPartialBridge best-effort rolls back a half-created bridge so a failed
// Create never leaves an orphaned instance or Exchange asset behind. instanceID == 0
// means no instance was created yet.
func (r *MCPBridgeResource) cleanupPartialBridge(ctx context.Context, orgID, envID string, instanceID int, assetID, version string) {
	if instanceID != 0 {
		if err := r.client.DeleteBridgeInstance(ctx, orgID, envID, instanceID); err != nil {
			tflog.Warn(ctx, "failed to roll back partially-created bridge instance", map[string]interface{}{"id": instanceID, "error": err.Error()})
		}
	}
	if err := r.client.DeleteBridgeAssetVersion(ctx, orgID, assetID, version); err != nil {
		tflog.Warn(ctx, "failed to roll back generated MCP asset", map[string]interface{}{"asset_id": assetID, "version": version, "error": err.Error()})
	}
}

// isAssetConflict reports whether err is an Exchange asset-version conflict (409).
// CreateAsset surfaces this as a plain formatted error (not a typed ConflictError),
// so fall back to matching the status and platform error code.
func isAssetConflict(err error) bool {
	if err == nil {
		return false
	}
	if client.IsConflict(err) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "status 409") || strings.Contains(msg, "ASSET_PRE_CONDITIONS_FAILED")
}

func (r *MCPBridgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MCPBridgeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	bridgeID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Bridge ID", "Could not parse MCP bridge ID as integer: "+data.ID.ValueString())
		return
	}

	instance, err := r.client.GetBridge(ctx, orgID, envID, bridgeID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading MCP bridge", "Could not read MCP bridge ID "+data.ID.ValueString()+": "+err.Error())
		return
	}

	// source_apis is user-owned config: snapshot and restore verbatim so platform-side
	// rewrites (routing upstream ids, etc.) never generate a spurious diff. On import
	// (state has no source_apis yet) reconstruct them from the live upstreams + policies.
	existingSources := data.SourceAPIs
	r.flattenBridge(instance, &data, orgID, envID)

	if existingSources.IsNull() || existingSources.IsUnknown() || len(existingSources.Elements()) == 0 {
		reconstructed, rerr := r.reconstructSourceAPIs(ctx, orgID, envID, instance)
		if rerr != nil {
			resp.Diagnostics.AddError("Error reconstructing MCP bridge source APIs on import", rerr.Error())
			return
		}
		data.SourceAPIs = reconstructed
		// Import path: backfill the RequiresReplace identity fields from the live
		// instance so the first plan does not spuriously want to recreate the bridge.
		backfillBridgeImportFields(&data, instance)
	} else {
		data.SourceAPIs = existingSources
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MCPBridgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MCPBridgeResourceModel
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

	bridgeID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Bridge ID", "Could not parse MCP bridge ID: "+state.ID.ValueString())
		return
	}

	planSources := r.toBridgeSources(ctx, plan.SourceAPIs, orgID)
	stateSources := r.toBridgeSources(ctx, state.SourceAPIs, orgID)

	// v1 supports in-place TOOL edits. Structural changes to a source API (its label,
	// upstream_uri, asset_id, group_id, version, or the set of source APIs) change the
	// routing/upstreams, which the durable update path cannot patch — require a replace.
	if bridgeStructuralSignature(planSources) != bridgeStructuralSignature(stateSources) {
		resp.Diagnostics.AddError(
			"Source API structure change requires replacement",
			"Changing a source API's label, upstream_uri, asset_id, group_id, version, or adding/removing "+
				"source APIs is not an in-place update for an MCP bridge. Recreate the resource (e.g. terraform "+
				"apply -replace) to apply structural changes. Only tool edits are updated in place.",
		)
		return
	}

	assetID := state.AssetID.ValueString()
	newVersion := agentstools.BumpPatchVersion(state.AssetVersion.ValueString())
	proxyURI := bridgeProxyURI(plan.Port.ValueInt64(), plan.BasePath.ValueString())

	metaJSON, err := json.Marshal(buildBridgeMetadata(proxyURI, planSources))
	if err != nil {
		resp.Diagnostics.AddError("Error building MCP metadata", err.Error())
		return
	}

	if _, err = r.client.PublishBridgeAsset(ctx, &agentstools.PublishBridgeAssetInput{
		OrganizationID: orgID,
		GroupID:        orgID,
		AssetID:        assetID,
		Version:        newVersion,
		Name:           state.MCPAssetName.ValueString(),
		MetadataJSON:   metaJSON,
	}); err != nil {
		resp.Diagnostics.AddError("Error republishing generated MCP asset", err.Error())
		return
	}

	if _, err = r.client.UpdateBridgeAssetVersion(ctx, orgID, envID, bridgeID, newVersion); err != nil {
		resp.Diagnostics.AddError("Error moving MCP bridge to new asset version", err.Error())
		return
	}

	ups, err := r.client.GetBridgeUpstreams(ctx, orgID, envID, bridgeID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MCP bridge upstreams", err.Error())
		return
	}
	if err = r.resyncTranscodingPolicies(ctx, orgID, envID, bridgeID, planSources, ups); err != nil {
		resp.Diagnostics.AddError("Error re-syncing MCP bridge transcoding policies", err.Error())
		return
	}

	instance, err := r.client.GetBridge(ctx, orgID, envID, bridgeID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MCP bridge after update", err.Error())
		return
	}

	plannedSources := plan.SourceAPIs
	r.flattenBridge(instance, &plan, orgID, envID)
	plan.SourceAPIs = plannedSources
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MCPBridgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MCPBridgeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	bridgeID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Bridge ID", "Could not parse MCP bridge ID: "+data.ID.ValueString())
		return
	}

	// Deleting the instance cascades its policies + upstreams; then hard-delete the
	// generated Exchange asset version to free the GAV for reuse.
	if err = r.client.DeleteBridgeInstance(ctx, orgID, envID, bridgeID); err != nil {
		resp.Diagnostics.AddError("Error deleting MCP bridge instance", err.Error())
		return
	}
	if !data.AssetID.IsNull() && data.AssetID.ValueString() != "" {
		if err = r.client.DeleteBridgeAssetVersion(ctx, orgID, data.AssetID.ValueString(), data.AssetVersion.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error hard-deleting generated MCP asset", err.Error())
		}
	}
}

func (r *MCPBridgeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: organization_id/environment_id/mcp_bridge_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// --- Helpers ---

func (r *MCPBridgeResource) resolveDeployment(ctx context.Context, orgID, envID, gatewayID string) (*agentstools.MCPBridgeDeployment, error) {
	gw, err := r.client.GetGatewayInfo(ctx, orgID, envID, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve gateway_id %s: %w", gatewayID, err)
	}
	return &agentstools.MCPBridgeDeployment{
		EnvironmentID:  envID,
		Type:           "HY",
		ExpectedStatus: "deployed",
		TargetID:       gw.ID,
		TargetName:     gw.Name,
		GatewayVersion: gw.RuntimeVersion,
	}, nil
}

// toBridgeSources converts the Terraform source_apis list into the normalized
// (Terraform-free) bridgeSource slice consumed by the wire-assembly helpers.
func (r *MCPBridgeResource) toBridgeSources(ctx context.Context, list types.List, orgID string) []bridgeSource {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var models []BridgeSourceAPIModel
	list.ElementsAs(ctx, &models, false)
	out := make([]bridgeSource, 0, len(models))
	for _, m := range models {
		grp := m.GroupID.ValueString()
		if grp == "" {
			grp = orgID
		}
		var toolModels []BridgeToolModel
		m.Tools.ElementsAs(ctx, &toolModels, false)
		tools := make([]bridgeTool, 0, len(toolModels))
		for _, tm := range toolModels {
			var q, h []string
			if !tm.QueryParams.IsNull() && !tm.QueryParams.IsUnknown() {
				tm.QueryParams.ElementsAs(ctx, &q, false)
			}
			if !tm.HeaderParams.IsNull() && !tm.HeaderParams.IsUnknown() {
				tm.HeaderParams.ElementsAs(ctx, &h, false)
			}
			tools = append(tools, bridgeTool{
				Name:         tm.Name.ValueString(),
				Description:  tm.Description.ValueString(),
				Method:       tm.Method.ValueString(),
				Path:         tm.Path.ValueString(),
				QueryParams:  q,
				HeaderParams: h,
				HasBody:      tm.HasBody.ValueBool(),
			})
		}
		out = append(out, bridgeSource{
			Label:       m.Label.ValueString(),
			UpstreamURI: m.UpstreamURI.ValueString(),
			AssetID:     m.AssetID.ValueString(),
			GroupID:     grp,
			Version:     m.Version.ValueString(),
			Tools:       tools,
		})
	}
	return out
}

// attachBridgePolicies attaches the 3 inbound MCP policies + one outbound
// mcp-transcoding policy per source API (mirrors the live 5-policy layout).
func (r *MCPBridgeResource) attachBridgePolicies(ctx context.Context, orgID, envID string, bridgeID int, sources []bridgeSource, ups []agentstools.MCPBridgeUpstreamDetail) error {
	pol := r.client.Policies

	order1, order2, order3 := 1, 2, 3
	if _, err := pol.CreateAPIPolicy(ctx, orgID, envID, bridgeID, &apimanagement.CreateAPIPolicyRequest{
		ConfigurationData: map[string]interface{}{},
		GroupID:           mcpPolicyGroupID,
		AssetID:           mcpSupportAsset,
		AssetVersion:      mcpSupportVersion,
		Order:             &order1,
	}); err != nil {
		return fmt.Errorf("attach mcp-support: %w", err)
	}
	if _, err := pol.CreateAPIPolicy(ctx, orgID, envID, bridgeID, &apimanagement.CreateAPIPolicyRequest{
		ConfigurationData: map[string]interface{}{"validateToolSchema": true},
		GroupID:           mcpPolicyGroupID,
		AssetID:           mcpSchemaValidationAsset,
		AssetVersion:      mcpSchemaValidationVer,
		Order:             &order2,
	}); err != nil {
		return fmt.Errorf("attach mcp-schema-validation: %w", err)
	}
	if _, err := pol.CreateAPIPolicy(ctx, orgID, envID, bridgeID, &apimanagement.CreateAPIPolicyRequest{
		ConfigurationData: routerConfig(sources),
		GroupID:           mcpPolicyGroupID,
		AssetID:           mcpTranscodingRouterAsst,
		AssetVersion:      mcpTranscodingRouterVer,
		Order:             &order3,
	}); err != nil {
		return fmt.Errorf("attach mcp-transcoding-router: %w", err)
	}
	return r.attachTranscodingPolicies(ctx, orgID, envID, bridgeID, sources, ups)
}

// attachTranscodingPolicies attaches one outbound mcp-transcoding policy per source
// API, wired to the matching upstream id.
func (r *MCPBridgeResource) attachTranscodingPolicies(ctx context.Context, orgID, envID string, bridgeID int, sources []bridgeSource, ups []agentstools.MCPBridgeUpstreamDetail) error {
	for _, src := range sources {
		upstreamID := matchUpstreamID(ups, src)
		if upstreamID == "" {
			return fmt.Errorf("could not match an upstream for source API %q (asset %s:%s)", src.Label, src.AssetID, src.Version)
		}
		if _, err := r.client.Policies.CreateOutboundAPIPolicy(ctx, orgID, envID, bridgeID, &apimanagement.CreateOutboundAPIPolicyRequest{
			ConfigurationData: transcodingConfig(src),
			GroupID:           mcpPolicyGroupID,
			AssetID:           mcpTranscodingAsset,
			AssetVersion:      mcpTranscodingVersion,
			UpstreamIDs:       []string{upstreamID},
			APIVersionID:      bridgeID,
		}); err != nil {
			return fmt.Errorf("attach mcp-transcoding for %q: %w", src.Label, err)
		}
	}
	return nil
}

// resyncTranscodingPolicies deletes the existing transcoding-router + per-upstream
// transcoding policies and re-creates them from the updated tool set (support +
// schema-validation are unchanged and left in place).
func (r *MCPBridgeResource) resyncTranscodingPolicies(ctx context.Context, orgID, envID string, bridgeID int, sources []bridgeSource, ups []agentstools.MCPBridgeUpstreamDetail) error {
	pol := r.client.Policies
	existing, err := pol.ListAPIPolicies(ctx, orgID, envID, bridgeID)
	if err != nil {
		return fmt.Errorf("list policies: %w", err)
	}
	for _, p := range existing {
		if p.AssetID == mcpTranscodingRouterAsst || p.AssetID == mcpTranscodingAsset {
			if derr := pol.DeleteAPIPolicy(ctx, orgID, envID, bridgeID, p.ID); derr != nil {
				return fmt.Errorf("delete policy %d (%s): %w", p.ID, p.AssetID, derr)
			}
		}
	}
	if _, err = pol.CreateAPIPolicy(ctx, orgID, envID, bridgeID, &apimanagement.CreateAPIPolicyRequest{
		ConfigurationData: routerConfig(sources),
		GroupID:           mcpPolicyGroupID,
		AssetID:           mcpTranscodingRouterAsst,
		AssetVersion:      mcpTranscodingRouterVer,
	}); err != nil {
		return fmt.Errorf("re-attach mcp-transcoding-router: %w", err)
	}
	return r.attachTranscodingPolicies(ctx, orgID, envID, bridgeID, sources, ups)
}

// bridgeToolMappingFromPolicies delegates to the shared, Terraform-free mapping in the
// client package (kept as an unexported wrapper so the resource's regression test — and
// callers — reference it by the local name). See BridgeToolMappingFromPolicies.
func bridgeToolMappingFromPolicies(policies []apimanagement.APIPolicy) (map[string]interface{}, []interface{}, map[string][]string) {
	return agentstools.BridgeToolMappingFromPolicies(policies)
}

// reconstructSourceAPIs rebuilds the source_apis list from the live instance (routing +
// upstreams) and the per-upstream transcoding policies. Used only on import, where state
// has no source_apis yet. The pure rebuild is shared with the data source via
// agentstools.ReconstructBridgeSources; here we only flatten it into resource-schema
// Terraform types. Tool descriptions live in the asset metadata and are left null (they
// are optional local config).
func (r *MCPBridgeResource) reconstructSourceAPIs(ctx context.Context, orgID, envID string, inst *agentstools.MCPBridge) (types.List, error) {
	srcListType := types.ListType{ElemType: types.ObjectType{AttrTypes: bridgeSourceAttrTypes}}

	ups, err := r.client.GetBridgeUpstreams(ctx, orgID, envID, inst.ID)
	if err != nil {
		return types.ListNull(srcListType.ElemType), err
	}
	policies, err := r.client.Policies.ListAPIPolicies(ctx, orgID, envID, inst.ID)
	if err != nil {
		return types.ListNull(srcListType.ElemType), err
	}

	sources := agentstools.ReconstructBridgeSources(orgID, inst, ups, policies)
	elems := make([]attr.Value, 0, len(sources))
	for _, src := range sources {
		toolElems := flattenReconstructedTools(src.Tools)
		toolsList, _ := types.ListValue(types.ObjectType{AttrTypes: bridgeToolAttrTypes}, toolElems)
		obj, diags := types.ObjectValue(bridgeSourceAttrTypes, map[string]attr.Value{
			"label":        types.StringValue(src.Label),
			"upstream_uri": types.StringValue(src.UpstreamURI),
			"asset_id":     types.StringValue(src.AssetID),
			"group_id":     types.StringValue(src.GroupID),
			"version":      types.StringValue(src.Version),
			"tools":        toolsList,
		})
		if diags.HasError() {
			continue
		}
		elems = append(elems, obj)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: bridgeSourceAttrTypes}, elems)
	if diags.HasError() {
		return types.ListNull(srcListType.ElemType), fmt.Errorf("failed to build reconstructed source_apis")
	}
	return list, nil
}

// flattenReconstructedTools converts shared ReconstructedTool values into resource-schema
// tool objects. A tool name equal to the derived slug is nulled (it is optional local
// config, so surfacing the derived value would show spurious drift after import).
func flattenReconstructedTools(tools []agentstools.ReconstructedTool) []attr.Value {
	out := make([]attr.Value, 0, len(tools))
	for _, t := range tools {
		nameVal := types.StringNull()
		if t.Name != "" && t.Name != agentstools.BridgeToolName(t.Method, t.Path) {
			nameVal = types.StringValue(t.Name)
		}
		qList, _ := types.ListValue(types.StringType, stringsToValues(t.QueryParams))
		hList, _ := types.ListValue(types.StringType, stringsToValues(t.HeaderParams))
		obj, diags := types.ObjectValue(bridgeToolAttrTypes, map[string]attr.Value{
			"name":          nameVal,
			"description":   types.StringNull(),
			"method":        types.StringValue(t.Method),
			"path":          types.StringValue(t.Path),
			"query_params":  qList,
			"header_params": hList,
			"has_body":      types.BoolValue(t.HasBody),
		})
		if diags.HasError() {
			continue
		}
		out = append(out, obj)
	}
	return out
}

func stringsToValues(in []string) []attr.Value {
	out := make([]attr.Value, 0, len(in))
	for _, s := range in {
		out = append(out, types.StringValue(s))
	}
	return out
}

func (r *MCPBridgeResource) flattenBridge(inst *agentstools.MCPBridge, data *MCPBridgeResourceModel, orgID, envID string) {
	data.ID = types.StringValue(strconv.Itoa(inst.ID))
	data.Technology = types.StringValue("flexGateway")
	if inst.AssetID != "" {
		data.AssetID = types.StringValue(inst.AssetID)
	}
	if inst.AssetVersion != "" {
		data.AssetVersion = types.StringValue(inst.AssetVersion)
	}
	if inst.ProductVersion != "" {
		data.ProductVersion = types.StringValue(inst.ProductVersion)
	} else if data.ProductVersion.IsUnknown() {
		data.ProductVersion = types.StringNull()
	}
	if inst.Status != "" {
		data.Status = types.StringValue(inst.Status)
	} else if data.Status.IsUnknown() {
		data.Status = types.StringNull()
	}
	if inst.EndpointURI != "" {
		data.ConsumerEndpoint = types.StringValue(inst.EndpointURI)
	} else {
		data.ConsumerEndpoint = types.StringNull()
	}

	if data.OrganizationID.IsNull() || data.OrganizationID.IsUnknown() || data.OrganizationID.ValueString() == "" {
		data.OrganizationID = types.StringValue(orgID)
	}
	data.EnvironmentID = types.StringValue(envID)

	if inst.Deployment != nil {
		data.Deployment = deploymentToObject(&DeploymentModel{
			EnvironmentID:  types.StringValue(inst.Deployment.EnvironmentID),
			Type:           types.StringValue(inst.Deployment.Type),
			ExpectedStatus: types.StringValue(inst.Deployment.ExpectedStatus),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue(inst.Deployment.TargetID),
			TargetName:     types.StringValue(inst.Deployment.TargetName),
			GatewayVersion: types.StringValue(inst.Deployment.GatewayVersion),
		})
	} else {
		data.Deployment = types.ObjectNull(deploymentAttrTypes)
	}
}

// --- Pure helpers (Terraform-free) ---

func bridgeProxyURI(port int64, basePath string) string {
	if port == 0 {
		port = 8081
	}
	bp := strings.TrimPrefix(strings.TrimSpace(basePath), "/")
	return fmt.Sprintf("http://0.0.0.0:%d/%s", port, bp)
}

// parseBridgeProxyURI extracts the listener port and base path from a bridge proxy URI
// of the form http://0.0.0.0:<port>/<base_path>. Used on import to recover the
// RequiresReplace port/base_path from the live instance endpoint.
func parseBridgeProxyURI(raw string) (port int64, basePath string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0, "", false
	}
	p, err := strconv.ParseInt(u.Port(), 10, 64)
	if err != nil {
		return 0, "", false
	}
	return p, strings.TrimPrefix(u.Path, "/"), true
}

// backfillBridgeImportFields populates the RequiresReplace identity fields (gateway_id,
// mcp_asset_name, port, base_path) from the live instance when they are absent — i.e. after
// `terraform import`, where only org/env/id are seeded. Without this, those fields read as
// empty and the first plan would want to destroy+recreate the bridge. mcp_asset_name is
// recovered as the (sanitized) generated asset ID; an mcp_asset_name containing characters the
// platform strips will surface a one-time diff the user can reconcile.
func backfillBridgeImportFields(data *MCPBridgeResourceModel, inst *agentstools.MCPBridge) {
	if (data.GatewayID.IsNull() || data.GatewayID.ValueString() == "") && inst.Deployment != nil && inst.Deployment.TargetID != "" {
		data.GatewayID = types.StringValue(inst.Deployment.TargetID)
	}
	if (data.MCPAssetName.IsNull() || data.MCPAssetName.ValueString() == "") && inst.AssetID != "" {
		data.MCPAssetName = types.StringValue(inst.AssetID)
	}
	var proxy string
	if inst.Endpoint != nil && inst.Endpoint.ProxyURI != nil {
		proxy = *inst.Endpoint.ProxyURI
	}
	if proxy == "" {
		return
	}
	if port, bp, okp := parseBridgeProxyURI(proxy); okp {
		if data.Port.IsNull() || data.Port.IsUnknown() || data.Port.ValueInt64() == 0 {
			data.Port = types.Int64Value(port)
		}
		if data.BasePath.IsNull() {
			data.BasePath = types.StringValue(bp)
		}
	}
}

// bridgeAssetID sanitizes a server name into a valid Exchange asset ID (spaces -> '-',
// dropping characters outside [A-Za-z0-9._-]; case preserved, as the platform does).
func bridgeAssetID(name string) string {
	var b strings.Builder
	for _, c := range name {
		switch {
		case c == ' ':
			b.WriteByte('-')
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_':
			b.WriteRune(c)
		}
	}
	return b.String()
}

// buildBridgeRouting builds routing[] — one route per source API, matched by the
// X-UPSTREAM-NAME header, carrying the upstream uri + connection at create time.
func buildBridgeRouting(sources []bridgeSource) []agentstools.MCPBridgeRoute {
	routes := make([]agentstools.MCPBridgeRoute, 0, len(sources))
	for _, s := range sources {
		routes = append(routes, agentstools.MCPBridgeRoute{
			Label: s.Label,
			Rules: &agentstools.MCPBridgeRules{Headers: map[string]string{"X-UPSTREAM-NAME": s.Label}},
			Upstreams: []agentstools.MCPBridgeRouteUpstream{{
				Weight: 100,
				URI:    s.UpstreamURI,
				Connection: &agentstools.MCPBridgeConnection{
					Label:   s.Label,
					AssetID: s.AssetID,
					GroupID: s.GroupID,
					Version: s.Version,
				},
			}},
		})
	}
	return routes
}

// matchUpstreamID finds the server-assigned upstream id for a source API, matching by
// the connection asset (assetId+version) first, then falling back to the backend uri.
func matchUpstreamID(ups []agentstools.MCPBridgeUpstreamDetail, src bridgeSource) string {
	for _, u := range ups {
		if u.Connection != nil && u.Connection.AssetID == src.AssetID && u.Connection.Version == src.Version {
			return u.ID
		}
	}
	for _, u := range ups {
		if u.URI == src.UpstreamURI {
			return u.ID
		}
	}
	return ""
}

// requiresReplaceOnStructuralChange forces the bridge to be replaced when a source
// API's structural identity changes (label, upstream_uri, asset_id, group_id, version,
// or adding/removing a source API) — those reshape the routing/upstreams, which the
// durable update path cannot patch. Tool-only edits leave the structural signature
// unchanged and update in place. Making this a plan modifier means `terraform plan`
// SHOWS the replacement up front, instead of the Update guard erroring at apply time.
func requiresReplaceOnStructuralChange() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
			// Skip create (no prior state) and unknown plans (e.g. tools sourced from a
			// not-yet-read data source): nothing structural to compare.
			if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
				return
			}
			if structuralSignatureFromList(ctx, req.StateValue) != structuralSignatureFromList(ctx, req.PlanValue) {
				resp.RequiresReplace = true
			}
		},
		"A source API structural change (label, upstream_uri, asset_id, group_id, version, or adding/removing a source API) requires replacement; tool-only edits update in place.",
		"A source API structural change (`label`, `upstream_uri`, `asset_id`, `group_id`, `version`, or adding/removing a source API) requires replacement; tool-only edits update in place.",
	)
}

// structuralSignatureFromList decodes a source_apis list value and returns its
// structural signature (ignoring tools). Returns "" if the list cannot be decoded.
func structuralSignatureFromList(ctx context.Context, l types.List) string {
	var models []BridgeSourceAPIModel
	if diags := l.ElementsAs(ctx, &models, false); diags.HasError() {
		return ""
	}
	sources := make([]bridgeSource, 0, len(models))
	for _, m := range models {
		sources = append(sources, bridgeSource{
			Label:       m.Label.ValueString(),
			UpstreamURI: m.UpstreamURI.ValueString(),
			AssetID:     m.AssetID.ValueString(),
			GroupID:     m.GroupID.ValueString(),
			Version:     m.Version.ValueString(),
		})
	}
	return bridgeStructuralSignature(sources)
}

// bridgeStructuralSignature is the identity of a bridge's routing/upstreams — the parts
// that cannot be patched in place. It deliberately EXCLUDES tools (which update in place).
func bridgeStructuralSignature(sources []bridgeSource) string {
	parts := make([]string, 0, len(sources))
	for _, s := range sources {
		parts = append(parts, strings.Join([]string{s.Label, s.UpstreamURI, s.AssetID, s.GroupID, s.Version}, "\x1f"))
	}
	return strings.Join(parts, "\x1e")
}

// bridgeToolSignature is the identity of a bridge's full tool set across all source APIs
// (name, description, method, path, params, body). It changes exactly when an update needs
// to republish the generated asset (and bump asset_version). Used by ModifyPlan to decide
// whether the computed asset_version must be re-planned as unknown.
func bridgeToolSignature(sources []bridgeSource) string {
	parts := make([]string, 0, len(sources))
	for _, s := range sources {
		parts = append(parts, s.Label)
		for _, t := range s.Tools {
			parts = append(parts, strings.Join([]string{
				t.Name, t.Description, t.Method, t.Path,
				strings.Join(t.QueryParams, ","), strings.Join(t.HeaderParams, ","),
				fmt.Sprintf("%t", t.HasBody),
			}, "\x1f"))
		}
	}
	return strings.Join(parts, "\x1e")
}
