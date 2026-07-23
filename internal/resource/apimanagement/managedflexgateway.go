package apimanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &ManagedOmniGatewayResource{}
	_ resource.ResourceWithConfigure   = &ManagedOmniGatewayResource{}
	_ resource.ResourceWithImportState = &ManagedOmniGatewayResource{}
)

// attr type maps — used to build and validate types.Object values for nested blocks.
var (
	ingressAttrTypes = map[string]attr.Type{
		"public_url":          types.StringType,
		"internal_url":        types.StringType,
		"forward_ssl_session": types.BoolType,
		"last_mile_security":  types.BoolType,
	}
	propertiesAttrTypes = map[string]attr.Type{
		"upstream_response_timeout": types.Int64Type,
		"connection_idle_timeout":   types.Int64Type,
	}
	loggingAttrTypes = map[string]attr.Type{
		"level":        types.StringType,
		"forward_logs": types.BoolType,
	}
	tracingLabelAttrTypes = map[string]attr.Type{
		"type":          types.StringType,
		"name":          types.StringType,
		"default_value": types.StringType,
		"key_name":      types.StringType,
	}
	tracingAttrTypes = map[string]attr.Type{
		"enabled":  types.BoolType,
		"sampling": types.Int64Type,
		"labels":   types.ListType{ElemType: types.ObjectType{AttrTypes: tracingLabelAttrTypes}},
	}
)

// ManagedOmniGatewayResource is the resource implementation.
type ManagedOmniGatewayResource struct {
	client *apimanagement.ManagedOmniGatewayClient
}

// ManagedOmniGatewayResourceModel is the Terraform state model.
// Nested optional+computed blocks use types.Object so the framework can
// represent the "unknown" value during plan without a conversion error.
type ManagedOmniGatewayResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	TargetID       types.String `tfsdk:"target_id"`
	TargetType     types.String `tfsdk:"target_type"`
	RuntimeVersion types.String `tfsdk:"runtime_version"`
	ReleaseChannel types.String `tfsdk:"release_channel"`
	Size           types.String `tfsdk:"size"`
	Status         types.String `tfsdk:"status"`
	Ingress        types.Object `tfsdk:"ingress"`
	Properties     types.Object `tfsdk:"properties"`
	Logging        types.Object `tfsdk:"logging"`
	Tracing        types.Object `tfsdk:"tracing"`
}

func NewManagedOmniGatewayResource() resource.Resource {
	return &ManagedOmniGatewayResource{}
}

func (r *ManagedOmniGatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_omni_gateway"
}

