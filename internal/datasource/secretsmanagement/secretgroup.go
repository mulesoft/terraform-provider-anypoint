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

var _ datasource.DataSource = &SecretGroupDataSource{}

func NewSecretGroupDataSource() datasource.DataSource {
	return &SecretGroupDataSource{}
}

type SecretGroupDataSource struct {
	client *secretsmgmt.SecretGroupClient
}

type SecretGroupDataSourceModel struct {
	OrganizationID types.String            `tfsdk:"organization_id"`
	EnvironmentID  types.String            `tfsdk:"environment_id"`
	SecretGroups   []SecretGroupItemModel  `tfsdk:"secret_groups"`
}

type SecretGroupItemModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Downloadable types.Bool   `tfsdk:"downloadable"`
	CurrentState types.String `tfsdk:"current_state"`
}

func (d *SecretGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_groups"
}

func (d *SecretGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all secret groups in a given environment.",
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
			"secret_groups": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of secret groups.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The secret group ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the secret group.",
						},
						"downloadable": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the secret group is downloadable.",
						},
						"current_state": schema.StringAttribute{
							Computed:    true,
							Description: "The current state of the secret group (e.g. Clear).",
						},
					},
				},
			},
		},
	}
}

func (d *SecretGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("expected *client.ClientConfig, got %T", req.ProviderData))
		return
	}
	c, err := secretsmgmt.NewSecretGroupClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SecretGroupClient", err.Error())
		return
	}
	d.client = c
}

func (d *SecretGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state SecretGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := state.EnvironmentID.ValueString()

	groups, err := d.client.ListSecretGroups(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing secret groups", err.Error())
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.SecretGroups = make([]SecretGroupItemModel, len(groups))
	for i, g := range groups {
		state.SecretGroups[i] = SecretGroupItemModel{
			ID:           types.StringValue(g.Meta.ID),
			Name:         types.StringValue(g.Name),
			Downloadable: types.BoolValue(g.Downloadable),
			CurrentState: types.StringValue(g.CurrentState),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
