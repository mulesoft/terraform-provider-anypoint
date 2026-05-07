package secretsmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
)

var (
	_ resource.Resource                = &TLSContextResource{}
	_ resource.ResourceWithConfigure   = &TLSContextResource{}
	_ resource.ResourceWithImportState = &TLSContextResource{}
)

type TLSContextResource struct {
	client *secretsmanagement.TLSContextClient
}

type TLSContextResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	OrganizationID             types.String `tfsdk:"organization_id"`
	EnvironmentID              types.String `tfsdk:"environment_id"`
	SecretGroupID              types.String `tfsdk:"secret_group_id"`
	Name                       types.String `tfsdk:"name"`
	Target                     types.String `tfsdk:"target"`
	KeystoreID                 types.String `tfsdk:"keystore_id"`
	TruststoreID               types.String `tfsdk:"truststore_id"`
	MinTLSVersion              types.String `tfsdk:"min_tls_version"`
	MaxTLSVersion              types.String `tfsdk:"max_tls_version"`
	AlpnProtocols              types.List   `tfsdk:"alpn_protocols"`
	CipherSuites               types.List   `tfsdk:"cipher_suites"`
	EnableClientCertValidation types.Bool   `tfsdk:"enable_client_cert_validation"`
	SkipServerCertValidation   types.Bool   `tfsdk:"skip_server_cert_validation"`
	ExpirationDate             types.String `tfsdk:"expiration_date"`
}

func NewTLSContextResource() resource.Resource {
	return &TLSContextResource{}
}

func (r *TLSContextResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_tls_context"
}

func (r *TLSContextResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Omni Gateway TLS context within a secret group in Anypoint Secrets Manager. " +
			"The target is automatically set to 'OmniGateway'. " +
			"References keystore and truststore resources by their IDs — the provider " +
			"automatically builds the internal path references (keystores/{id}, truststores/{id}).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the TLS context.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: "Environment ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_group_id": schema.StringAttribute{
				Description: "Secret group ID that this TLS context belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the TLS context.",
				Required:    true,
			},
			"target": schema.StringAttribute{
				Description: "Target runtime for the TLS context. Always 'OmniGateway' for this resource.",
				Computed:    true,
				Default:     stringdefault.StaticString("OmniGateway"),
			},
			"keystore_id": schema.StringAttribute{
				Description: "ID of the keystore in the same secret group. " +
					"Use anypoint_secret_group_keystore.example.id to reference it.",
				Optional: true,
			},
			"truststore_id": schema.StringAttribute{
				Description: "ID of the truststore in the same secret group. " +
					"Use anypoint_secret_group_truststore.example.id to reference it.",
				Optional: true,
			},
			"min_tls_version": schema.StringAttribute{
				Description: "Minimum TLS version. Valid values: TLSv1.1, TLSv1.2, TLSv1.3. Defaults to TLSv1.3.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("TLSv1.3"),
				Validators: []validator.String{
					stringvalidator.OneOf("TLSv1.1", "TLSv1.2", "TLSv1.3"),
				},
			},
			"max_tls_version": schema.StringAttribute{
				Description: "Maximum TLS version. Valid values: TLSv1.1, TLSv1.2, TLSv1.3. Defaults to TLSv1.3.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("TLSv1.3"),
				Validators: []validator.String{
					stringvalidator.OneOf("TLSv1.1", "TLSv1.2", "TLSv1.3"),
				},
			},
			"alpn_protocols": schema.ListAttribute{
				Description: "ALPN protocol negotiation list. Valid element values: 'h2', 'http/1.1'. " +
					"Order determines preference: [\"h2\", \"http/1.1\"] prefers H2, " +
					"[\"http/1.1\", \"h2\"] prefers HTTP/1.1.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("h2", "http/1.1"),
					),
				},
			},
			"cipher_suites": schema.ListAttribute{
				Description: "Allowed cipher suites. Empty list means use defaults.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"enable_client_cert_validation": schema.BoolAttribute{
				Description: "Enable mutual TLS client certificate validation (inbound).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"skip_server_cert_validation": schema.BoolAttribute{
				Description: "Skip server certificate validation (outbound).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"expiration_date": schema.StringAttribute{
				Description: "Expiration date of the TLS context.",
				Computed:    true,
			},
		},
	}
}

func (r *TLSContextResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	tlsClient, err := secretsmanagement.NewTLSContextClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create TLS Context Client",
			"An unexpected error occurred: "+err.Error(),
		)
		return
	}
	r.client = tlsClient
}

