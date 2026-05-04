package cloudhub2

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &TLSContextResource{}
	_ resource.ResourceWithConfigure   = &TLSContextResource{}
	_ resource.ResourceWithImportState = &TLSContextResource{}
)

// TLSContextResource is the resource implementation.
type TLSContextResource struct {
	client *cloudhub2.TLSContextClient
}

// TLSContextResourceModel describes the resource data model.
type TLSContextResourceModel struct {
	ID             types.String `tfsdk:"id"`
	PrivateSpaceID types.String `tfsdk:"private_space_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	KeyStoreType   types.String `tfsdk:"keystore_type"`
	// PEM fields
	Certificate         types.String `tfsdk:"certificate"`
	Key                 types.String `tfsdk:"key"`
	KeyFileName         types.String `tfsdk:"key_filename"`
	CertificateFileName types.String `tfsdk:"certificate_filename"`
	// JKS fields
	KeystoreBase64   types.String `tfsdk:"keystore_base64"`
	StorePassphrase  types.String `tfsdk:"store_passphrase"`
	Alias            types.String `tfsdk:"alias"`
	KeystoreFileName types.String `tfsdk:"keystore_filename"`
	// Common fields
	KeyPassphrase types.String `tfsdk:"key_passphrase"`
	// Ciphers
	Ciphers types.Object `tfsdk:"ciphers"`
	// Computed fields
	Type       types.String `tfsdk:"type"`
	TrustStore types.Object `tfsdk:"trust_store"`
	KeyStore   types.Object `tfsdk:"key_store"`
}

func NewTLSContextResource() resource.Resource {
	return &TLSContextResource{}
}

// Metadata returns the resource type name.
func (r *TLSContextResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tls_context"
}

// Schema defines the schema for the resource.
func (r *TLSContextResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudHub 2.0 TLS Context with support for both PEM and JKS keystores.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the TLS context.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space this TLS context belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the TLS context.",
				Required:    true,
			},
			"keystore_type": schema.StringAttribute{
				Description: "The type of keystore: 'PEM' or 'JKS'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// PEM-specific fields
			"certificate": schema.StringAttribute{
				Description: "PEM certificate content (required for PEM keystore).",
				Optional:    true,
				Sensitive:   true,
			},
			"key": schema.StringAttribute{
				Description: "PEM private key content (required for PEM keystore).",
				Optional:    true,
				Sensitive:   true,
			},
			"key_filename": schema.StringAttribute{
				Description: "Filename for the private key (PEM keystore).",
				Optional:    true,
			},
			"certificate_filename": schema.StringAttribute{
				Description: "Filename for the certificate (PEM keystore).",
				Optional:    true,
			},
			// JKS-specific fields
			"keystore_base64": schema.StringAttribute{
				Description: "Base64 encoded JKS keystore content (required for JKS keystore).",
				Optional:    true,
				Sensitive:   true,
			},
			"store_passphrase": schema.StringAttribute{
				Description: "Store passphrase for JKS keystore (required for JKS keystore).",
				Optional:    true,
				Sensitive:   true,
			},
			"alias": schema.StringAttribute{
				Description: "Alias for JKS keystore (required for JKS keystore).",
				Optional:    true,
			},
			"keystore_filename": schema.StringAttribute{
				Description: "Filename for the JKS keystore (required for JKS keystore).",
				Optional:    true,
			},
			// Common fields
			"key_passphrase": schema.StringAttribute{
				Description: "Passphrase for the private key.",
				Optional:    true,
				Sensitive:   true,
			},
			// Cipher configuration
			"ciphers": schema.SingleNestedAttribute{
				Description: "Cipher configuration for the TLS context.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"aes128_gcm_sha256": schema.BoolAttribute{
						Description: "Enable AES128-GCM-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"aes128_sha256": schema.BoolAttribute{
						Description: "Enable AES128-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"aes256_gcm_sha384": schema.BoolAttribute{
						Description: "Enable AES256-GCM-SHA384 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"aes256_sha256": schema.BoolAttribute{
						Description: "Enable AES256-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"dhe_rsa_aes128_sha256": schema.BoolAttribute{
						Description: "Enable DHE-RSA-AES128-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"dhe_rsa_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "Enable DHE-RSA-AES256-GCM-SHA384 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"dhe_rsa_aes256_sha256": schema.BoolAttribute{
						Description: "Enable DHE-RSA-AES256-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"ecdhe_ecdsa_aes128_gcm_sha256": schema.BoolAttribute{
						Description: "Enable ECDHE-ECDSA-AES128-GCM-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"ecdhe_ecdsa_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "Enable ECDHE-ECDSA-AES256-GCM-SHA384 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"ecdhe_rsa_aes128_gcm_sha256": schema.BoolAttribute{
						Description: "Enable ECDHE-RSA-AES128-GCM-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"ecdhe_rsa_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "Enable ECDHE-RSA-AES256-GCM-SHA384 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"ecdhe_ecdsa_chacha20_poly1305": schema.BoolAttribute{
						Description: "Enable ECDHE-ECDSA-CHACHA20-POLY1305 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"ecdhe_rsa_chacha20_poly1305": schema.BoolAttribute{
						Description: "Enable ECDHE-RSA-CHACHA20-POLY1305 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"dhe_rsa_chacha20_poly1305": schema.BoolAttribute{
						Description: "Enable DHE-RSA-CHACHA20-POLY1305 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"tls_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "Enable TLS-AES256-GCM-SHA384 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"tls_chacha20_poly1305_sha256": schema.BoolAttribute{
						Description: "Enable TLS-CHACHA20-POLY1305-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"tls_aes128_gcm_sha256": schema.BoolAttribute{
						Description: "Enable TLS-AES128-GCM-SHA256 cipher.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
			// Computed fields
			"type": schema.StringAttribute{
				Description: "The type of TLS context.",
				Computed:    true,
			},
			"trust_store": schema.SingleNestedAttribute{
				Description: "Trust store information.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"filename": schema.StringAttribute{
						Description: "Trust store filename.",
						Computed:    true,
					},
					"expiration_date": schema.StringAttribute{
						Description: "Trust store expiration date.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Trust store type.",
						Computed:    true,
					},
				},
			},
			"key_store": schema.SingleNestedAttribute{
				Description: "Key store information.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"filename": schema.StringAttribute{
						Description: "Key store filename.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Key store type.",
						Computed:    true,
					},
					"cn": schema.StringAttribute{
						Description: "Common name from the certificate.",
						Computed:    true,
					},
					"san": schema.ListAttribute{
						Description: "Subject alternative names.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"expiration_date": schema.StringAttribute{
						Description: "Key store expiration date.",
						Computed:    true,
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *TLSContextResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	tlsClient, err := cloudhub2.NewTLSContextClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create TLS Context Client",
			"An unexpected error occurred when creating the TLS Context client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = tlsClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *TLSContextResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TLSContextResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Validate keystore type and required fields
	keystoreType := data.KeyStoreType.ValueString()
	if keystoreType != "PEM" && keystoreType != "JKS" {
		resp.Diagnostics.AddError(
			"Invalid keystore type",
			"Keystore type must be either 'PEM' or 'JKS'",
		)
		return
	}

	// Build cipher configuration
	ciphersAttrs := data.Ciphers.Attributes()
	ciphers := cloudhub2.CiphersConfig{
		AES128GcmSha256:            ciphersAttrs["aes128_gcm_sha256"].(types.Bool).ValueBool(),
		AES128Sha256:               ciphersAttrs["aes128_sha256"].(types.Bool).ValueBool(),
		AES256GcmSha384:            ciphersAttrs["aes256_gcm_sha384"].(types.Bool).ValueBool(),
		AES256Sha256:               ciphersAttrs["aes256_sha256"].(types.Bool).ValueBool(),
		DHERsaAES128Sha256:         ciphersAttrs["dhe_rsa_aes128_sha256"].(types.Bool).ValueBool(),
		DHERsaAES256GcmSha384:      ciphersAttrs["dhe_rsa_aes256_gcm_sha384"].(types.Bool).ValueBool(),
		DHERsaAES256Sha256:         ciphersAttrs["dhe_rsa_aes256_sha256"].(types.Bool).ValueBool(),
		ECDHEECDSAAES128GcmSha256:  ciphersAttrs["ecdhe_ecdsa_aes128_gcm_sha256"].(types.Bool).ValueBool(),
		ECDHEECDSAAES256GcmSha384:  ciphersAttrs["ecdhe_ecdsa_aes256_gcm_sha384"].(types.Bool).ValueBool(),
		ECDHERsaAES128GcmSha256:    ciphersAttrs["ecdhe_rsa_aes128_gcm_sha256"].(types.Bool).ValueBool(),
		ECDHERsaAES256GcmSha384:    ciphersAttrs["ecdhe_rsa_aes256_gcm_sha384"].(types.Bool).ValueBool(),
		ECDHEECDSAChacha20Poly1305: ciphersAttrs["ecdhe_ecdsa_chacha20_poly1305"].(types.Bool).ValueBool(),
		ECDHERsaChacha20Poly1305:   ciphersAttrs["ecdhe_rsa_chacha20_poly1305"].(types.Bool).ValueBool(),
		DHERsaChacha20Poly1305:     ciphersAttrs["dhe_rsa_chacha20_poly1305"].(types.Bool).ValueBool(),
		TLSAES256GcmSha384:         ciphersAttrs["tls_aes256_gcm_sha384"].(types.Bool).ValueBool(),
		TLSChacha20Poly1305Sha256:  ciphersAttrs["tls_chacha20_poly1305_sha256"].(types.Bool).ValueBool(),
		TLSAES128GcmSha256:         ciphersAttrs["tls_aes128_gcm_sha256"].(types.Bool).ValueBool(),
	}

	// Build keystore request based on type
	keystoreRequest := cloudhub2.KeyStoreRequest{
		Source: keystoreType,
	}

	switch keystoreType {
	case "PEM":
		// Validate PEM required fields
		if data.Certificate.IsNull() || data.Key.IsNull() {
			resp.Diagnostics.AddError(
				"Missing required PEM fields",
				"Certificate and key are required for PEM keystore type",
			)
			return
		}
		cert := data.Certificate.ValueString()
		key := data.Key.ValueString()
		keystoreRequest.Certificate = &cert
		keystoreRequest.Key = &key

		if !data.KeyFileName.IsNull() {
			keyFilename := data.KeyFileName.ValueString()
			keystoreRequest.KeyFileName = &keyFilename
		}
		if !data.CertificateFileName.IsNull() {
			certFilename := data.CertificateFileName.ValueString()
			keystoreRequest.CertificateFileName = &certFilename
		}
	case "JKS":
		// Validate JKS required fields
		if data.KeystoreBase64.IsNull() || data.StorePassphrase.IsNull() || data.Alias.IsNull() {
			resp.Diagnostics.AddError(
				"Missing required JKS fields",
				"KeystoreBase64, storePassphrase, and alias are required for JKS keystore type",
			)
			return
		}
		keystoreBase64 := data.KeystoreBase64.ValueString()
		storePassphrase := data.StorePassphrase.ValueString()
		alias := data.Alias.ValueString()
		keystoreRequest.KeystoreBase64 = &keystoreBase64
		keystoreRequest.StorePassphrase = &storePassphrase
		keystoreRequest.Alias = &alias

		if !data.KeystoreFileName.IsNull() {
			keystoreFilename := data.KeystoreFileName.ValueString()
			keystoreRequest.KeystoreFileName = &keystoreFilename
		}
	}

	// Set key passphrase if provided
	if !data.KeyPassphrase.IsNull() {
		keyPassphrase := data.KeyPassphrase.ValueString()
		keystoreRequest.KeyPassphrase = &keyPassphrase
	}

	// Create the TLS context
	createRequest := &cloudhub2.CreateTLSContextRequest{
		Name: data.Name.ValueString(),
		TLSConfig: cloudhub2.TLSConfigRequest{
			KeyStore: keystoreRequest,
		},
		Ciphers: ciphers,
	}

	var err error
	err = r.client.CreateTLSContext(ctx, orgID, data.PrivateSpaceID.ValueString(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating TLS context",
			"Could not create TLS context: "+err.Error(),
		)
		return
	}

	// API returns 201 with no body, so we need to find the created TLS context by name
	tlsContexts, err := r.client.ListTLSContexts(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing TLS contexts after creation",
			"TLS context was created but could not retrieve its details: "+err.Error(),
		)
		return
	}

	// Find the TLS context with the matching name
	var tlsContext *cloudhub2.TLSContext
	for _, ctx := range tlsContexts {
		if ctx.Name == data.Name.ValueString() {
			tlsContext = &ctx
			break
		}
	}

	if tlsContext == nil {
		resp.Diagnostics.AddError(
			"Error finding created TLS context",
			"TLS context was created but could not find it in the list of contexts",
		)
		return
	}

	// Map response to state
	data.ID = types.StringValue(tlsContext.ID)
	data.OrganizationID = types.StringValue(orgID) // Set the actual org ID used
	data.Name = types.StringValue(tlsContext.Name)
	data.Type = types.StringValue(tlsContext.Type)

	// Map ciphers to state
	ciphersStateObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"aes128_gcm_sha256":             types.BoolType,
			"aes128_sha256":                 types.BoolType,
			"aes256_gcm_sha384":             types.BoolType,
			"aes256_sha256":                 types.BoolType,
			"dhe_rsa_aes128_sha256":         types.BoolType,
			"dhe_rsa_aes256_gcm_sha384":     types.BoolType,
			"dhe_rsa_aes256_sha256":         types.BoolType,
			"ecdhe_ecdsa_aes128_gcm_sha256": types.BoolType,
			"ecdhe_ecdsa_aes256_gcm_sha384": types.BoolType,
			"ecdhe_rsa_aes128_gcm_sha256":   types.BoolType,
			"ecdhe_rsa_aes256_gcm_sha384":   types.BoolType,
			"ecdhe_ecdsa_chacha20_poly1305": types.BoolType,
			"ecdhe_rsa_chacha20_poly1305":   types.BoolType,
			"dhe_rsa_chacha20_poly1305":     types.BoolType,
			"tls_aes256_gcm_sha384":         types.BoolType,
			"tls_chacha20_poly1305_sha256":  types.BoolType,
			"tls_aes128_gcm_sha256":         types.BoolType,
		},
		map[string]attr.Value{
			"aes128_gcm_sha256":             types.BoolValue(tlsContext.Ciphers.AES128GcmSha256),
			"aes128_sha256":                 types.BoolValue(tlsContext.Ciphers.AES128Sha256),
			"aes256_gcm_sha384":             types.BoolValue(tlsContext.Ciphers.AES256GcmSha384),
			"aes256_sha256":                 types.BoolValue(tlsContext.Ciphers.AES256Sha256),
			"dhe_rsa_aes128_sha256":         types.BoolValue(tlsContext.Ciphers.DHERsaAES128Sha256),
			"dhe_rsa_aes256_gcm_sha384":     types.BoolValue(tlsContext.Ciphers.DHERsaAES256GcmSha384),
			"dhe_rsa_aes256_sha256":         types.BoolValue(tlsContext.Ciphers.DHERsaAES256Sha256),
			"ecdhe_ecdsa_aes128_gcm_sha256": types.BoolValue(tlsContext.Ciphers.ECDHEECDSAAES128GcmSha256),
			"ecdhe_ecdsa_aes256_gcm_sha384": types.BoolValue(tlsContext.Ciphers.ECDHEECDSAAES256GcmSha384),
			"ecdhe_rsa_aes128_gcm_sha256":   types.BoolValue(tlsContext.Ciphers.ECDHERsaAES128GcmSha256),
			"ecdhe_rsa_aes256_gcm_sha384":   types.BoolValue(tlsContext.Ciphers.ECDHERsaAES256GcmSha384),
			"ecdhe_ecdsa_chacha20_poly1305": types.BoolValue(tlsContext.Ciphers.ECDHEECDSAChacha20Poly1305),
			"ecdhe_rsa_chacha20_poly1305":   types.BoolValue(tlsContext.Ciphers.ECDHERsaChacha20Poly1305),
			"dhe_rsa_chacha20_poly1305":     types.BoolValue(tlsContext.Ciphers.DHERsaChacha20Poly1305),
			"tls_aes256_gcm_sha384":         types.BoolValue(tlsContext.Ciphers.TLSAES256GcmSha384),
			"tls_chacha20_poly1305_sha256":  types.BoolValue(tlsContext.Ciphers.TLSChacha20Poly1305Sha256),
			"tls_aes128_gcm_sha256":         types.BoolValue(tlsContext.Ciphers.TLSAES128GcmSha256),
		},
	)
	data.Ciphers = ciphersStateObj

	// Map trust store if present, otherwise set to null
	if tlsContext.TrustStore != nil {
		trustStoreObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"filename":        types.StringType,
				"expiration_date": types.StringType,
				"type":            types.StringType,
			},
			map[string]attr.Value{
				"filename":        types.StringValue(tlsContext.TrustStore.FileName),
				"expiration_date": types.StringValue(tlsContext.TrustStore.ExpirationDate),
				"type":            types.StringValue(tlsContext.TrustStore.Type),
			},
		)
		data.TrustStore = trustStoreObj
	} else {
		// Set trust store to null since API doesn't return trustStore field
		data.TrustStore = types.ObjectNull(map[string]attr.Type{
			"filename":        types.StringType,
			"expiration_date": types.StringType,
			"type":            types.StringType,
		})
	}

	// Map key store if present
	if tlsContext.KeyStore != nil {
		var sanElements []attr.Value
		for _, san := range tlsContext.KeyStore.SAN {
			sanElements = append(sanElements, types.StringValue(san))
		}
		sanList, _ := types.ListValue(types.StringType, sanElements)

		keyStoreObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"filename":        types.StringType,
				"type":            types.StringType,
				"cn":              types.StringType,
				"san":             types.ListType{ElemType: types.StringType},
				"expiration_date": types.StringType,
			},
			map[string]attr.Value{
				"filename":        types.StringValue(tlsContext.KeyStore.FileName),
				"type":            types.StringValue(tlsContext.KeyStore.Type),
				"cn":              types.StringValue(tlsContext.KeyStore.CN),
				"san":             sanList,
				"expiration_date": types.StringValue(tlsContext.KeyStore.ExpirationDate),
			},
		)
		data.KeyStore = keyStoreObj
	}

	tflog.Trace(ctx, "created TLS context")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TLSContextResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TLSContextResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the TLS context from the API
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	tlsContext, err := r.client.GetTLSContext(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading TLS context",
			"Could not read TLS context ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update computed fields only
	data.ID = types.StringValue(tlsContext.ID)
	data.Name = types.StringValue(tlsContext.Name)
	data.Type = types.StringValue(tlsContext.Type)

	// Update ciphers from response
	ciphersStateObj, _ := types.ObjectValue(
		map[string]attr.Type{
			"aes128_gcm_sha256":             types.BoolType,
			"aes128_sha256":                 types.BoolType,
			"aes256_gcm_sha384":             types.BoolType,
			"aes256_sha256":                 types.BoolType,
			"dhe_rsa_aes128_sha256":         types.BoolType,
			"dhe_rsa_aes256_gcm_sha384":     types.BoolType,
			"dhe_rsa_aes256_sha256":         types.BoolType,
			"ecdhe_ecdsa_aes128_gcm_sha256": types.BoolType,
			"ecdhe_ecdsa_aes256_gcm_sha384": types.BoolType,
			"ecdhe_rsa_aes128_gcm_sha256":   types.BoolType,
			"ecdhe_rsa_aes256_gcm_sha384":   types.BoolType,
			"ecdhe_ecdsa_chacha20_poly1305": types.BoolType,
			"ecdhe_rsa_chacha20_poly1305":   types.BoolType,
			"dhe_rsa_chacha20_poly1305":     types.BoolType,
			"tls_aes256_gcm_sha384":         types.BoolType,
			"tls_chacha20_poly1305_sha256":  types.BoolType,
			"tls_aes128_gcm_sha256":         types.BoolType,
		},
		map[string]attr.Value{
			"aes128_gcm_sha256":             types.BoolValue(tlsContext.Ciphers.AES128GcmSha256),
			"aes128_sha256":                 types.BoolValue(tlsContext.Ciphers.AES128Sha256),
			"aes256_gcm_sha384":             types.BoolValue(tlsContext.Ciphers.AES256GcmSha384),
			"aes256_sha256":                 types.BoolValue(tlsContext.Ciphers.AES256Sha256),
			"dhe_rsa_aes128_sha256":         types.BoolValue(tlsContext.Ciphers.DHERsaAES128Sha256),
			"dhe_rsa_aes256_gcm_sha384":     types.BoolValue(tlsContext.Ciphers.DHERsaAES256GcmSha384),
			"dhe_rsa_aes256_sha256":         types.BoolValue(tlsContext.Ciphers.DHERsaAES256Sha256),
			"ecdhe_ecdsa_aes128_gcm_sha256": types.BoolValue(tlsContext.Ciphers.ECDHEECDSAAES128GcmSha256),
			"ecdhe_ecdsa_aes256_gcm_sha384": types.BoolValue(tlsContext.Ciphers.ECDHEECDSAAES256GcmSha384),
			"ecdhe_rsa_aes128_gcm_sha256":   types.BoolValue(tlsContext.Ciphers.ECDHERsaAES128GcmSha256),
			"ecdhe_rsa_aes256_gcm_sha384":   types.BoolValue(tlsContext.Ciphers.ECDHERsaAES256GcmSha384),
			"ecdhe_ecdsa_chacha20_poly1305": types.BoolValue(tlsContext.Ciphers.ECDHEECDSAChacha20Poly1305),
			"ecdhe_rsa_chacha20_poly1305":   types.BoolValue(tlsContext.Ciphers.ECDHERsaChacha20Poly1305),
			"dhe_rsa_chacha20_poly1305":     types.BoolValue(tlsContext.Ciphers.DHERsaChacha20Poly1305),
			"tls_aes256_gcm_sha384":         types.BoolValue(tlsContext.Ciphers.TLSAES256GcmSha384),
			"tls_chacha20_poly1305_sha256":  types.BoolValue(tlsContext.Ciphers.TLSChacha20Poly1305Sha256),
			"tls_aes128_gcm_sha256":         types.BoolValue(tlsContext.Ciphers.TLSAES128GcmSha256),
		},
	)
	data.Ciphers = ciphersStateObj

	// Update trust store if present, otherwise set to null
	if tlsContext.TrustStore != nil {
		trustStoreObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"filename":        types.StringType,
				"expiration_date": types.StringType,
				"type":            types.StringType,
			},
			map[string]attr.Value{
				"filename":        types.StringValue(tlsContext.TrustStore.FileName),
				"expiration_date": types.StringValue(tlsContext.TrustStore.ExpirationDate),
				"type":            types.StringValue(tlsContext.TrustStore.Type),
			},
		)
		data.TrustStore = trustStoreObj
	} else {
		// Set trust store to null since API doesn't return trustStore field
		data.TrustStore = types.ObjectNull(map[string]attr.Type{
			"filename":        types.StringType,
			"expiration_date": types.StringType,
			"type":            types.StringType,
		})
	}

	// Update key store if present
	if tlsContext.KeyStore != nil {
		var sanElements []attr.Value
		for _, san := range tlsContext.KeyStore.SAN {
			sanElements = append(sanElements, types.StringValue(san))
		}
		sanList, _ := types.ListValue(types.StringType, sanElements)

		keyStoreObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"filename":        types.StringType,
				"type":            types.StringType,
				"cn":              types.StringType,
				"san":             types.ListType{ElemType: types.StringType},
				"expiration_date": types.StringType,
			},
			map[string]attr.Value{
				"filename":        types.StringValue(tlsContext.KeyStore.FileName),
				"type":            types.StringValue(tlsContext.KeyStore.Type),
				"cn":              types.StringValue(tlsContext.KeyStore.CN),
				"san":             sanList,
				"expiration_date": types.StringValue(tlsContext.KeyStore.ExpirationDate),
			},
		)
		data.KeyStore = keyStoreObj
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *TLSContextResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TLSContextResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateRequest := &cloudhub2.UpdateTLSContextRequest{}
	hasChanges := false

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		updateRequest.Name = &name
		hasChanges = true
	}

	// Note: Keystore and cipher updates would require replacement in most cases
	// For now, we'll only support name updates
	if !hasChanges {
		// No changes to apply
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// Update the TLS context
	// Determine organization ID from plan or default to client's org
	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	tlsContext, err := r.client.UpdateTLSContext(ctx, orgID, plan.PrivateSpaceID.ValueString(), plan.ID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating TLS context",
			"Could not update TLS context: "+err.Error(),
		)
		return
	}

	// Update state with response
	plan.ID = types.StringValue(tlsContext.ID)
	plan.Name = types.StringValue(tlsContext.Name)
	plan.Type = types.StringValue(tlsContext.Type)

	tflog.Trace(ctx, "updated TLS context")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TLSContextResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TLSContextResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the TLS context
	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	err := r.client.DeleteTLSContext(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting TLS context",
			"Could not delete TLS context: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted TLS context")
}

// ImportState imports the resource into Terraform state.
func (r *TLSContextResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: private_space_id:tls_context_id
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: private_space_id:tls_context_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
