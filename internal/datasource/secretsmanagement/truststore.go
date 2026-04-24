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

var _ datasource.DataSource = &TruststoreDataSource{}

func NewTruststoreDataSource() datasource.DataSource {
	return &TruststoreDataSource{}
}

type TruststoreDataSource struct {
	client *secretsmgmt.TruststoreClient
}

type TruststoreDataSourceModel struct {
	OrganizationID types.String          `tfsdk:"organization_id"`
	EnvironmentID  types.String          `tfsdk:"environment_id"`
	SecretGroupID  types.String          `tfsdk:"secret_group_id"`
	Truststores    []TruststoreItemModel `tfsdk:"truststores"`
}

type TruststoreItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Algorithm      types.String `tfsdk:"algorithm"`
}

func (d *TruststoreDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_truststores"
}

func (d *TruststoreDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all truststores within a secret group.",
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
			"truststores": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of truststores.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The truststore ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the truststore.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The truststore type (PEM, JKS, PKCS12, JCEKS).",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "The expiration date of the truststore.",
						},
						"algorithm": schema.StringAttribute{
							Computed:    true,
							Description: "The algorithm used by the truststore.",
						},
					},
				},
			},
		},
	}
}

func (d *TruststoreDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.ClientConfig, got %T", req.ProviderData))
		return
	}
	c, err := secretsmgmt.NewTruststoreClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create TruststoreClient", err.Error())
		return
	}
	d.client = c
}

func (d *TruststoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TruststoreDataSourceModel
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

	items, err := d.client.ListTruststores(ctx, orgID, envID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing truststores", err.Error())
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.Truststores = make([]TruststoreItemModel, len(items))
	for i, ts := range items {
		state.Truststores[i] = TruststoreItemModel{
			ID:             types.StringValue(ts.Meta.ID),
			Name:           types.StringValue(ts.Name),
			Type:           types.StringValue(ts.Type),
			ExpirationDate: types.StringValue(ts.ExpirationDate),
			Algorithm:      types.StringValue(ts.Algorithm),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
