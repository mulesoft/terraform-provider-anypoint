package secretsmanagement

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
)

var (
	_ resource.Resource                = &CertificatePinsetResource{}
	_ resource.ResourceWithConfigure   = &CertificatePinsetResource{}
	_ resource.ResourceWithImportState = &CertificatePinsetResource{}
)

type CertificatePinsetResource struct {
	client *secretsmanagement.CertificatePinsetClient
}

type CertificatePinsetResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	SecretGroupID  types.String `tfsdk:"secret_group_id"`
	Name           types.String `tfsdk:"name"`
	PinsetB64      types.String `tfsdk:"certificate_pinset_base64"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Algorithm      types.String `tfsdk:"algorithm"`
}

func NewCertificatePinsetResource() resource.Resource {
	return &CertificatePinsetResource{}
}

func (r *CertificatePinsetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_certificate_pinset"
}

func (r *CertificatePinsetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a certificate pinset within a secret group in Anypoint Secrets Manager. " +
			"A certificate pinset is used for certificate pinning validation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Unique identifier of the certificate pinset.",
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
				Description:   "Secret group ID that this certificate pinset belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "Name of the certificate pinset.",
				Required:    true,
			},
			"certificate_pinset_base64": schema.StringAttribute{
				Description: "Base64-encoded certificate file for pinning. " +
					"For PEM: base64encode(file(\"cert.pem\")).",
				Required:  true,
				Sensitive: true,
			},
			"expiration_date": schema.StringAttribute{
				Description: "Expiration date of the pinned certificate.",
				Computed:    true,
			},
			"algorithm": schema.StringAttribute{
				Description: "Signature algorithm of the pinned certificate.",
				Computed:    true,
			},
		},
	}
}

func (r *CertificatePinsetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T.", req.ProviderData))
		return
	}
	pinClient, err := secretsmanagement.NewCertificatePinsetClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Certificate Pinset Client", err.Error())
		return
	}
	r.client = pinClient
}

func (r *CertificatePinsetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CertificatePinsetResourceModel
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

	pinBytes, err := base64.StdEncoding.DecodeString(data.PinsetB64.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error decoding certificate pinset", "Failed to decode certificate_pinset_base64: "+err.Error())
		return
	}

	createReq := &secretsmanagement.CreateCertificatePinsetRequest{
		Name:    data.Name.ValueString(),
		PinFile: pinBytes,
	}

	pin, err := r.client.CreateCertificatePinset(ctx, orgID, envID, sgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating certificate pinset", err.Error())
		return
	}

	r.flatten(pin, &data, orgID, envID, sgID)
	tflog.Trace(ctx, "created certificate pinset", map[string]interface{}{"id": pin.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CertificatePinsetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CertificatePinsetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	pin, err := r.client.GetCertificatePinset(ctx, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading certificate pinset", err.Error())
		return
	}

	savedPinset := data.PinsetB64
	r.flatten(pin, &data, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString())
	data.PinsetB64 = savedPinset

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CertificatePinsetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CertificatePinsetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	pinBytes, err := base64.StdEncoding.DecodeString(plan.PinsetB64.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error decoding certificate pinset", "Failed to decode certificate_pinset_base64: "+err.Error())
		return
	}

	updateReq := &secretsmanagement.CreateCertificatePinsetRequest{
		Name:    plan.Name.ValueString(),
		PinFile: pinBytes,
	}

	pin, err := r.client.UpdateCertificatePinset(ctx, orgID, state.EnvironmentID.ValueString(), state.SecretGroupID.ValueString(), state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating certificate pinset", err.Error())
		return
	}

	r.flatten(pin, &plan, orgID, state.EnvironmentID.ValueString(), state.SecretGroupID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CertificatePinsetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CertificatePinsetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	if err := r.client.DeleteCertificatePinset(ctx, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString(), data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting certificate pinset", err.Error())
	}
}

func (r *CertificatePinsetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected: organization_id/environment_id/secret_group_id/certificate_pinset_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

func (r *CertificatePinsetResource) flatten(pin *secretsmanagement.CertificatePinsetResponse, data *CertificatePinsetResourceModel, orgID, envID, sgID string) {
	data.ID = types.StringValue(pin.Meta.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.SecretGroupID = types.StringValue(sgID)
	data.Name = types.StringValue(pin.Name)

	if pin.ExpirationDate != "" {
		data.ExpirationDate = types.StringValue(pin.ExpirationDate)
	} else {
		data.ExpirationDate = types.StringValue("")
	}
	if pin.Algorithm != "" {
		data.Algorithm = types.StringValue(pin.Algorithm)
	} else {
		data.Algorithm = types.StringValue("")
	}
}
