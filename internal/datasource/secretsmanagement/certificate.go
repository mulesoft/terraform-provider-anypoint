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

var _ datasource.DataSource = &CertificateDataSource{}

func NewCertificateDataSource() datasource.DataSource {
	return &CertificateDataSource{}
}

type CertificateDataSource struct {
	client *secretsmgmt.CertificateClient
}

type CertificateDataSourceModel struct {
	OrganizationID types.String           `tfsdk:"organization_id"`
	EnvironmentID  types.String           `tfsdk:"environment_id"`
	SecretGroupID  types.String           `tfsdk:"secret_group_id"`
	Certificates   []CertificateItemModel `tfsdk:"certificates"`
}

type CertificateItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Algorithm      types.String `tfsdk:"algorithm"`
}

func (d *CertificateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_certificates"
}

func (d *CertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all certificates within a secret group.",
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
			"certificates": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of certificates.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The certificate ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the certificate.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The certificate type (PEM, JKS, PKCS12, JCEKS).",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "The expiration date of the certificate.",
						},
						"algorithm": schema.StringAttribute{
							Computed:    true,
							Description: "The algorithm used by the certificate.",
						},
					},
				},
			},
		},
	}
}

func (d *CertificateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.ClientConfig, got %T", req.ProviderData))
		return
	}
	c, err := secretsmgmt.NewCertificateClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create CertificateClient", err.Error())
		return
	}
	d.client = c
}

func (d *CertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state CertificateDataSourceModel
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

	items, err := d.client.ListCertificates(ctx, orgID, envID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing certificates", err.Error())
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.Certificates = make([]CertificateItemModel, len(items))
	for i, cert := range items {
		state.Certificates[i] = CertificateItemModel{
			ID:             types.StringValue(cert.Meta.ID),
			Name:           types.StringValue(cert.Name),
			Type:           types.StringValue(cert.Type),
			ExpirationDate: types.StringValue(cert.ExpirationDate),
			Algorithm:      types.StringValue(cert.Algorithm),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
