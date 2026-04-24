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

var _ datasource.DataSource = &CertificatePinsetDataSource{}

func NewCertificatePinsetDataSource() datasource.DataSource {
	return &CertificatePinsetDataSource{}
}

type CertificatePinsetDataSource struct {
	client *secretsmgmt.CertificatePinsetClient
}

type CertificatePinsetDataSourceModel struct {
	OrganizationID      types.String                  `tfsdk:"organization_id"`
	EnvironmentID       types.String                  `tfsdk:"environment_id"`
	SecretGroupID       types.String                  `tfsdk:"secret_group_id"`
	CertificatePinsets  []CertificatePinsetItemModel  `tfsdk:"certificate_pinsets"`
}

type CertificatePinsetItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Algorithm      types.String `tfsdk:"algorithm"`
}

func (d *CertificatePinsetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_certificate_pinsets"
}

func (d *CertificatePinsetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all certificate pinsets within a secret group.",
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
			"certificate_pinsets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of certificate pinsets.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The certificate pinset ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the certificate pinset.",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "The expiration date of the certificate pinset.",
						},
						"algorithm": schema.StringAttribute{
							Computed:    true,
							Description: "The algorithm used by the certificate pinset.",
						},
					},
				},
			},
		},
	}
}

func (d *CertificatePinsetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.ClientConfig, got %T", req.ProviderData))
		return
	}
	c, err := secretsmgmt.NewCertificatePinsetClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create CertificatePinsetClient", err.Error())
		return
	}
	d.client = c
}

func (d *CertificatePinsetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state CertificatePinsetDataSourceModel
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

	items, err := d.client.ListCertificatePinsets(ctx, orgID, envID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing certificate pinsets", err.Error())
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.CertificatePinsets = make([]CertificatePinsetItemModel, len(items))
	for i, pin := range items {
		state.CertificatePinsets[i] = CertificatePinsetItemModel{
			ID:             types.StringValue(pin.Meta.ID),
			Name:           types.StringValue(pin.Name),
			ExpirationDate: types.StringValue(pin.ExpirationDate),
			Algorithm:      types.StringValue(pin.Algorithm),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
