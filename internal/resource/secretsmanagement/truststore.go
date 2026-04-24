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
	_ resource.Resource                = &TruststoreResource{}
	_ resource.ResourceWithConfigure   = &TruststoreResource{}
	_ resource.ResourceWithImportState = &TruststoreResource{}
)

type TruststoreResource struct {
	client *secretsmanagement.TruststoreClient
}

type TruststoreResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	SecretGroupID  types.String `tfsdk:"secret_group_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	TrustStoreB64  types.String `tfsdk:"truststore_base64"`
	Passphrase     types.String `tfsdk:"passphrase"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Algorithm      types.String `tfsdk:"algorithm"`
}

func NewTruststoreResource() resource.Resource {
	return &TruststoreResource{}
}

func (r *TruststoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_truststore"
}

func (r *TruststoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a truststore within a secret group in Anypoint Secrets Manager. " +
			"Supports PEM, JKS, PKCS12, and JCEKS formats. " +
			"Use base64encode(file(...)) for PEM text files or filebase64(...) for binary JKS/PKCS12 files.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the truststore.",
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
				Description: "Secret group ID that this truststore belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the truststore.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Truststore format: PEM, JKS, PKCS12, or JCEKS.",
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
			"truststore_base64": schema.StringAttribute{
				Description: "Base64-encoded truststore file content. " +
					"For PEM: base64encode(file(\"truststore.pem\")). " +
					"For JKS/PKCS12: filebase64(\"truststore.jks\").",
				Required:  true,
				Sensitive: true,
			},
			"passphrase": schema.StringAttribute{
				Description: "Passphrase for the truststore. Required for JKS, PKCS12, and JCEKS formats.",
				Optional:    true,
				Sensitive:   true,
			},
			"expiration_date": schema.StringAttribute{
				Description: "Expiration date of the certificate in the truststore.",
				Computed:    true,
			},
			"algorithm": schema.StringAttribute{
				Description: "Signature algorithm of the certificate.",
				Computed:    true,
			},
		},
	}
}

func (r *TruststoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	tsClient, err := secretsmanagement.NewTruststoreClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Truststore Client",
			"An unexpected error occurred: "+err.Error(),
		)
		return
	}
	r.client = tsClient
}

func (r *TruststoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TruststoreResourceModel
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
		resp.Diagnostics.AddError("Error building truststore request", err.Error())
		return
	}

	ts, err := r.client.CreateTruststore(ctx, orgID, envID, sgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating truststore", "Could not create truststore: "+err.Error())
		return
	}

	r.flattenTruststore(ts, &data, orgID, envID, sgID)
	tflog.Trace(ctx, "created truststore", map[string]interface{}{"id": ts.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TruststoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TruststoreResourceModel
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

	ts, err := r.client.GetTruststore(ctx, orgID, envID, sgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading truststore", "Could not read truststore: "+err.Error())
		return
	}

	savedTrustStore := data.TrustStoreB64
	savedPassphrase := data.Passphrase

	r.flattenTruststore(ts, &data, orgID, envID, sgID)

	data.TrustStoreB64 = savedTrustStore
	data.Passphrase = savedPassphrase

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TruststoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TruststoreResourceModel
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
		resp.Diagnostics.AddError("Error building truststore update request", err.Error())
		return
	}

	ts, err := r.client.UpdateTruststore(ctx, orgID, envID, sgID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating truststore", "Could not update truststore: "+err.Error())
		return
	}

	r.flattenTruststore(ts, &plan, orgID, envID, sgID)
	tflog.Trace(ctx, "updated truststore", map[string]interface{}{"id": ts.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TruststoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TruststoreResourceModel
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

	if err := r.client.DeleteTruststore(ctx, orgID, envID, sgID, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting truststore", "Could not delete truststore: "+err.Error())
		return
	}
	tflog.Trace(ctx, "deleted truststore", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *TruststoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/secret_group_id/truststore_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// --- Helpers ---

func (r *TruststoreResource) expandRequest(data *TruststoreResourceModel) (*secretsmanagement.CreateTruststoreRequest, error) {
	createReq := &secretsmanagement.CreateTruststoreRequest{
		Name:       data.Name.ValueString(),
		Type:       data.Type.ValueString(),
		Passphrase: data.Passphrase.ValueString(),
	}

	if !data.TrustStoreB64.IsNull() && !data.TrustStoreB64.IsUnknown() && data.TrustStoreB64.ValueString() != "" {
		tsBytes, err := base64.StdEncoding.DecodeString(data.TrustStoreB64.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to decode truststore_base64: %w", err)
		}
		createReq.TrustStore = tsBytes
	}

	return createReq, nil
}

func (r *TruststoreResource) flattenTruststore(ts *secretsmanagement.TruststoreResponse, data *TruststoreResourceModel, orgID, envID, sgID string) {
	data.ID = types.StringValue(ts.Meta.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.SecretGroupID = types.StringValue(sgID)
	data.Name = types.StringValue(ts.Name)
	data.Type = types.StringValue(ts.Type)

	if ts.ExpirationDate != "" {
		data.ExpirationDate = types.StringValue(ts.ExpirationDate)
	} else {
		data.ExpirationDate = types.StringValue("")
	}
	if ts.Algorithm != "" {
		data.Algorithm = types.StringValue(ts.Algorithm)
	} else {
		data.Algorithm = types.StringValue("")
	}
}