func (r *TLSContextResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TLSContextResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	sgID := data.SecretGroupID.ValueString()

	createReq := r.expandTLSContext(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tls, err := r.client.CreateTLSContext(ctx, orgID, envID, sgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating TLS context", "Could not create TLS context: "+err.Error())
		return
	}

	r.flattenTLSContext(ctx, tls, &data, orgID, envID, sgID, &resp.Diagnostics)
	tflog.Trace(ctx, "created TLS context", map[string]interface{}{"id": tls.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TLSContextResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TLSContextResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	sgID := data.SecretGroupID.ValueString()

	tls, err := r.client.GetTLSContext(ctx, orgID, envID, sgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading TLS context", "Could not read TLS context: "+err.Error())
		return
	}

	r.flattenTLSContext(ctx, tls, &data, orgID, envID, sgID, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TLSContextResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TLSContextResourceModel
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
	sgID := state.SecretGroupID.ValueString()

	updateReq := r.expandTLSContext(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tls, err := r.client.UpdateTLSContext(ctx, orgID, envID, sgID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating TLS context", "Could not update TLS context: "+err.Error())
		return
	}

	r.flattenTLSContext(ctx, tls, &plan, orgID, envID, sgID, &resp.Diagnostics)
	tflog.Trace(ctx, "updated TLS context", map[string]interface{}{"id": tls.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TLSContextResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The SM API has no individual sub-resource DELETE endpoint (returns 405).
	// Sub-resources are removed by deleting the parent secret group.
	// Removing from Terraform state only; platform cleanup is the secret group's responsibility.
	var data TLSContextResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	tflog.Trace(ctx, "removed TLS context from state (no-op delete)", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *TLSContextResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/secret_group_id/tls_context_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// --- Helpers ---

func (r *TLSContextResource) expandTLSContext(ctx context.Context, data *TLSContextResourceModel, _ *diag.Diagnostics) *secretsmanagement.TLSContext {
	target := data.Target.ValueString()
	if target == "OmniGateway" {
		target = "FlexGateway"
	}
	tlsCtx := &secretsmanagement.TLSContext{
		Name:   data.Name.ValueString(),
		Target: target,
	}

	// Build keystore path reference from ID
	if !data.KeystoreID.IsNull() && !data.KeystoreID.IsUnknown() && data.KeystoreID.ValueString() != "" {
		tlsCtx.Keystore = &secretsmanagement.TLSContextPathRef{
			Path: "keystores/" + data.KeystoreID.ValueString(),
		}
	}

	// Build truststore path reference from ID
	if !data.TruststoreID.IsNull() && !data.TruststoreID.IsUnknown() && data.TruststoreID.ValueString() != "" {
		tlsCtx.Truststore = &secretsmanagement.TLSContextPathRef{
			Path: "truststores/" + data.TruststoreID.ValueString(),
		}
	}

	if !data.MinTLSVersion.IsNull() && !data.MinTLSVersion.IsUnknown() {
		tlsCtx.MinTLSVersion = data.MinTLSVersion.ValueString()
	}
	if !data.MaxTLSVersion.IsNull() && !data.MaxTLSVersion.IsUnknown() {
		tlsCtx.MaxTLSVersion = data.MaxTLSVersion.ValueString()
	}

	if !data.AlpnProtocols.IsNull() && !data.AlpnProtocols.IsUnknown() {
		var alpn []string
		data.AlpnProtocols.ElementsAs(ctx, &alpn, false)
		tlsCtx.AlpnProtocols = alpn
	}

	if !data.CipherSuites.IsNull() && !data.CipherSuites.IsUnknown() {
		var ciphers []string
		data.CipherSuites.ElementsAs(ctx, &ciphers, false)
		tlsCtx.CipherSuites = ciphers
	}

	tlsCtx.InboundSettings = &secretsmanagement.TLSInboundSettings{
		EnableClientCertValidation: data.EnableClientCertValidation.ValueBool(),
	}
	tlsCtx.OutboundSettings = &secretsmanagement.TLSOutboundSettings{
		SkipServerCertValidation: data.SkipServerCertValidation.ValueBool(),
	}

	return tlsCtx
}

func (r *TLSContextResource) flattenTLSContext(ctx context.Context, tls *secretsmanagement.TLSContextResponse, data *TLSContextResourceModel, orgID, envID, sgID string, _ *diag.Diagnostics) {
	data.ID = types.StringValue(tls.Meta.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.SecretGroupID = types.StringValue(sgID)
	data.Name = types.StringValue(tls.Name)
	target := tls.Target
	if target == "FlexGateway" {
		target = "OmniGateway"
	}
	data.Target = types.StringValue(target)

	// Extract keystore ID from path "keystores/{id}"
	if tls.Keystore != nil && tls.Keystore.Path != "" {
		data.KeystoreID = types.StringValue(extractIDFromPath(tls.Keystore.Path))
	}

	// Extract truststore ID from path "truststores/{id}"
	if tls.Truststore != nil && tls.Truststore.Path != "" {
		data.TruststoreID = types.StringValue(extractIDFromPath(tls.Truststore.Path))
	}

	if tls.MinTLSVersion != "" {
		data.MinTLSVersion = types.StringValue(tls.MinTLSVersion)
	}
	if tls.MaxTLSVersion != "" {
		data.MaxTLSVersion = types.StringValue(tls.MaxTLSVersion)
	}

	if len(tls.AlpnProtocols) > 0 {
		listVal, _ := types.ListValueFrom(ctx, types.StringType, tls.AlpnProtocols)
		data.AlpnProtocols = listVal
	}

	if len(tls.CipherSuites) > 0 {
		listVal, _ := types.ListValueFrom(ctx, types.StringType, tls.CipherSuites)
		data.CipherSuites = listVal
	}

	if tls.InboundSettings != nil {
		data.EnableClientCertValidation = types.BoolValue(tls.InboundSettings.EnableClientCertValidation)
	}
	if tls.OutboundSettings != nil {
		data.SkipServerCertValidation = types.BoolValue(tls.OutboundSettings.SkipServerCertValidation)
	}

	if tls.ExpirationDate != "" {
		data.ExpirationDate = types.StringValue(tls.ExpirationDate)
	} else {
		data.ExpirationDate = types.StringValue("")
	}
}

// extractIDFromPath extracts the UUID from paths like "keystores/{id}" or "truststores/{id}".
func extractIDFromPath(pathStr string) string {
	parts := strings.Split(pathStr, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return pathStr
}