func (r *ManagedOmniGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudHub 2.0 Managed Omni Gateway instance in Anypoint Platform.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the managed Omni Gateway.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the managed Omni Gateway.",
				Required:    true,
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
				Description: "The environment ID where the gateway will be deployed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_id": schema.StringAttribute{
				Description: "The target ID for the gateway deployment. For a private space this is the " +
					"private space UUID; for a CloudHub 2.0 shared space this is the region slug " +
					"(e.g. 'cloudhub-us-east-1'). The provider auto-detects the target type — no extra " +
					"configuration is needed to deploy into a shared space.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_type": schema.StringAttribute{
				Description: "The type of the deployment target, resolved by the provider: " +
					"'private-space' or 'shared-space'.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"runtime_version": schema.StringAttribute{
				Description: "The Omni Gateway runtime version (e.g., '1.9.9'). " +
					"If omitted, the provider auto-selects the latest version for the chosen release_channel.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"release_channel": schema.StringAttribute{
				Description: "The release channel for the gateway. Valid values: 'lts', 'edge'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("lts"),
				Validators: []validator.String{
					stringvalidator.OneOf("lts", "edge"),
				},
			},
			"size": schema.StringAttribute{
				Description: "The size of the gateway instance. Valid values: 'small', 'large'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("small"),
				Validators: []validator.String{
					stringvalidator.OneOf("small", "large"),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the managed Omni Gateway.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ingress": schema.SingleNestedAttribute{
				Description: "Ingress configuration for the gateway.\n\n" +
					"  - `public_url` and `internal_url` are auto-derived from the target's domain when omitted.\n" +
					"  - Set `public_url` to your own value (e.g. a custom domain) to override the auto-derived URL.\n" +
					"  - To reset back to auto-derived after having set a custom value, set `public_url = \"\"`.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"public_url": schema.StringAttribute{
						Description: "The public URL for the gateway ingress. " +
							"Auto-derived from the target domain when empty.",
						Optional: true,
						Computed: true,
					},
					"internal_url": schema.StringAttribute{
						Description: "The internal URL for the gateway ingress. " +
							"Auto-derived from the target domain when empty.",
						Optional: true,
						Computed: true,
					},
					"forward_ssl_session": schema.BoolAttribute{
						Description: "Whether to forward SSL sessions to upstream services.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"last_mile_security": schema.BoolAttribute{
						Description: "Whether to enable last-mile security (TLS between gateway and upstream).",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
				},
			},
			"properties": schema.SingleNestedAttribute{
				Description: "Runtime properties for the gateway.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"upstream_response_timeout": schema.Int64Attribute{
						Description: "Timeout in seconds for upstream service responses.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(15),
					},
					"connection_idle_timeout": schema.Int64Attribute{
						Description: "Timeout in seconds for idle connections.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(60),
					},
				},
			},
			"logging": schema.SingleNestedAttribute{
				Description: "Logging configuration for the gateway.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"level": schema.StringAttribute{
						Description: "The log level. Valid values: 'debug', 'info', 'warn', 'error'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("info"),
						Validators: []validator.String{
							stringvalidator.OneOf("debug", "info", "warn", "error"),
						},
					},
					"forward_logs": schema.BoolAttribute{
						Description: "Whether to forward logs to Anypoint Monitoring.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
				},
			},
			"tracing": schema.SingleNestedAttribute{
				Description: "Distributed tracing configuration for the gateway.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Whether distributed tracing is enabled.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"sampling": schema.Int64Attribute{
						Description: "Sampling rate as a percentage (1–100). Defaults to 1.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(1),
					},
					"labels": schema.ListNestedAttribute{
						Description: "Labels attached to tracing spans.",
						Optional:    true,
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "Label source type. Valid values: 'environment', 'requestHeader', 'literal'.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.OneOf("environment", "requestHeader", "literal"),
									},
								},
								"name": schema.StringAttribute{
									Description: "Display name of the label.",
									Required:    true,
								},
								"default_value": schema.StringAttribute{
									Description: "Default value for the label.",
									Required:    true,
								},
								"key_name": schema.StringAttribute{
									Description: "Key name for 'environment' and 'requestHeader' types. Not applicable for 'literal'.",
									Optional:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ManagedOmniGatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	gwClient, err := apimanagement.NewManagedOmniGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Managed Omni Gateway API Client",
			"An unexpected error occurred when creating the Managed Omni Gateway API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = gwClient
}

// --- CRUD ---

