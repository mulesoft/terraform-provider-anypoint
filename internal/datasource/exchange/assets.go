package exchange

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

var (
	_ datasource.DataSource              = &AssetsDataSource{}
	_ datasource.DataSourceWithConfigure = &AssetsDataSource{}
)

// AssetsDataSource is the data source implementation for listing Exchange assets.
type AssetsDataSource struct {
	client *exchange.AssetClient
}

// AssetsDataSourceModel describes the data source data model.
type AssetsDataSourceModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
	Type           types.String `tfsdk:"type"`
	Search         types.String `tfsdk:"search"`
	Limit          types.Int64  `tfsdk:"limit"`
	Assets         types.List   `tfsdk:"assets"`
}

func NewAssetsDataSource() datasource.DataSource {
	return &AssetsDataSource{}
}

func (d *AssetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exchange_assets"
}

func (d *AssetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Exchange assets in an organization. Supports filtering by type and free-text search.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Description: "The organization ID to list assets from.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Filter by asset type (rest-api, http-api, evented-api, graphql-api, custom, connector, app, template, example, policy, agent, llm, mcp).",
				Optional:    true,
			},
			"search": schema.StringAttribute{
				Description: "Free-text search query to filter assets by name or description.",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Optional cap on the total number of assets to return. When omitted, " +
					"ALL matching assets are returned (the data source paginates through every " +
					"page automatically). When set to a positive value, at most that many assets " +
					"are returned.",
				Optional: true,
			},
			"assets": schema.ListNestedAttribute{
				Description: "The list of Exchange assets matching the query.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.StringAttribute{
							Description: "The group ID of the asset.",
							Computed:    true,
						},
						"asset_id": schema.StringAttribute{
							Description: "The asset ID.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "The latest version of the asset.",
							Computed:    true,
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
							Description: "The asset type.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The lifecycle status.",
							Computed:    true,
						},
						"is_public": schema.BoolAttribute{
							Description: "Whether the asset is publicly visible.",
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
				},
			},
		},
	}
}

func (d *AssetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AssetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AssetsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// limit is an OPTIONAL cap on the total number of assets returned. When it is
	// unset (or <= 0) we fetch EVERY matching asset by paginating to completion —
	// the correct default for a list data source. When set, we honor it as an
	// explicit upper bound. A negative value is treated as "no cap" (fetch all).
	limit := 0 // 0 => fetch all (ListAllAssets walks every page)
	if !data.Limit.IsNull() && !data.Limit.IsUnknown() {
		if v := int(data.Limit.ValueInt64()); v > 0 {
			limit = v
		}
	}

	listReq := &exchange.ListAssetsRequest{
		OrganizationID: data.OrganizationID.ValueString(),
		Limit:          limit,
	}

	if !data.Search.IsNull() && !data.Search.IsUnknown() {
		listReq.Search = data.Search.ValueString()
	}
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		listReq.Type = data.Type.ValueString()
	}

	// ListAllAssets paginates: without it, an org with more matching assets than a
	// single Exchange page would be silently truncated (the response is a bare array
	// with no total, so the truncation is invisible to the caller).
	assets, err := d.client.ListAllAssets(ctx, listReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing exchange assets",
			fmt.Sprintf("Could not list assets for organization %s: %s",
				data.OrganizationID.ValueString(), err.Error()),
		)
		return
	}

	// Define the object type for asset entries
	assetObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"group_id":     types.StringType,
			"asset_id":     types.StringType,
			"version":      types.StringType,
			"name":         types.StringType,
			"description":  types.StringType,
			"type":         types.StringType,
			"status":       types.StringType,
			"is_public":    types.BoolType,
			"created_date": types.StringType,
			"updated_date": types.StringType,
		},
	}

	// Map assets to state
	assetElements := make([]attr.Value, len(assets))
	for i, asset := range assets {
		assetElements[i], _ = types.ObjectValue(
			assetObjectType.AttrTypes,
			map[string]attr.Value{
				"group_id":     types.StringValue(asset.GroupID),
				"asset_id":     types.StringValue(asset.AssetID),
				"version":      types.StringValue(asset.Version),
				"name":         types.StringValue(asset.Name),
				"description":  types.StringValue(asset.Description),
				"type":         types.StringValue(asset.Type),
				"status":       types.StringValue(asset.Status),
				"is_public":    types.BoolValue(asset.IsPublic),
				"created_date": types.StringValue(asset.CreatedDate),
				"updated_date": types.StringValue(asset.UpdatedDate),
			},
		)
	}

	data.Assets = types.ListValueMust(assetObjectType, assetElements)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
