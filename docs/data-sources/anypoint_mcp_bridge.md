---
page_title: "anypoint_mcp_bridge Data Source - terraform-provider-anypoint"
subcategory: "Agents Tools"
description: |-
  Fetches the full details of a single MCP bridge by ID, including its source APIs and reconstructed tools.
---

# anypoint_mcp_bridge (Data Source)

Fetches the full details of a single MCP bridge by ID. In addition to the core instance
fields, it reconstructs the bridge's `source_apis` — each source REST API and the MCP
tools exposed for it — from the live transcoding policies (the same reconstruction the
`anypoint_mcp_bridge` resource performs on import).

## Example Usage

```terraform
data "anypoint_mcp_bridge" "one" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  id              = "21058094"
}

output "bridge_source_tools" {
  value = [
    for s in data.anypoint_mcp_bridge.one.source_apis : {
      label = s.label
      tools = [for t in s.tools : t.name]
    }
  ]
}
```

## Schema

### Required

- `id` (String) The numeric identifier of the MCP bridge (API Manager instance ID).
- `environment_id` (String) The environment ID where the MCP bridge is deployed.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `gateway_id` (String) The Flex Gateway ID the bridge is deployed to.
- `port` (Number) The listener port derived from the bridge proxy URI.
- `base_path` (String) The base path derived from the bridge proxy URI.
- `asset_id` (String) The generated Exchange asset ID backing the bridge.
- `asset_version` (String) The generated Exchange asset version.
- `product_version` (String) The product version.
- `group_id` (String) The Exchange group (organization) ID.
- `technology` (String) The gateway technology (`flexGateway` for MCP bridges).
- `instance_label` (String) The label of the MCP bridge.
- `status` (String) The current status of the MCP bridge.
- `consumer_endpoint` (String) The consumer-facing MCP endpoint URI (UI: "Consumer Endpoint"). Populated from the platform's `endpointUri`; may be null for self-managed (flexGateway) bridges — use `proxy_uri` instead.
- `proxy_uri` (String) The gateway proxy URI where the bridge listens (`http://0.0.0.0:<port>/<base_path>`), reconstructed from the instance endpoint.
- `deployment` (Object) Deployment target configuration. See [`deployment`](#nestedschema--deployment) below.
- `source_apis` (List of Object) The source REST APIs bridged by this MCP server, with their reconstructed tools. See [`source_apis`](#nestedschema--source_apis) below.

<a id="nestedschema--deployment"></a>
### Nested Schema for `deployment`

Read-Only:

- `environment_id` (String) The environment ID for deployment.
- `type` (String) Deployment type (e.g. `HY`).
- `expected_status` (String) Expected deployment status.
- `target_id` (String) The target gateway ID.
- `target_name` (String) The target gateway name.
- `gateway_version` (String) The gateway runtime version.

<a id="nestedschema--source_apis"></a>
### Nested Schema for `source_apis`

Read-Only:

- `label` (String) The source API label (`X-UPSTREAM-NAME`).
- `upstream_uri` (String) The backend URI of the source REST API.
- `asset_id` (String) The source REST API's Exchange asset ID.
- `group_id` (String) The source REST API's Exchange group ID.
- `version` (String) The source REST API's Exchange asset version.
- `tools` (List of Object) The MCP tools exposed for this source API. See [`tools`](#nestedschema--source_apis--tools) below.

<a id="nestedschema--source_apis--tools"></a>
### Nested Schema for `source_apis.tools`

Read-Only:

- `name` (String) The tool name.
- `description` (String) The tool description (null; the description lives in the generated asset metadata).
- `method` (String) The HTTP method the tool maps to.
- `path` (String) The REST path the tool maps to.
- `query_params` (List of String) Query parameter names passed through.
- `header_params` (List of String) Header parameter names passed through.
- `has_body` (Boolean) Whether the tool sends a request body.
