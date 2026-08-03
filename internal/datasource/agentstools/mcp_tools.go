package agentstools

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

var (
	_ datasource.DataSource              = &MCPToolsDataSource{}
	_ datasource.DataSourceWithConfigure = &MCPToolsDataSource{}
)

// mcpToolAttrTypes MUST match the anypoint_mcp_bridge resource's `tools` block object
// type exactly, so the output can be assigned directly:
//
//	source_apis = [{ ... tools = data.anypoint_mcp_tools.x.tools }]
var mcpToolAttrTypes = map[string]attr.Type{
	"name":          types.StringType,
	"description":   types.StringType,
	"method":        types.StringType,
	"path":          types.StringType,
	"query_params":  types.ListType{ElemType: types.StringType},
	"header_params": types.ListType{ElemType: types.StringType},
	"has_body":      types.BoolType,
}

// MCPToolsDataSource parses a source REST API's Exchange spec into MCP tools
// (Approach D / DS-hybrid). Read-only: a parser failure fails `plan` harmlessly
// instead of half-building a bridge.
type MCPToolsDataSource struct {
	client *agentstools.MCPToolsClient
}

type MCPToolsDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	GroupID          types.String `tfsdk:"group_id"`
	AssetID          types.String `tfsdk:"asset_id"`
	Version          types.String `tfsdk:"version"`
	ExcludeToolNames types.List   `tfsdk:"exclude_tool_names"`
	ExcludeMethods   types.List   `tfsdk:"exclude_methods"`
	SpecType         types.String `tfsdk:"spec_type"`
	Tools            types.List   `tfsdk:"tools"`
}

func NewMCPToolsDataSource() datasource.DataSource {
	return &MCPToolsDataSource{}
}

func (d *MCPToolsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_tools"
}

func (d *MCPToolsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Parses a source REST API's Exchange spec (OpenAPI/Swagger or RAML) into a " +
			"normalized MCP tool list. Feed the `tools` output straight into an " +
			"anypoint_mcp_bridge source API to auto-derive tools instead of declaring each one " +
			"by hand:\n\n" +
			"    source_apis = [{ ... tools = data.anypoint_mcp_tools.example.tools }]\n\n" +
			"This is read-only: a spec that cannot be parsed fails `plan` cleanly (no half-built " +
			"bridge). RAML parsing is best-effort; if it fails, declare tools explicitly.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier group_id/asset_id/version.",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. Defaults to the provider credentials organization.",
			},
			"group_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Exchange group ID that owns the asset. Defaults to organization_id.",
			},
			"asset_id": schema.StringAttribute{
				Required:    true,
				Description: "The Exchange asset ID of the source REST API to parse (e.g. \"petstore-api\").",
			},
			"version": schema.StringAttribute{
				Required:    true,
				Description: "The Exchange asset version of the source REST API (e.g. \"1.0.0\").",
			},
			"exclude_tool_names": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Tool names (operationIds or derived names) to omit from the output.",
			},
			"exclude_methods": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "HTTP methods to omit entirely (e.g. [\"DELETE\"]). Case-insensitive.",
			},
			"spec_type": schema.StringAttribute{
				Computed:    true,
				Description: "The detected spec format: oas3, oas2, or raml.",
			},
			"tools": schema.ListNestedAttribute{
				Computed: true,
				Description: "The parsed tools, one per REST operation, sorted by path then method. " +
					"Shape matches anypoint_mcp_bridge source_apis[].tools for direct assignment.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The tool name (the spec operationId when present, else null so the bridge derives <method>_<slug(path)>).",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The operation summary/description, or null.",
						},
						"method": schema.StringAttribute{
							Computed:    true,
							Description: "The HTTP method (upper-case).",
						},
						"path": schema.StringAttribute{
							Computed:    true,
							Description: "The operation path, e.g. /pets/{petId}.",
						},
						"query_params": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Query parameter names exposed as tool inputs.",
						},
						"header_params": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Header parameter names exposed as tool inputs.",
						},
						"has_body": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the operation takes a request body.",
						},
					},
				},
			},
		},
	}
}

func (d *MCPToolsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	toolsClient, err := agentstools.NewMCPToolsClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create MCP Tools Client", err.Error())
		return
	}
	d.client = toolsClient
}

func (d *MCPToolsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MCPToolsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	groupID := data.GroupID.ValueString()
	if groupID == "" {
		groupID = orgID
	}
	assetID := data.AssetID.ValueString()
	version := data.Version.ValueString()

	excludeNames := stringListToSet(ctx, data.ExcludeToolNames)
	excludeMethods := stringListToLowerSet(ctx, data.ExcludeMethods)

	parsed, specType, err := d.client.GetAssetTools(ctx, groupID, assetID, version)
	if err != nil {
		resp.Diagnostics.AddError("Unable to parse source API spec into tools",
			fmt.Sprintf("asset %s/%s/%s: %s", groupID, assetID, version, err.Error()))
		return
	}

	toolObjType := types.ObjectType{AttrTypes: mcpToolAttrTypes}
	elems := make([]attr.Value, 0, len(parsed))
	for _, t := range parsed {
		if excludeMethods[toLower(t.Method)] {
			continue
		}
		nameVal := types.StringNull()
		if t.Name != "" {
			nameVal = types.StringValue(t.Name)
		}
		if !nameVal.IsNull() && excludeNames[t.Name] {
			continue
		}
		descVal := types.StringNull()
		if t.Description != "" {
			descVal = types.StringValue(t.Description)
		}
		obj, diags := types.ObjectValue(mcpToolAttrTypes, map[string]attr.Value{
			"name":          nameVal,
			"description":   descVal,
			"method":        types.StringValue(t.Method),
			"path":          types.StringValue(t.Path),
			"query_params":  stringSliceToList(t.QueryParams),
			"header_params": stringSliceToList(t.HeaderParams),
			"has_body":      types.BoolValue(t.HasBody),
		})
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		elems = append(elems, obj)
	}

	toolsList, diags := types.ListValue(toolObjType, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.OrganizationID = types.StringValue(orgID)
	data.GroupID = types.StringValue(groupID)
	data.SpecType = types.StringValue(specType)
	data.Tools = toolsList
	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", groupID, assetID, version))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func stringSliceToList(vals []string) types.List {
	elems := make([]attr.Value, 0, len(vals))
	for _, v := range vals {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

func stringListToSet(ctx context.Context, l types.List) map[string]bool {
	out := map[string]bool{}
	if l.IsNull() || l.IsUnknown() {
		return out
	}
	var vals []string
	l.ElementsAs(ctx, &vals, false)
	for _, v := range vals {
		out[v] = true
	}
	return out
}

func stringListToLowerSet(ctx context.Context, l types.List) map[string]bool {
	out := map[string]bool{}
	for k := range stringListToSet(ctx, l) {
		out[toLower(k)] = true
	}
	return out
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
