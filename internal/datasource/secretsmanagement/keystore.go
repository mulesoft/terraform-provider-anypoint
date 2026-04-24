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

var _ datasource.DataSource = &KeystoreDataSource{}

func NewKeystoreDataSource() datasource.DataSource {
	return &KeystoreDataSource{}
}

type KeystoreDataSource struct {
	client *secretsmgmt.KeystoreClient
}

type KeystoreDataSourceModel struct {
	OrganizationID types.String        `tfsdk:"organization_id"`
	EnvironmentID  types.String        `tfsdk:"environment_id"`
	SecretGroupID  types.String        `tfsdk:"secret_group_id"`
	Keystores      []KeystoreItemModel `tfsdk:"keystores"`
}

type KeystoreItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Algorithm      types.String `tfsdk:"algorithm"`
}

func (d *KeystoreDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_keystores"
}

func (d *KeystoreDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all keystores within a secret group.",
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
			"keystores": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of keystores.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The keystore ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the keystore.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The keystore type (PEM, JKS, PKCS12, JCEKS).",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "The expiration date of the keystore.",
						},
						"algorithm": schema.StringAttribute{
							Computed:    true,
							Description: "The algorithm used by the keystore.",
						},
					},
				},
			},
		},
	}
}

func (d *KeystoreDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.ClientConfig, got %T", req.ProviderData))
		return
	}
	c, err := secretsmgmt.NewKeystoreClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create KeystoreClient", err.Error())
		return
	}
	d.client = c
}

func (d *KeystoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KeystoreDataSourceModel
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

	items, err := d.client.ListKeystores(ctx, orgID, envID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing keystores", err.Error())
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.Keystores = make([]KeystoreItemModel, len(items))
	for i, ks := range items {
		state.Keystores[i] = KeystoreItemModel{
			ID:             types.StringValue(ks.Meta.ID),
			Name:           types.StringValue(ks.Name),
			Type:           types.StringValue(ks.Type),
			ExpirationDate: types.StringValue(ks.ExpirationDate),
			Algorithm:      types.StringValue(ks.Algorithm),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
