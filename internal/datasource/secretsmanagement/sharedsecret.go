package secretsmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	secretsmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
)

var _ datasource.DataSource = &SharedSecretDataSource{}

func NewSharedSecretDataSource() datasource.DataSource {
	return &SharedSecretDataSource{}
}

type SharedSecretDataSource struct {
	client *secretsmgmt.SharedSecretClient
}

type SharedSecretDataSourceModel struct {
	OrganizationID types.String            `tfsdk:"organization_id"`
	EnvironmentID  types.String            `tfsdk:"environment_id"`
	SecretGroupID  types.String            `tfsdk:"secret_group_id"`
	SharedSecrets  []SharedSecretItemModel `tfsdk:"shared_secrets"`
}

type SharedSecretItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Username       types.String `tfsdk:"username"`
	AccessKeyID    types.String `tfsdk:"access_key_id"`
}

func (d *SharedSecretDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_shared_secrets"
}

func (d *SharedSecretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all shared secrets within a secret group. Sensitive values are not returned by the API.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. Defaults to the provider organization.",
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "The environment ID.",
			},
			"secret_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The secret group ID.",
			},
			"shared_secrets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of shared secrets.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The shared secret ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the shared secret.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The shared secret type (UsernamePassword, S3Credential, SymmetricKey, Blob).",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "The expiration date of the shared secret.",
						},
						"username": schema.StringAttribute{
							Computed:    true,
							Description: "Username, returned only for UsernamePassword type.",
						},
						"access_key_id": schema.StringAttribute{
							Computed:    true,
							Description: "Access key ID, returned only for S3Credential type.",
						},
					},
				},
			},
		},
	}
}

func (d *SharedSecretDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.Config, got %T", req.ProviderData))
		return
	}
	c, err := secretsmgmt.NewSharedSecretClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SharedSecretClient", err.Error())
		return
	}
	d.client = c
}

func (d *SharedSecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state SharedSecretDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := state.EnvironmentID.ValueString()
	sgID := state.SecretGroupID.ValueString()

	items, err := d.client.ListSharedSecrets(ctx, orgID, envID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing shared secrets", err.Error())
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.SharedSecrets = make([]SharedSecretItemModel, len(items))
	for i, ss := range items {
		state.SharedSecrets[i] = SharedSecretItemModel{
			ID:             types.StringValue(ss.Meta.ID),
			Name:           types.StringValue(ss.Name),
			Type:           types.StringValue(ss.Type),
			ExpirationDate: types.StringValue(ss.ExpirationDate),
			Username:       types.StringValue(ss.Username),
			AccessKeyID:    types.StringValue(ss.AccessKeyID),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
