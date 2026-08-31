---
page_title: "anypoint_mcp_tools Data Source - terraform-provider-anypoint"
subcategory: "Agents Tools"
description: |-
  Parses a source REST API's Exchange spec (OpenAPI/Swagger or RAML) into a normalized MCP tool list.
---

# anypoint_mcp_tools (Data Source)

Parses a source REST API's Exchange spec (OpenAPI/Swagger or RAML) into a normalized MCP tool list, one tool per REST operation. Feed the `tools` output directly into an [`anypoint_mcp_bridge`](../resources/anypoint_mcp_bridge.md) source API to auto-derive tools instead of declaring each one by hand.

This is the read-only "DS-hybrid" companion to `anypoint_mcp_bridge`: the risky spec parsing lives in a data source, so a spec that cannot be parsed fails `plan` cleanly instead of half-building a bridge. RAML parsing is best-effort; if it fails, declare tools explicitly on the resource.

The data source downloads the asset's best available spec file (preferring `fat-oas`, then `oas`, `fat-raml`, `raml`), unzips it if needed, and parses every operation. Output is sorted deterministically by path then method, so re-reads never cause plan churn. Path parameters (`{...}`) are omitted from `query_params`/`header_params` because the bridge derives URI params from the path automatically.

## Example Usage

```terraform
# Parse a source REST API's spec into tools...
data "anypoint_mcp_tools" "petstore" {
  organization_id = var.organization_id
  asset_id        = "petstore-api"
  version         = "1.0.0"

  # Optional filters
  exclude_methods    = ["DELETE"]
  exclude_tool_names = ["deprecatedOp"]
}

# ...and feed them straight into a bridge source API.
resource "anypoint_mcp_bridge" "b" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id
  mcp_asset_name  = "petstore-bridge"

  source_apis = [
    {
      label        = "petstore"
      upstream_uri = "https://backend.example.com/petstore/v1"
      asset_id     = "petstore-api"
      version      = "1.0.0"
      tools        = data.anypoint_mcp_tools.petstore.tools
    },
  ]
}

output "parsed_tool_count" {
  value = length(data.anypoint_mcp_tools.petstore.tools)
}
```

## Schema

### Required

- `asset_id` (String) The Exchange asset ID of the source REST API to parse (e.g., `petstore-api`).
- `version` (String) The Exchange asset version of the source REST API (e.g., `1.0.0`).

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.
- `group_id` (String) The Exchange group ID that owns the asset. Defaults to `organization_id`.
- `exclude_tool_names` (List of String) Tool names (operationIds or derived names) to omit from the output.
- `exclude_methods` (List of String) HTTP methods to omit entirely (e.g., `["DELETE"]`). Case-insensitive.

### Read-Only

- `id` (String) Composite identifier `group_id/asset_id/version`.
- `spec_type` (String) The detected spec format: `oas3`, `oas2`, or `raml`.
- `tools` (List of Object) The parsed tools, one per REST operation, sorted by path then method. Its object shape matches `anypoint_mcp_bridge` `source_apis[].tools` for direct assignment. See [`tools`](#nestedschema--tools) below.

<a id="nestedschema--tools"></a>
### Nested Schema for `tools`

Read-Only:

- `name` (String) The tool name (the spec `operationId` when present, else null so the bridge derives `<method>_<slug(path)>`).
- `description` (String) The operation summary/description, or null.
- `method` (String) The HTTP method (upper-case).
- `path` (String) The operation path, e.g., `/pets/{petId}`.
- `query_params` (List of String) Query parameter names exposed as tool inputs.
- `header_params` (List of String) Header parameter names exposed as tool inputs.
- `has_body` (Boolean) Whether the operation takes a request body.
