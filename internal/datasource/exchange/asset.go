package exchange

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

	// Nested collections — surfaced so consumers see the full asset (parity with the
	// resource's readback), NOT just its scalar metadata. All are Computed.
	Tags               types.List   `tfsdk:"tags"`
	Pages              types.List   `tfsdk:"pages"`
	Instances          types.List   `tfsdk:"instances"`
	Categories         types.List   `tfsdk:"categories"`
	CustomFields       types.List   `tfsdk:"custom_fields"`
	TermsAndConditions types.String `tfsdk:"terms_and_conditions"`
}

// --- Nested object types (mirror the resource schema shapes) ---

func dsPageObjectType() attr.Type {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"page_name": types.StringType,
		"content":   types.StringType,
		"page_path": types.StringType,
	}}
}

func dsInstanceObjectType() attr.Type {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":         types.StringType,
		"endpoint_uri": types.StringType,
		"is_public":    types.BoolType,
		"instance_id":  types.StringType,
	}}
}

func dsKeyValuesObjectType() attr.Type {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"key":    types.StringType,
		"values": types.ListType{ElemType: types.StringType},
	}}
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
			"tags": schema.ListAttribute{
				Description: "The asset tags (labels).",
				ElementType: types.StringType,
				Computed:    true,
			},
			"terms_and_conditions": schema.StringAttribute{
				Description: "The markdown content of the asset's Terms & Conditions portal page (empty if none).",
				Computed:    true,
			},
			"pages": schema.ListNestedAttribute{
				Description: "The asset's documentation portal pages (excludes synthetic and Terms & Conditions pages).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"page_name": schema.StringAttribute{
							Description: "The page name.",
							Computed:    true,
						},
						"content": schema.StringAttribute{
							Description: "The markdown content of the page.",
							Computed:    true,
						},
						"page_path": schema.StringAttribute{
							Description: "The API-assigned path of the page.",
							Computed:    true,
						},
					},
				},
			},
			"instances": schema.ListNestedAttribute{
				Description: "The asset's external (non-managed) API instances.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The instance name.",
							Computed:    true,
						},
						"endpoint_uri": schema.StringAttribute{
							Description: "The instance endpoint URI.",
							Computed:    true,
						},
						"is_public": schema.BoolAttribute{
							Description: "Whether the instance is publicly visible.",
							Computed:    true,
						},
						"instance_id": schema.StringAttribute{
							Description: "The API-assigned instance identifier.",
							Computed:    true,
						},
					},
				},
			},
			"categories": schema.ListNestedAttribute{
				Description: "The category assignments on the asset.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The category key (display name).",
							Computed:    true,
						},
						"values": schema.ListAttribute{
							Description: "The assigned category values.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
			"custom_fields": schema.ListNestedAttribute{
				Description: "The custom-field assignments on the asset.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The custom-field key.",
							Computed:    true,
						},
						"values": schema.ListAttribute{
							Description: "The assigned custom-field values.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
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

	// --- Nested collections (parity with the resource's readback) ---
	// tags / categories / custom_fields / instances all come from the GetAsset
	// response above (no extra calls). pages + terms_and_conditions each need
	// portal-page reads; failures there are non-fatal (empty result) because a
	// data source should surface what it can rather than erroring the whole read.
	data.Tags = mapTagsToList(asset.Labels)
	data.Categories = mapKeyValuesToList(categoriesToKV(asset.Categories))
	data.CustomFields = mapKeyValuesToList(customFieldsToKV(asset.CustomFields))
	data.Instances = mapInstancesToList(asset.Instances)
	data.Pages = d.readPages(ctx, data.GroupID.ValueString(), data.AssetID.ValueString(), data.Version.ValueString())
	data.TermsAndConditions = d.readTerms(ctx, data.GroupID.ValueString(), data.AssetID.ValueString(), data.Version.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// --- Mapping helpers (data-source local; no prior-state reordering needed) ---

func mapTagsToList(labels []string) types.List {
	elems := make([]attr.Value, len(labels))
	for i, l := range labels {
		elems[i] = types.StringValue(l)
	}
	return types.ListValueMust(types.StringType, elems)
}

// kv is a normalized key→values pair used for both categories and custom fields.
type kv struct {
	key    string
	values []string
}

func categoriesToKV(cats []exchange.Category) []kv {
	out := make([]kv, 0, len(cats))
	for _, c := range cats {
		out = append(out, kv{key: c.Key, values: c.Value})
	}
	return out
}

func customFieldsToKV(fields []exchange.CustomField) []kv {
	out := make([]kv, 0, len(fields))
	for _, f := range fields {
		// The API returns value as interface{} — could be string, []string, or []interface{}.
		var values []string
		switch v := f.Value.(type) {
		case string:
			values = []string{v}
		case []string:
			values = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					values = append(values, s)
				}
			}
		}
		out = append(out, kv{key: f.Key, values: values})
	}
	return out
}

func mapKeyValuesToList(pairs []kv) types.List {
	elems := make([]attr.Value, 0, len(pairs))
	for _, p := range pairs {
		valAttrs := make([]attr.Value, len(p.values))
		for i, v := range p.values {
			valAttrs[i] = types.StringValue(v)
		}
		obj := types.ObjectValueMust(
			map[string]attr.Type{"key": types.StringType, "values": types.ListType{ElemType: types.StringType}},
			map[string]attr.Value{
				"key":    types.StringValue(p.key),
				"values": types.ListValueMust(types.StringType, valAttrs),
			},
		)
		elems = append(elems, obj)
	}
	return types.ListValueMust(dsKeyValuesObjectType(), elems)
}

// mapInstancesToList surfaces only external (non-managed) instances — matching the
// resource, whose `instances` attribute manages exactly those.
func mapInstancesToList(raw []interface{}) types.List {
	elems := make([]attr.Value, 0, len(raw))
	for _, inst := range raw {
		instMap, ok := inst.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := instMap["type"].(string); t != "external" {
			continue
		}
		name, _ := instMap["name"].(string)
		endpointURI, _ := instMap["endpointUri"].(string)
		isPublic, _ := instMap["isPublic"].(bool)
		id, _ := instMap["id"].(string)
		obj := types.ObjectValueMust(
			map[string]attr.Type{
				"name":         types.StringType,
				"endpoint_uri": types.StringType,
				"is_public":    types.BoolType,
				"instance_id":  types.StringType,
			},
			map[string]attr.Value{
				"name":         types.StringValue(name),
				"endpoint_uri": types.StringValue(endpointURI),
				"is_public":    types.BoolValue(isPublic),
				"instance_id":  types.StringValue(id),
			},
		)
		elems = append(elems, obj)
	}
	return types.ListValueMust(dsInstanceObjectType(), elems)
}

// readPages lists the asset's documentation portal pages and their content. It
// mirrors the resource's readPagesIntoState filtering (drop synthetic + .terms
// pages) but needs no reordering: a data source has no prior state to preserve.
// Any error is non-fatal — the pages list is simply left empty.
func (d *AssetDataSource) readPages(ctx context.Context, groupID, assetID, version string) types.List {
	pages, err := d.client.ListPortalPages(ctx, groupID, assetID, version)
	if err != nil {
		return types.ListValueMust(dsPageObjectType(), []attr.Value{})
	}

	elems := make([]attr.Value, 0, len(pages))
	for _, p := range pages {
		if p.Synthetic || p.Name == ".terms" || strings.HasSuffix(p.Path, "/.terms") {
			continue
		}
		content, contentErr := d.client.GetPortalPageContent(ctx, groupID, assetID, version, p.Path)
		if contentErr != nil {
			content = ""
		}
		obj := types.ObjectValueMust(
			map[string]attr.Type{
				"page_name": types.StringType,
				"content":   types.StringType,
				"page_path": types.StringType,
			},
			map[string]attr.Value{
				"page_name": types.StringValue(p.Name),
				"content":   types.StringValue(content),
				"page_path": types.StringValue(p.Path),
			},
		)
		elems = append(elems, obj)
	}
	return types.ListValueMust(dsPageObjectType(), elems)
}

// readTerms fetches the special ".terms" portal page content. Non-fatal on error
// (returns empty), matching the resource's readTermsIntoState behavior.
func (d *AssetDataSource) readTerms(ctx context.Context, groupID, assetID, version string) types.String {
	content, err := d.client.GetPortalPageContent(ctx, groupID, assetID, version, ".terms")
	if err != nil {
		return types.StringValue("")
	}
	return types.StringValue(content)
}
