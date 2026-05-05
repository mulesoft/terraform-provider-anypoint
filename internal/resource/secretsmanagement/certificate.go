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
	_ resource.Resource                = &CertificateResource{}
	_ resource.ResourceWithConfigure   = &CertificateResource{}
	_ resource.ResourceWithImportState = &CertificateResource{}
)

type CertificateResource struct {
	client *secretsmanagement.CertificateClient
}

type CertificateResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	SecretGroupID  types.String `tfsdk:"secret_group_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	CertStoreB64   types.String `tfsdk:"certificate_base64"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Algorithm      types.String `tfsdk:"algorithm"`
}

func NewCertificateResource() resource.Resource {
	return &CertificateResource{}
}

func (r *CertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_certificate"
}

func (r *CertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a certificate within a secret group in Anypoint Secrets Manager. " +
			"Supports PEM, JKS, PKCS12, and JCEKS formats.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Unique identifier of the certificate.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID.",
				Optional:    true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Description:   "Environment ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"secret_group_id": schema.StringAttribute{
				Description:   "Secret group ID that this certificate belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "Name of the certificate.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Certificate format: PEM, JKS, PKCS12, or JCEKS.",
				Optional:    true, Computed: true,
				Default:       stringdefault.StaticString("PEM"),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf("PEM", "JKS", "PKCS12", "JCEKS")},
			},
			"certificate_base64": schema.StringAttribute{
				Description: "Base64-encoded certificate file content. " +
					"For PEM: base64encode(file(\"cert.pem\")). For binary: filebase64(\"cert.der\").",
				Required:  true,
				Sensitive: true,
			},
			"expiration_date": schema.StringAttribute{
				Description: "Expiration date of the certificate.",
				Computed:    true,
			},
			"algorithm": schema.StringAttribute{
				Description: "Signature algorithm of the certificate.",
				Computed:    true,
			},
		},
	}
}

func (r *CertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	certClient, err := secretsmanagement.NewCertificateClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Certificate Client", err.Error())
		return
	}
	r.client = certClient
}

func (r *CertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CertificateResourceModel
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

	certBytes, err := base64.StdEncoding.DecodeString(data.CertStoreB64.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error decoding certificate", "Failed to decode certificate_base64: "+err.Error())
		return
	}

	createReq := &secretsmanagement.CreateCertificateRequest{
		Name:     data.Name.ValueString(),
		Type:     data.Type.ValueString(),
		CertFile: certBytes,
	}

	cert, err := r.client.CreateCertificate(ctx, orgID, envID, sgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating certificate", err.Error())
		return
	}

	r.flatten(cert, &data, orgID, envID, sgID)
	tflog.Trace(ctx, "created certificate", map[string]interface{}{"id": cert.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	cert, err := r.client.GetCertificate(ctx, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading certificate", err.Error())
		return
	}

	savedCert := data.CertStoreB64
	r.flatten(cert, &data, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString())
	data.CertStoreB64 = savedCert

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	certBytes, err := base64.StdEncoding.DecodeString(plan.CertStoreB64.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error decoding certificate", "Failed to decode certificate_base64: "+err.Error())
		return
	}

	updateReq := &secretsmanagement.CreateCertificateRequest{
		Name:     plan.Name.ValueString(),
		Type:     plan.Type.ValueString(),
		CertFile: certBytes,
	}

	cert, err := r.client.UpdateCertificate(ctx, orgID, state.EnvironmentID.ValueString(), state.SecretGroupID.ValueString(), state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating certificate", err.Error())
		return
	}

	r.flatten(cert, &plan, orgID, state.EnvironmentID.ValueString(), state.SecretGroupID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The SM API has no individual sub-resource DELETE endpoint (returns 405).
	// Sub-resources are removed by deleting the parent secret group.
	// Removing from Terraform state only; platform cleanup is the secret group's responsibility.
	var data CertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	tflog.Trace(ctx, "removed certificate from state (no-op delete)", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *CertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected: organization_id/environment_id/secret_group_id/certificate_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

func (r *CertificateResource) flatten(cert *secretsmanagement.CertificateResponse, data *CertificateResourceModel, orgID, envID, sgID string) {
	data.ID = types.StringValue(cert.Meta.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.SecretGroupID = types.StringValue(sgID)
	data.Name = types.StringValue(cert.Name)
	data.Type = types.StringValue(cert.Type)

	if cert.ExpirationDate != "" {
		data.ExpirationDate = types.StringValue(cert.ExpirationDate)
	} else {
		data.ExpirationDate = types.StringValue("")
	}
	if cert.Algorithm != "" {
		data.Algorithm = types.StringValue(cert.Algorithm)
	} else {
		data.Algorithm = types.StringValue("")
	}
}
