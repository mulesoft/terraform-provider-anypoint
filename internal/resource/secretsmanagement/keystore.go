package secretsmanagement

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &KeystoreResource{}
	_ resource.ResourceWithConfigure   = &KeystoreResource{}
	_ resource.ResourceWithImportState = &KeystoreResource{}
)

type KeystoreResource struct {
	client *secretsmanagement.KeystoreClient
}

type KeystoreResourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	SecretGroupID    types.String `tfsdk:"secret_group_id"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	CertificateB64   types.String `tfsdk:"certificate_base64"`
	KeyB64           types.String `tfsdk:"key_base64"`
	KeystoreFileB64  types.String `tfsdk:"keystore_file_base64"`
	Passphrase       types.String `tfsdk:"passphrase"`
	Alias            types.String `tfsdk:"alias"`
	CaPathB64        types.String `tfsdk:"ca_path_base64"`
	ExpirationDate   types.String `tfsdk:"expiration_date"`
	Algorithm        types.String `tfsdk:"algorithm"`
}

func NewKeystoreResource() resource.Resource {
	return &KeystoreResource{}
}

func (r *KeystoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_keystore"
}

func (r *KeystoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a keystore within a secret group in Anypoint Secrets Manager. " +
			"Supports PEM, JKS, PKCS12, and JCEKS formats. " +
			"Use filebase64() to read binary files (JKS/PKCS12) or file() for PEM text files.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the keystore.",
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
				Description: "Secret group ID that this keystore belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the keystore.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Keystore format: PEM, JKS, PKCS12, or JCEKS.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("PEM"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("PEM", "JKS", "PKCS12", "JCEKS"),
				},
			},

			// --- PEM-specific fields ---
			"certificate_base64": schema.StringAttribute{
				Description: "Base64-encoded certificate content. For PEM files use base64encode(file(\"cert.pem\")), " +
					"or for binary formats use filebase64(\"cert.der\").",
				Optional:  true,
				Sensitive: true,
			},
			"key_base64": schema.StringAttribute{
				Description: "Base64-encoded private key content. For PEM keys use base64encode(file(\"key.pem\")), " +
					"or for binary keys use filebase64(\"key.der\"). Required for PEM type.",
				Optional:  true,
				Sensitive: true,
			},

			// --- JKS/PKCS12/JCEKS-specific fields ---
			"keystore_file_base64": schema.StringAttribute{
				Description: "Base64-encoded keystore file content. Use filebase64(\"keystore.jks\") or filebase64(\"keystore.p12\"). " +
					"Required for JKS, PKCS12, and JCEKS types.",
				Optional:  true,
				Sensitive: true,
			},
			"passphrase": schema.StringAttribute{
				Description: "Passphrase for the keystore or encrypted PEM key.",
				Optional:  true,
				Sensitive: true,
			},
			"alias": schema.StringAttribute{
				Description: "Alias of the entry within the keystore. Used for JKS, PKCS12, and JCEKS types.",
				Optional: true,
			},

			// --- Optional CA chain ---
			"ca_path_base64": schema.StringAttribute{
				Description: "Base64-encoded CA certificate chain (truststore). Optional for all types.",
				Optional:  true,
				Sensitive: true,
			},

			// --- Read-only fields from API ---
			"expiration_date": schema.StringAttribute{
				Description: "Expiration date of the certificate in the keystore.",
				Computed:    true,
			},
			"algorithm": schema.StringAttribute{
				Description: "Signature algorithm of the certificate.",
				Computed:    true,
			},
		},
	}
}

func (r *KeystoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T.", req.ProviderData),
		)
		return
	}

	ksClient, err := secretsmanagement.NewKeystoreClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Keystore Client",
			"An unexpected error occurred: "+err.Error(),
		)
		return
	}
	r.client = ksClient
}

func (r *KeystoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KeystoreResourceModel
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

	createReq, err := r.expandRequest(&data)
	if err != nil {
		resp.Diagnostics.AddError("Error building keystore request", err.Error())
		return
	}

	ks, err := r.client.CreateKeystore(ctx, orgID, envID, sgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating keystore", "Could not create keystore: "+err.Error())
		return
	}

	r.flattenKeystore(ks, &data, orgID, envID, sgID)
	tflog.Trace(ctx, "created keystore", map[string]interface{}{"id": ks.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeystoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KeystoreResourceModel
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

	ks, err := r.client.GetKeystore(ctx, orgID, envID, sgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading keystore", "Could not read keystore: "+err.Error())
		return
	}

	// Preserve sensitive fields that the API does not return
	savedCert := data.CertificateB64
	savedKey := data.KeyB64
	savedKsFile := data.KeystoreFileB64
	savedPassphrase := data.Passphrase
	savedCaPath := data.CaPathB64

	r.flattenKeystore(ks, &data, orgID, envID, sgID)

	data.CertificateB64 = savedCert
	data.KeyB64 = savedKey
	data.KeystoreFileB64 = savedKsFile
	data.Passphrase = savedPassphrase
	data.CaPathB64 = savedCaPath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeystoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state KeystoreResourceModel
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

	updateReq, err := r.expandRequest(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building keystore update request", err.Error())
		return
	}

	ks, err := r.client.UpdateKeystore(ctx, orgID, envID, sgID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating keystore", "Could not update keystore: "+err.Error())
		return
	}

	r.flattenKeystore(ks, &plan, orgID, envID, sgID)

	// Preserve sensitive inputs that the API doesn't echo back
	plan.CertificateB64 = plan.CertificateB64
	plan.KeyB64 = plan.KeyB64
	plan.KeystoreFileB64 = plan.KeystoreFileB64
	plan.Passphrase = plan.Passphrase
	plan.CaPathB64 = plan.CaPathB64

	tflog.Trace(ctx, "updated keystore", map[string]interface{}{"id": ks.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KeystoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KeystoreResourceModel
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

	if err := r.client.DeleteKeystore(ctx, orgID, envID, sgID, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting keystore", "Could not delete keystore: "+err.Error())
		return
	}
	tflog.Trace(ctx, "deleted keystore", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *KeystoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Format: organization_id/environment_id/secret_group_id/keystore_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/secret_group_id/keystore_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// --- Helpers ---

func (r *KeystoreResource) expandRequest(data *KeystoreResourceModel) (*secretsmanagement.CreateKeystoreRequest, error) {
	ksType := data.Type.ValueString()

	createReq := &secretsmanagement.CreateKeystoreRequest{
		Name:       data.Name.ValueString(),
		Type:       ksType,
		Passphrase: data.Passphrase.ValueString(),
		Alias:      data.Alias.ValueString(),
	}

	switch ksType {
	case "PEM":
		if !data.CertificateB64.IsNull() && !data.CertificateB64.IsUnknown() && data.CertificateB64.ValueString() != "" {
			certBytes, err := base64.StdEncoding.DecodeString(data.CertificateB64.ValueString())
			if err != nil {
				return nil, fmt.Errorf("failed to decode certificate_base64: %w", err)
			}
			createReq.Certificate = certBytes
		}
		if !data.KeyB64.IsNull() && !data.KeyB64.IsUnknown() && data.KeyB64.ValueString() != "" {
			keyBytes, err := base64.StdEncoding.DecodeString(data.KeyB64.ValueString())
			if err != nil {
				return nil, fmt.Errorf("failed to decode key_base64: %w", err)
			}
			createReq.Key = keyBytes
		}

	case "JKS", "PKCS12", "JCEKS":
		if !data.KeystoreFileB64.IsNull() && !data.KeystoreFileB64.IsUnknown() && data.KeystoreFileB64.ValueString() != "" {
			ksBytes, err := base64.StdEncoding.DecodeString(data.KeystoreFileB64.ValueString())
			if err != nil {
				return nil, fmt.Errorf("failed to decode keystore_file_base64: %w", err)
			}
			createReq.Keystore = ksBytes
		}
	}

	if !data.CaPathB64.IsNull() && !data.CaPathB64.IsUnknown() && data.CaPathB64.ValueString() != "" {
		caBytes, err := base64.StdEncoding.DecodeString(data.CaPathB64.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to decode ca_path_base64: %w", err)
		}
		createReq.CaPath = caBytes
	}

	return createReq, nil
}

func (r *KeystoreResource) flattenKeystore(ks *secretsmanagement.KeystoreResponse, data *KeystoreResourceModel, orgID, envID, sgID string) {
	data.ID = types.StringValue(ks.Meta.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.SecretGroupID = types.StringValue(sgID)
	data.Name = types.StringValue(ks.Name)
	data.Type = types.StringValue(ks.Type)

	if ks.ExpirationDate != "" {
		data.ExpirationDate = types.StringValue(ks.ExpirationDate)
	} else {
		data.ExpirationDate = types.StringValue("")
	}
	if ks.Algorithm != "" {
		data.Algorithm = types.StringValue(ks.Algorithm)
	} else {
		data.Algorithm = types.StringValue("")
	}
}
