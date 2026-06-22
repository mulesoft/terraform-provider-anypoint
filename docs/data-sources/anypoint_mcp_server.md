---
page_title: "anypoint_mcp_server Data Source - terraform-provider-anypoint"
subcategory: "Agents Tools"
description: |-
  Fetches the full details of a single MCP server by ID.
---

# anypoint_mcp_server (Data Source)

Fetches the full details of a single MCP (Model Context Protocol) server by ID, including its Exchange asset specification, endpoint, deployment target, and routing rules.

## Example Usage

```terraform
data "anypoint_mcp_server" "example" {
  id              = var.mcp_server_id
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "mcp_server_status" {
  value = data.anypoint_mcp_server.example.status
}

output "mcp_server_consumer_endpoint" {
  value = data.anypoint_mcp_server.example.consumer_endpoint
}

output "mcp_server_upstream_id" {
  value = data.anypoint_mcp_server.example.upstream_id
}
```

## Schema

### Required

- `id` (String) The numeric identifier of the MCP server.
- `environment_id` (String) The environment ID where the MCP server is deployed.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `technology` (String) The gateway technology (typically `omniGateway` for MCP servers).
- `provider_id` (String) The identity provider ID for the MCP server.
- `instance_label` (String) A human-readable label for this MCP server.
- `approval_method` (String) Client approval method (e.g., `manual`, or null if no approval required).
- `status` (String) The current status of the MCP server.
- `asset_id` (String) The Exchange asset ID backing this MCP server.
- `asset_version` (String) The Exchange asset version.
- `product_version` (String) The product version.
- `consumer_endpoint` (String) Consumer-facing endpoint URI (the public URL clients use to reach the MCP server).
- `upstream_id` (String) The server-assigned upstream ID for the first upstream backend.
- `spec` (Object) The Exchange asset specification backing this MCP server. See [`spec`](#nestedschema--spec) below.
- `endpoint` (Object) Endpoint / proxy configuration for the MCP server. See [`endpoint`](#nestedschema--endpoint) below.
- `deployment` (Object) Deployment target configuration. See [`deployment`](#nestedschema--deployment) below.
- `routing` (List of Object) Routing rules with weighted upstream backends. See [`routing`](#nestedschema--routing) below.

<a id="nestedschema--spec"></a>
### Nested Schema for `spec`

Read-Only:

- `asset_id` (String) The Exchange asset ID.
- `group_id` (String) The Exchange group (organization) ID.
- `version` (String) The asset version.

<a id="nestedschema--endpoint"></a>
### Nested Schema for `endpoint`

Read-Only:

- `deployment_type` (String) Deployment type (`HY` for hybrid, `CH` for CloudHub, `RF` for Runtime Fabric).
- `type` (String) Endpoint protocol type (for MCP servers, this is `mcp`).
- `base_path` (String) MCP server base path for Omni Gateway (e.g., `my-mcp-server`).
- `uri` (String) Direct implementation URI (if configured instead of `base_path`).
- `response_timeout` (Number) Response timeout in milliseconds.

<a id="nestedschema--deployment"></a>
### Nested Schema for `deployment`

Read-Only:

- `environment_id` (String) The environment ID for deployment.
- `type` (String) Deployment type (`HY`, `CH`, `RF`).
- `expected_status` (String) Expected deployment status (`deployed`, `undeployed`).
- `overwrite` (Boolean) Whether to overwrite an existing deployment.
- `target_id` (String) The target gateway ID to deploy to.
- `target_name` (String) The target gateway name.
- `gateway_version` (String) The Omni Gateway runtime version.

<a id="nestedschema--routing"></a>
### Nested Schema for `routing`

Read-Only:

- `label` (String) A label for this route.
- `rules` (Object) Match conditions for this route. See [`rules`](#nestedschema--routing--rules) below.
- `upstreams` (List of Object) Weighted upstream backends for this route (actual MCP server implementation endpoints). See [`upstreams`](#nestedschema--routing--upstreams) below.

<a id="nestedschema--routing--rules"></a>
### Nested Schema for `routing.rules`

Read-Only:

- `methods` (String) Pipe-separated HTTP methods (e.g., `GET`, `POST|PUT`).
- `path` (String) URL path pattern to match (e.g., `/api/*`).
- `host` (String) Host header value to match.
- `headers` (Map of String) Header key-value pairs to match.

<a id="nestedschema--routing--upstreams"></a>
### Nested Schema for `routing.upstreams`

Read-Only:

- `weight` (Number) Traffic weight percentage (0-100).
- `uri` (String) The upstream backend URI.
- `label` (String) A label for this upstream.
- `tls_context_id` (String) TLS context for upstream connections (format: `secretGroupId/tlsContextId`).