func (r *ManagedOmniGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedOmniGatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	gwName := data.Name.ValueString()
	targetID := data.TargetID.ValueString()
	releaseChannel := data.ReleaseChannel.ValueString()

	runtimeVersion := data.RuntimeVersion.ValueString()
	if runtimeVersion == "" {
		versionsResp, err := r.client.GetGatewayVersions(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Error fetching gateway versions",
				"Could not fetch available gateway versions: "+err.Error())
			return
		}
		runtimeVersion = versionsResp.LatestVersionForChannel(releaseChannel)
		if runtimeVersion == "" {
			resp.Diagnostics.AddError("No versions available",
				fmt.Sprintf("No runtime versions found for release channel %q", releaseChannel))
			return
		}
		tflog.Info(ctx, "Auto-selected runtime version", map[string]interface{}{
			"channel": releaseChannel,
			"version": runtimeVersion,
		})
		data.RuntimeVersion = types.StringValue(runtimeVersion)
	}

	cfg := r.expandConfiguration(data)

	// Detect the target type up front. The platform derives private-vs-shared
	// server-side from target_id (the create body carries no type field), but the
	// provider must know it to decide how to handle ingress URLs. Detection is
	// best-effort: on error or "not found" we fall back to private-space behavior,
	// which is the historical default.
	detectedType, detectErr := r.client.GetTargetType(ctx, orgID, targetID)
	if detectErr != nil {
		tflog.Warn(ctx, "Could not resolve target type; assuming private-space",
			map[string]interface{}{"target_id": targetID, "error": detectErr.Error()})
	}
	isSharedSpace := detectedType == apimanagement.TargetTypeSharedSpace

	if isSharedSpace {
		// Shared space (CloudHub 2.0): the platform assigns the public URL (a
		// random-slug host the client cannot compute) and there is no internal
		// URL. Never derive or send ingress URLs — clear them so the platform
		// assigns, and read the assigned public_url back from the create response.
		cfg.Ingress.PublicURL = ""
		cfg.Ingress.InternalURL = ""
	} else if cfg.Ingress.PublicURL == "" || cfg.Ingress.InternalURL == "" {
		// Private space: auto-derive public_url / internal_url from the target's
		// provisioned domains when the user did not supply them.
		domainsResp, err := r.client.GetDomains(ctx, orgID, targetID, envID)
		if err != nil {
			if client.IsNotFound(err) {
				resp.Diagnostics.AddError(
					"Target not found or domains not provisioned",
					fmt.Sprintf("Could not fetch domains for target_id %q in environment %q. "+
						"Verify that the target (private space) exists and is fully provisioned. "+
						"If the private space was recently created, wait a few minutes and retry.",
						targetID, envID),
				)
			} else {
				resp.Diagnostics.AddError("Error fetching domains", "Could not fetch domains for target: "+err.Error())
			}
			return
		}
		derivedPublic, derivedInternal := apimanagement.BuildIngressURLs(gwName, domainsResp.Domains)
		if len(derivedPublic) == 0 {
			resp.Diagnostics.AddError(
				"No domains available for target",
				fmt.Sprintf("The target %q in environment %q has no domains provisioned yet. "+
					"This usually means the private space is still initializing. "+
					"Please wait for the private space to be fully provisioned and retry.", targetID, envID),
			)
			return
		}
		if cfg.Ingress.PublicURL == "" {
			cfg.Ingress.PublicURL = derivedPublic[0]
		}
		if cfg.Ingress.InternalURL == "" {
			cfg.Ingress.InternalURL = derivedInternal
		}
	}

	createReq := &apimanagement.CreateManagedOmniGatewayRequest{
		Name:           gwName,
		TargetID:       targetID,
		RuntimeVersion: runtimeVersion,
		ReleaseChannel: releaseChannel,
		Size:           data.Size.ValueString(),
		Configuration:  cfg,
	}

	// Save plan tracing before the API call — the Create response may not
	// echo configuration.tracing back, which would leave Enabled=false and
	// trigger a "provider produced an unexpected new value" framework error.
	planTracing := data.Tracing

	gw, err := r.client.CreateManagedOmniGateway(ctx, orgID, envID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating managed Omni Gateway", "Could not create managed Omni Gateway, unexpected error: "+err.Error())
		return
	}

	r.flattenGateway(gw, &data, orgID, envID)
	data.Tracing = reconcileTracing(planTracing, data.Tracing)
	// The create (api/v1) response may omit targetType; fall back to the value we
	// detected so the computed attribute settles to a concrete value on create.
	if data.TargetType.ValueString() == "" && detectedType != "" {
		data.TargetType = types.StringValue(detectedType)
	}
	tflog.Trace(ctx, "created managed omni gateway", map[string]interface{}{"id": gw.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedOmniGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedOmniGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	gw, err := r.client.GetManagedOmniGateway(ctx, orgID, envID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading managed Omni Gateway", "Could not read managed Omni Gateway ID "+data.ID.ValueString()+": "+err.Error())
		return
	}

	r.flattenGateway(gw, &data, orgID, envID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedOmniGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ManagedOmniGatewayResourceModel
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

	cfg := r.expandConfiguration(plan)

	// Shared-space targets have a platform-assigned public URL and no internal
	// URL. target_type is already resolved in state (from Create/Read), so no
	// extra API call is needed. Skip domain derivation entirely — the plan's
	// ingress URLs (carried forward from state via UseStateForUnknown) already
	// hold the platform-assigned public_url, which is the safe value to echo back.
	isSharedSpace := state.TargetType.ValueString() == apimanagement.TargetTypeSharedSpace

	// Resolve public_url / internal_url: use user-provided values when set,
	// otherwise auto-derive from the target's domain (same logic as Create).
	// Private space only — shared space skips derivation.
	if !isSharedSpace && (cfg.Ingress.PublicURL == "" || cfg.Ingress.InternalURL == "") {
		targetID := state.TargetID.ValueString()
		gwName := plan.Name.ValueString()
		domainsResp, err := r.client.GetDomains(ctx, orgID, targetID, envID)
		if err != nil {
			tflog.Warn(ctx, "Could not fetch domains for URL derivation during update; keeping existing URLs",
				map[string]interface{}{"error": err.Error()})
		} else {
			derivedPublic, derivedInternal := apimanagement.BuildIngressURLs(gwName, domainsResp.Domains)
			if cfg.Ingress.PublicURL == "" && len(derivedPublic) > 0 {
				cfg.Ingress.PublicURL = derivedPublic[0]
			}
			if cfg.Ingress.InternalURL == "" {
				cfg.Ingress.InternalURL = derivedInternal
			}
		}
	}

	// PUT requires the full object — send everything from the plan.
	updateReq := &apimanagement.UpdateManagedOmniGatewayRequest{
		Name:           plan.Name.ValueString(),
		TargetID:       state.TargetID.ValueString(), // target_id is RequiresReplace, so state == plan
		RuntimeVersion: plan.RuntimeVersion.ValueString(),
		ReleaseChannel: plan.ReleaseChannel.ValueString(),
		Size:           plan.Size.ValueString(),
		Configuration:  cfg,
	}

	// Save plan tracing before the API call — the Update response may not
	// echo configuration.tracing back (same issue as Create).
	planTracing := plan.Tracing

	gw, err := r.client.UpdateManagedOmniGateway(ctx, orgID, envID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating managed Omni Gateway", "Could not update managed Omni Gateway, unexpected error: "+err.Error())
		return
	}

	r.flattenGateway(gw, &plan, orgID, envID)
	plan.Tracing = reconcileTracing(planTracing, plan.Tracing)
	// Retain prior status to avoid a Terraform consistency-check error when the
	// gateway re-enters its async provisioning cycle (RUNNING → APPLYING) after a
	// size or config change. The framework compares post-apply state against the
	// plan value (which inherits the prior RUNNING status via UseStateForUnknown)
	// and errors when it sees APPLYING. Read() will pick up the live status on the
	// next refresh cycle.
	plan.Status = state.Status
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ManagedOmniGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedOmniGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	err := r.client.DeleteManagedOmniGateway(ctx, orgID, envID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting managed Omni Gateway", "Could not delete managed Omni Gateway, unexpected error: "+err.Error())
		return
	}
}

// ImportState supports two import ID formats:
//   - "<environmentID>/<gatewayID>"                   — falls back to root org (backwards compatible)
//   - "<organizationID>/<environmentID>/<gatewayID>"  — required when the gateway lives in a sub-org
func (r *ManagedOmniGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	case 3:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: environment_id/gateway_id or organization_id/environment_id/gateway_id")
	}
}

// --- Helpers ---

// expandConfiguration converts the Terraform model into the API configuration struct.
// All nested blocks are types.Object so we pull values via .Attributes().
// When a block is null/unknown (i.e. the user omitted it), we fall back to the same
// defaults declared in the schema so the API never receives invalid zero values.
func (r *ManagedOmniGatewayResource) expandConfiguration(data ManagedOmniGatewayResourceModel) apimanagement.ManagedOmniGatewayConfig {
	cfg := apimanagement.ManagedOmniGatewayConfig{}

	if !data.Ingress.IsNull() && !data.Ingress.IsUnknown() {
		attrs := data.Ingress.Attributes()
		cfg.Ingress = apimanagement.IngressConfig{
			PublicURL:         attrs["public_url"].(types.String).ValueString(),
			InternalURL:       attrs["internal_url"].(types.String).ValueString(),
			ForwardSSLSession: attrs["forward_ssl_session"].(types.Bool).ValueBool(),
			LastMileSecurity:  attrs["last_mile_security"].(types.Bool).ValueBool(),
		}
	} else {
		cfg.Ingress = apimanagement.IngressConfig{
			ForwardSSLSession: true,
			LastMileSecurity:  true,
		}
	}

	if !data.Properties.IsNull() && !data.Properties.IsUnknown() {
		attrs := data.Properties.Attributes()
		cfg.Properties = apimanagement.PropertiesConfig{
			UpstreamResponseTimeout: int(attrs["upstream_response_timeout"].(types.Int64).ValueInt64()),
			ConnectionIdleTimeout:   int(attrs["connection_idle_timeout"].(types.Int64).ValueInt64()),
		}
	} else {
		cfg.Properties = apimanagement.PropertiesConfig{
			UpstreamResponseTimeout: 15,
			ConnectionIdleTimeout:   60,
		}
	}

	if !data.Logging.IsNull() && !data.Logging.IsUnknown() {
		attrs := data.Logging.Attributes()
		cfg.Logging = apimanagement.LoggingConfig{
			Level:       attrs["level"].(types.String).ValueString(),
			ForwardLogs: attrs["forward_logs"].(types.Bool).ValueBool(),
		}
	} else {
		cfg.Logging = apimanagement.LoggingConfig{
			Level:       "info",
			ForwardLogs: true,
		}
	}

	if !data.Tracing.IsNull() && !data.Tracing.IsUnknown() {
		attrs := data.Tracing.Attributes()
		tc := apimanagement.TracingConfig{
			Enabled:  attrs["enabled"].(types.Bool).ValueBool(),
			Sampling: int(attrs["sampling"].(types.Int64).ValueInt64()),
		}
		if labelsVal, ok := attrs["labels"].(types.List); ok && !labelsVal.IsNull() && !labelsVal.IsUnknown() {
			for _, elem := range labelsVal.Elements() {
				obj := elem.(types.Object)
				la := obj.Attributes()
				keyName := ""
				if kn, ok := la["key_name"].(types.String); ok && !kn.IsNull() && !kn.IsUnknown() {
					keyName = kn.ValueString()
				}
				tc.Labels = append(tc.Labels, apimanagement.TracingLabel{
					Type:         la["type"].(types.String).ValueString(),
					Name:         la["name"].(types.String).ValueString(),
					DefaultValue: la["default_value"].(types.String).ValueString(),
					KeyName:      keyName,
				})
			}
		}
		cfg.Tracing = tc
	} else {
		cfg.Tracing = apimanagement.TracingConfig{
			Enabled:  false,
			Sampling: 1,
		}
	}

	return cfg
}

// flattenGateway maps the API response into the Terraform state model.
// Nested blocks are stored as types.Object to handle unknown values during plan.
func (r *ManagedOmniGatewayResource) flattenGateway(gw *apimanagement.ManagedOmniGateway, data *ManagedOmniGatewayResourceModel, orgID, envID string) {
	data.ID = types.StringValue(gw.ID)
	data.Name = types.StringValue(gw.Name)
	data.TargetID = types.StringValue(gw.TargetID)
	// target_type is only reliably present on the xapi/v1 single-GET response;
	// the create/update (api/v1) response may omit it. Preserve any prior known
	// value rather than clobbering it with an empty string, so state stays stable.
	if gw.TargetType != "" {
		data.TargetType = types.StringValue(gw.TargetType)
	} else if data.TargetType.IsNull() || data.TargetType.IsUnknown() {
		data.TargetType = types.StringValue(gw.TargetType)
	}
	data.RuntimeVersion = types.StringValue(gw.RuntimeVersion)
	data.ReleaseChannel = types.StringValue(gw.ReleaseChannel)
	data.Size = types.StringValue(gw.Size)
	data.Status = types.StringValue(gw.Status)

	if data.OrganizationID.IsNull() || data.OrganizationID.IsUnknown() || data.OrganizationID.ValueString() == "" {
		data.OrganizationID = types.StringValue(orgID)
	}
	data.EnvironmentID = types.StringValue(envID)

	ingressObj, ingressDiags := types.ObjectValue(ingressAttrTypes, map[string]attr.Value{
		"public_url":          types.StringValue(gw.Configuration.Ingress.PublicURL),
		"internal_url":        types.StringValue(gw.Configuration.Ingress.InternalURL),
		"forward_ssl_session": types.BoolValue(gw.Configuration.Ingress.ForwardSSLSession),
		"last_mile_security":  types.BoolValue(gw.Configuration.Ingress.LastMileSecurity),
	})
	if ingressDiags.HasError() {
		data.Ingress = types.ObjectNull(ingressAttrTypes)
	} else {
		data.Ingress = ingressObj
	}

	propertiesObj, propDiags := types.ObjectValue(propertiesAttrTypes, map[string]attr.Value{
		"upstream_response_timeout": types.Int64Value(int64(gw.Configuration.Properties.UpstreamResponseTimeout)),
		"connection_idle_timeout":   types.Int64Value(int64(gw.Configuration.Properties.ConnectionIdleTimeout)),
	})
	if propDiags.HasError() {
		data.Properties = types.ObjectNull(propertiesAttrTypes)
	} else {
		data.Properties = propertiesObj
	}

	loggingObj, logDiags := types.ObjectValue(loggingAttrTypes, map[string]attr.Value{
		"level":        types.StringValue(gw.Configuration.Logging.Level),
		"forward_logs": types.BoolValue(gw.Configuration.Logging.ForwardLogs),
	})
	if logDiags.HasError() {
		data.Logging = types.ObjectNull(loggingAttrTypes)
	} else {
		data.Logging = loggingObj
	}

	labelElems := make([]attr.Value, 0, len(gw.Configuration.Tracing.Labels))
	for _, l := range gw.Configuration.Tracing.Labels {
		keyName := types.StringNull()
		if l.KeyName != "" {
			keyName = types.StringValue(l.KeyName)
		}
		labelObj, _ := types.ObjectValue(tracingLabelAttrTypes, map[string]attr.Value{
			"type":          types.StringValue(l.Type),
			"name":          types.StringValue(l.Name),
			"default_value": types.StringValue(l.DefaultValue),
			"key_name":      keyName,
		})
		labelElems = append(labelElems, labelObj)
	}
	labelsList, _ := types.ListValue(types.ObjectType{AttrTypes: tracingLabelAttrTypes}, labelElems)

	sampling := gw.Configuration.Tracing.Sampling
	if sampling == 0 {
		sampling = 1
	}
	tracingObj, traceDiags := types.ObjectValue(tracingAttrTypes, map[string]attr.Value{
		"enabled":  types.BoolValue(gw.Configuration.Tracing.Enabled),
		"sampling": types.Int64Value(int64(sampling)),
		"labels":   labelsList,
	})
	if traceDiags.HasError() {
		data.Tracing = types.ObjectNull(tracingAttrTypes)
	} else {
		data.Tracing = tracingObj
	}
}

// reconcileTracing returns the API-returned tracing value when the API response
// looks complete, or falls back to the plan value when the Gateway Manager
// POST/PUT response silently drops tracing fields (e.g. returns enabled=false
// after we sent enabled=true). This prevents "provider produced an unexpected
// new value" framework errors. Read() always uses the live API value.
func reconcileTracing(plan, fromAPI types.Object) types.Object {
	// The plan value is only usable as a fallback when it is a concrete (known,
	// non-null) object. On Create the user typically OMITS `tracing`, so the
	// Optional+Computed plan value is UNKNOWN — there is nothing to preserve from
	// the plan, and the flattened API value is the authoritative computed result.
	// Returning the (unknown) plan on this path would leave a Computed attribute
	// unknown after apply, which the framework rejects with "Provider returned
	// invalid result object after apply". Always prefer the known API value when
	// the plan is not concrete. (This guard must come first: every later branch
	// may `return plan`, which is only safe once we know plan is concrete.)
	if plan.IsNull() || plan.IsUnknown() {
		return fromAPI
	}
	// If flatten could not produce a usable API value, fall back to the (now
	// known) plan value.
	if fromAPI.IsNull() || fromAPI.IsUnknown() {
		return plan
	}
	apiAttrs := fromAPI.Attributes()
	planAttrs := plan.Attributes()
	if apiAttrs == nil || planAttrs == nil {
		return plan
	}
	apiEnabled, ok1 := apiAttrs["enabled"].(types.Bool)
	planEnabled, ok2 := planAttrs["enabled"].(types.Bool)
	if !ok1 || !ok2 {
		return plan
	}
	// If the API echoed false but the plan requested true, the API silently
	// dropped the tracing block — keep the full plan value in state.
	if !apiEnabled.ValueBool() && planEnabled.ValueBool() {
		return plan
	}
	return fromAPI
}
