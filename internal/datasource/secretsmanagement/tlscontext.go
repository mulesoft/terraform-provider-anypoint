package secretsmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	secretsmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
)

var _ datasource.DataSource = &TLSContextDataSource{}

func NewTLSContextDataSource() datasource.DataSource {
	return &TLSContextDataSource{}
}

type TLSContextDataSource struct {
	client *secretsmgmt.TLSContextClient
}

type TLSContextDataSourceModel struct {
	OrganizationID types.String          `tfsdk:"organization_id"`
	EnvironmentID  types.String          `tfsdk:"environment_id"`
	SecretGroupID  types.String          `tfsdk:"secret_group_id"`
	TLSContexts    []TLSContextItemModel `tfsdk:"tls_contexts"`
}

type TLSContextItemModel struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Target                     types.String `tfsdk:"target"`
	MinTLSVersion              types.String `tfsdk:"min_tls_version"`
	MaxTLSVersion              types.String `tfsdk:"max_tls_version"`
	ExpirationDate             types.String `tfsdk:"expiration_date"`
	EnableClientCertValidation types.Bool   `tfsdk:"enable_client_cert_validation"`
	SkipServerCertValidation   types.Bool   `tfsdk:"skip_server_cert_validation"`
	AlpnProtocols              types.String `tfsdk:"alpn_protocols"`
	CipherSuites               types.String `tfsdk:"cipher_suites"`
}

func (d *TLSContextDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_group_tls_contexts"
}

func (d *TLSContextDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all TLS contexts within a secret group.",
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
			"tls_contexts": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of TLS contexts.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The TLS context ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the TLS context.",
						},
						"target": schema.StringAttribute{
							Computed:    true,
							Description: "The target (e.g. FlexGateway).",
						},
						"min_tls_version": schema.StringAttribute{
							Computed:    true,
							Description: "Minimum TLS version.",
						},
						"max_tls_version": schema.StringAttribute{
							Computed:    true,
							Description: "Maximum TLS version.",
						},
						"expiration_date": schema.StringAttribute{
							Computed:    true,
							Description: "The expiration date of the TLS context.",
						},
						"enable_client_cert_validation": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether client certificate validation is enabled.",
						},
						"skip_server_cert_validation": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether server certificate validation is skipped.",
						},
						"alpn_protocols": schema.StringAttribute{
							Computed:    true,
							Description: "Comma-separated list of ALPN protocols.",
						},
						"cipher_suites": schema.StringAttribute{
							Computed:    true,
							Description: "Comma-separated list of cipher suites.",
						},
					},
				},
			},
		},
	}
}

func (d *TLSContextDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.Config, got %T", req.ProviderData))
		return
	}
	c, err := secretsmgmt.NewTLSContextClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create TLSContextClient", err.Error())
		return
	}
	d.client = c
}

func (d *TLSContextDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TLSContextDataSourceModel
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

	items, err := d.client.ListTLSContexts(ctx, orgID, envID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing TLS contexts", err.Error())
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.TLSContexts = make([]TLSContextItemModel, len(items))
	for i, tls := range items {
		var enableClientCert, skipServerCert bool
		if tls.InboundSettings != nil {
			enableClientCert = tls.InboundSettings.EnableClientCertValidation
		}
		if tls.OutboundSettings != nil {
			skipServerCert = tls.OutboundSettings.SkipServerCertValidation
		}
		state.TLSContexts[i] = TLSContextItemModel{
			ID:                         types.StringValue(tls.Meta.ID),
			Name:                       types.StringValue(tls.Name),
			Target:                     types.StringValue(tls.Target),
			MinTLSVersion:              types.StringValue(tls.MinTLSVersion),
			MaxTLSVersion:              types.StringValue(tls.MaxTLSVersion),
			ExpirationDate:             types.StringValue(tls.ExpirationDate),
			EnableClientCertValidation: types.BoolValue(enableClientCert),
			SkipServerCertValidation:   types.BoolValue(skipServerCert),
			AlpnProtocols:              types.StringValue(strings.Join(tls.AlpnProtocols, ",")),
			CipherSuites:               types.StringValue(strings.Join(tls.CipherSuites, ",")),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
