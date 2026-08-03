---
page_title: "anypoint_mcp_bridges Data Source - terraform-provider-anypoint"
subcategory: "Agents Tools"
description: |-
  Lists all MCP bridges registered in API Manager for the given environment.
---

# anypoint_mcp_bridges (Data Source)

Lists all MCP bridges (API Manager instances tagged `metadata.generatedBy=mcp_bridge`)
in the given environment. Plain MCP servers and other APIs are filtered out — only
bridges are returned.

## Example Usage

```terraform
data "anypoint_mcp_bridges" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "mcp_bridge_ids" {
  value = [for b in data.anypoint_mcp_bridges.all.bridges : b.id]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID to list MCP bridges from.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>`.
- `bridges` (List of Object) List of MCP bridges. See [`bridges`](#nestedschema--bridges) below.

<a id="nestedschema--bridges"></a>
### Nested Schema for `bridges`

Read-Only:

- `id` (String) The numeric ID of the MCP bridge (API Manager instance ID).
- `asset_id` (String) The generated Exchange asset ID backing the bridge.
- `asset_version` (String) The generated Exchange asset version.
- `product_version` (String) The product version.
- `group_id` (String) The Exchange group (organization) ID.
- `technology` (String) The gateway technology (`flexGateway` for MCP bridges).
- `instance_label` (String) The label of the MCP bridge.
- `status` (String) The current status of the MCP bridge.
- `endpoint_uri` (String) The consumer-facing endpoint URI (`endpointUri`); may be null for self-managed (flexGateway) bridges — use `proxy_uri` instead.
- `proxy_uri` (String) The gateway proxy URI where the bridge listens (`http://0.0.0.0:<port>/<base_path>`). The list endpoint omits it, so it is fetched per bridge; null if the per-bridge fetch fails.
