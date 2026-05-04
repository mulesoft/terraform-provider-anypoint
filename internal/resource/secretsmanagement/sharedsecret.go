package secretsmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
)

var (
	_ resource.Resource                = &SharedSecretResource{}
	_ resource.ResourceWithConfigure   = &SharedSecretResource{}
	_ resource.ResourceWithImportState = &SharedSecretResource{}
)

type SharedSecretResource struct {
	client *secretsmanagement.SharedSecretClient
}

type SharedSecretResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	SecretGroupID  types.String `tfsdk:"secret_group_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	ExpirationDate types.String `tfsdk:"expiration_date"`

	// UsernamePassword
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`

	// S3Credential
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`

	// SymmetricKey
	Key types.String `tfsdk:"key"`

	// Blob
	Content types.String `tfsdk:"content"`
}

func NewSharedSecretResource() resource.Resource {
	return &SharedSecretResource{}
}

func (r *SharedSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_shared_secret"
}

func (r *SharedSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a shared secret within a secret group in Anypoint Secrets Manager. " +
			"Supports four types: UsernamePassword, S3Credential, SymmetricKey, and Blob. " +
			"Provide the type-specific fields based on the chosen type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Unique identifier of the shared secret.",
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
				Description:   "Secret group ID that this shared secret belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "Name of the shared secret.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description:   "Type of shared secret: UsernamePassword, S3Credential, SymmetricKey, or Blob.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("UsernamePassword", "S3Credential", "SymmetricKey", "Blob"),
				},
			},
			"expiration_date": schema.StringAttribute{
				Description: "Optional expiration date (e.g. 2026-03-31).",
				Optional:    true,
				Computed:    true,
			},

			// --- UsernamePassword fields ---
			"username": schema.StringAttribute{
				Description: "Username (for UsernamePassword type).",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password (for UsernamePassword type).",
				Optional:    true,
				Sensitive:   true,
			},

			// --- S3Credential fields ---
			"access_key_id": schema.StringAttribute{
				Description: "AWS access key ID (for S3Credential type).",
				Optional:    true,
			},
			"secret_access_key": schema.StringAttribute{
				Description: "AWS secret access key (for S3Credential type).",
				Optional:    true,
				Sensitive:   true,
			},

			// --- SymmetricKey fields ---
			"key": schema.StringAttribute{
				Description: "Base64-encoded symmetric key (for SymmetricKey type).",
				Optional:    true,
				Sensitive:   true,
			},

			// --- Blob fields ---
			"content": schema.StringAttribute{
				Description: "Secret content string (for Blob type).",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *SharedSecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	ssClient, err := secretsmanagement.NewSharedSecretClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Shared Secret Client", err.Error())
		return
	}
	r.client = ssClient
}

func (r *SharedSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SharedSecretResourceModel
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

	createReq := r.expand(&data)

	ss, err := r.client.CreateSharedSecret(ctx, orgID, envID, sgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating shared secret", err.Error())
		return
	}

	r.flatten(ss, &data, orgID, envID, sgID)
	tflog.Trace(ctx, "created shared secret", map[string]interface{}{"id": ss.Meta.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SharedSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SharedSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	ss, err := r.client.GetSharedSecret(ctx, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading shared secret", err.Error())
		return
	}

	// Preserve sensitive fields the API won't return
	savedPassword := data.Password
	savedSecretAccessKey := data.SecretAccessKey
	savedKey := data.Key
	savedContent := data.Content

	r.flatten(ss, &data, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString())

	data.Password = savedPassword
	data.SecretAccessKey = savedSecretAccessKey
	data.Key = savedKey
	data.Content = savedContent

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SharedSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SharedSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	updateReq := r.expand(&plan)

	ss, err := r.client.UpdateSharedSecret(ctx, orgID, state.EnvironmentID.ValueString(), state.SecretGroupID.ValueString(), state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating shared secret", err.Error())
		return
	}

	r.flatten(ss, &plan, orgID, state.EnvironmentID.ValueString(), state.SecretGroupID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SharedSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SharedSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	if err := r.client.DeleteSharedSecret(ctx, orgID, data.EnvironmentID.ValueString(), data.SecretGroupID.ValueString(), data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting shared secret", err.Error())
	}
}

func (r *SharedSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected: organization_id/environment_id/secret_group_id/shared_secret_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// --- Helpers ---

func (r *SharedSecretResource) expand(data *SharedSecretResourceModel) *secretsmanagement.SharedSecret {
	ss := &secretsmanagement.SharedSecret{
		Name: data.Name.ValueString(),
		Type: data.Type.ValueString(),
	}

	if !data.ExpirationDate.IsNull() && !data.ExpirationDate.IsUnknown() && data.ExpirationDate.ValueString() != "" {
		ss.ExpirationDate = data.ExpirationDate.ValueString()
	}

	switch ss.Type {
	case "UsernamePassword":
		ss.Username = data.Username.ValueString()
		ss.Password = data.Password.ValueString()
	case "S3Credential":
		ss.AccessKeyID = data.AccessKeyID.ValueString()
		ss.SecretAccessKey = data.SecretAccessKey.ValueString()
	case "SymmetricKey":
		ss.Key = data.Key.ValueString()
	case "Blob":
		ss.Content = data.Content.ValueString()
	}

	return ss
}

func (r *SharedSecretResource) flatten(ss *secretsmanagement.SharedSecretResponse, data *SharedSecretResourceModel, orgID, envID, sgID string) {
	data.ID = types.StringValue(ss.Meta.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.SecretGroupID = types.StringValue(sgID)
	data.Name = types.StringValue(ss.Name)
	data.Type = types.StringValue(ss.Type)

	if ss.ExpirationDate != "" {
		data.ExpirationDate = types.StringValue(ss.ExpirationDate)
	} else {
		data.ExpirationDate = types.StringValue("")
	}

	// API may return non-sensitive fields
	if ss.Username != "" {
		data.Username = types.StringValue(ss.Username)
	}
	if ss.AccessKeyID != "" {
		data.AccessKeyID = types.StringValue(ss.AccessKeyID)
	}
}
