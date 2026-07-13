package exchange

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

var (
	_ datasource.DataSource              = &AssetDataSource{}
	_ datasource.DataSourceWithConfigure = &AssetDataSource{}
)

// AssetDataSource is the data source implementation.
type AssetDataSource struct {
	client *exchange.AssetClient
}

// AssetDataSourceModel describes the data source data model.
type AssetDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.String `tfsdk:"group_id"`
	AssetID types.String `tfsdk:"asset_id"`
	Version types.String `tfsdk:"version"`

	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Type         types.String `tfsdk:"type"`
	Status       types.String `tfsdk:"status"`
	ContactName  types.String `tfsdk:"contact_name"`
	ContactEmail types.String `tfsdk:"contact_email"`
	Manager      types.String `tfsdk:"manager"`

	IsPublic     types.Bool   `tfsdk:"is_public"`
	IsSnapshot   types.Bool   `tfsdk:"is_snapshot"`
	MinorVersion types.String `tfsdk:"minor_version"`
	VersionGroup types.String `tfsdk:"version_group"`
	CreatedDate  types.String `tfsdk:"created_date"`
	UpdatedDate  types.String `tfsdk:"updated_date"`
}

func NewAssetDataSource() datasource.DataSource {
	return &AssetDataSource{}
}

func (d *AssetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exchange_asset"
}

func (d *AssetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a specific Exchange asset version by its GAV coordinates (groupId, assetId, version).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier (groupId/assetId/version).",
				Computed:    true,
			},
			"group_id": schema.StringAttribute{
				Description: "The group ID of the asset.",
				Required:    true,
			},
			"asset_id": schema.StringAttribute{
				Description: "The asset ID.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: "The semantic version of the asset.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The display name of the asset.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The asset description.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The asset type (custom, rest-api, graphql-api, etc.).",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The lifecycle status (published, deprecated, development).",
				Computed:    true,
			},
			"contact_name": schema.StringAttribute{
				Description: "Contact person name.",
				Computed:    true,
			},
			"contact_email": schema.StringAttribute{
				Description: "Contact email.",
				Computed:    true,
			},
			"manager": schema.StringAttribute{
				Description: "Asset manager.",
				Computed:    true,
			},
			"is_public": schema.BoolAttribute{
				Description: "Whether the asset is publicly visible.",
				Computed:    true,
			},
			"is_snapshot": schema.BoolAttribute{
				Description: "Whether this is a snapshot version.",
				Computed:    true,
			},
			"minor_version": schema.StringAttribute{
				Description: "The minor version (e.g. '1.0').",
				Computed:    true,
			},
			"version_group": schema.StringAttribute{
				Description: "The version group.",
				Computed:    true,
			},
			"created_date": schema.StringAttribute{
				Description: "When the asset was created.",
				Computed:    true,
			},
			"updated_date": schema.StringAttribute{
				Description: "When the asset was last updated.",
				Computed:    true,
			},
		},
	}
}

func (d *AssetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	assetClient, err := exchange.NewAssetClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Exchange Asset Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	d.client = assetClient
}

func (d *AssetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AssetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	asset, err := d.client.GetAsset(ctx,
		data.GroupID.ValueString(),
		data.AssetID.ValueString(),
		data.Version.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading exchange asset",
			fmt.Sprintf("Could not read asset %s/%s/%s: %s",
				data.GroupID.ValueString(),
				data.AssetID.ValueString(),
				data.Version.ValueString(),
				err.Error()),
		)
		return
	}

	// Map response to state
	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", asset.GroupID, asset.AssetID, asset.Version))
	data.Name = types.StringValue(asset.Name)
	data.Description = types.StringValue(asset.Description)
	data.Type = types.StringValue(asset.Type)
	data.Status = types.StringValue(asset.Status)
	data.IsPublic = types.BoolValue(asset.IsPublic)
	data.IsSnapshot = types.BoolValue(asset.IsSnapshot)
	data.MinorVersion = types.StringValue(asset.MinorVersion)
	data.VersionGroup = types.StringValue(asset.VersionGroup)
	data.CreatedDate = types.StringValue(asset.CreatedDate)
	data.UpdatedDate = types.StringValue(asset.UpdatedDate)

	if asset.ContactName != nil {
		data.ContactName = types.StringValue(*asset.ContactName)
	} else {
		data.ContactName = types.StringValue("")
	}
	if asset.ContactEmail != nil {
		data.ContactEmail = types.StringValue(*asset.ContactEmail)
	} else {
		data.ContactEmail = types.StringValue("")
	}
	if asset.Manager != nil {
		data.Manager = types.StringValue(*asset.Manager)
	} else {
		data.Manager = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
