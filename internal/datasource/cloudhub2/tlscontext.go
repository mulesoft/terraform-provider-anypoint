package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &TLSContextDataSource{}
	_ datasource.DataSourceWithConfigure = &TLSContextDataSource{}
)

// TLSContextDataSource is the data source implementation.
type TLSContextDataSource struct {
	client *cloudhub2.TLSContextClient
}

// TLSContextDataSourceModel describes the data source data model.
type TLSContextDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	PrivateSpaceID types.String `tfsdk:"private_space_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Type           types.String `tfsdk:"type"`
	Ciphers        types.Object `tfsdk:"ciphers"`
	TrustStore     types.Object `tfsdk:"trust_store"`
	KeyStore       types.Object `tfsdk:"key_store"`
}

func NewTLSContextDataSource() datasource.DataSource {
	return &TLSContextDataSource{}
}

// Metadata returns the data source type name.
func (d *TLSContextDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tls_context"
}

// Schema defines the schema for the data source.
func (d *TLSContextDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a CloudHub 2.0 TLS context.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the TLS context.",
				Required:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The private space ID where the TLS context is located.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the TLS context.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the TLS context.",
				Computed:    true,
			},
			"ciphers": schema.SingleNestedAttribute{
				Description: "Cipher configuration for the TLS context.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"aes128_gcm_sha256": schema.BoolAttribute{
						Description: "AES128-GCM-SHA256 cipher status.",
						Computed:    true,
					},
					"aes128_sha256": schema.BoolAttribute{
						Description: "AES128-SHA256 cipher status.",
						Computed:    true,
					},
					"aes256_gcm_sha384": schema.BoolAttribute{
						Description: "AES256-GCM-SHA384 cipher status.",
						Computed:    true,
					},
					"aes256_sha256": schema.BoolAttribute{
						Description: "AES256-SHA256 cipher status.",
						Computed:    true,
					},
					"dhe_rsa_aes128_sha256": schema.BoolAttribute{
						Description: "DHE-RSA-AES128-SHA256 cipher status.",
						Computed:    true,
					},
					"dhe_rsa_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "DHE-RSA-AES256-GCM-SHA384 cipher status.",
						Computed:    true,
					},
					"dhe_rsa_aes256_sha256": schema.BoolAttribute{
						Description: "DHE-RSA-AES256-SHA256 cipher status.",
						Computed:    true,
					},
					"ecdhe_ecdsa_aes128_gcm_sha256": schema.BoolAttribute{
						Description: "ECDHE-ECDSA-AES128-GCM-SHA256 cipher status.",
						Computed:    true,
					},
					"ecdhe_ecdsa_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "ECDHE-ECDSA-AES256-GCM-SHA384 cipher status.",
						Computed:    true,
					},
					"ecdhe_rsa_aes128_gcm_sha256": schema.BoolAttribute{
						Description: "ECDHE-RSA-AES128-GCM-SHA256 cipher status.",
						Computed:    true,
					},
					"ecdhe_rsa_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "ECDHE-RSA-AES256-GCM-SHA384 cipher status.",
						Computed:    true,
					},
					"ecdhe_ecdsa_chacha20_poly1305": schema.BoolAttribute{
						Description: "ECDHE-ECDSA-CHACHA20-POLY1305 cipher status.",
						Computed:    true,
					},
					"ecdhe_rsa_chacha20_poly1305": schema.BoolAttribute{
						Description: "ECDHE-RSA-CHACHA20-POLY1305 cipher status.",
						Computed:    true,
					},
					"dhe_rsa_chacha20_poly1305": schema.BoolAttribute{
						Description: "DHE-RSA-CHACHA20-POLY1305 cipher status.",
						Computed:    true,
					},
					"tls_aes256_gcm_sha384": schema.BoolAttribute{
						Description: "TLS-AES256-GCM-SHA384 cipher status.",
						Computed:    true,
					},
					"tls_chacha20_poly1305_sha256": schema.BoolAttribute{
						Description: "TLS-CHACHA20-POLY1305-SHA256 cipher status.",
						Computed:    true,
					},
					"tls_aes128_gcm_sha256": schema.BoolAttribute{
						Description: "TLS-AES128-GCM-SHA256 cipher status.",
						Computed:    true,
					},
				},
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

// Configure adds the provider configured client to the data source.
func (d *TLSContextDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Extract the client configuration
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Create the TLS context client
	tlsContextClient, err := cloudhub2.NewTLSContextClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create CloudHub 2.0 TLS Context API Client",
			"An unexpected error occurred when creating the CloudHub 2.0 TLS Context API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"CloudHub 2.0 Client Error: "+err.Error(),
		)
		return
	}

	d.client = tlsContextClient
}

// Read refreshes the Terraform state with the latest data.
func (d *TLSContextDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TLSContextDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// Get the TLS context from the API
	tlsContext, err := d.client.GetTLSContext(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading TLS context",
			"Could not read TLS context ID "+data.ID.ValueString()+" in private space "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(tlsContext.ID)
	data.Name = types.StringValue(tlsContext.Name)
	data.Type = types.StringValue(tlsContext.Type)

	// Map ciphers
	ciphersObj, _ := types.ObjectValue(
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
	data.Ciphers = ciphersObj

	// Map trust store if present
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
