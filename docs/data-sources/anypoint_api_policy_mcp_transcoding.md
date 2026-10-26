---
page_title: "anypoint_api_policy_mcp_transcoding Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the MCP Transcoding policy applied to an Anypoint API instance.
---

# anypoint_api_policy_mcp_transcoding (Data Source)

Reads the MCP Transcoding policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

This is the per-upstream **outbound** transcoding policy that maps MCP tool calls onto the HTTP
operations of one source API. An MCP bridge carries one of these per entry in its `source_apis`,
so an instance created by [`anypoint_mcp_bridge`](../resources/anypoint_mcp_bridge.md) with more
than one source API will have several. In that case pass an explicit `policy_id` — auto-discovery
by asset type returns only the first match.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_mcp_transcoding" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "mcp_transcoding_configuration" {
  value = data.anypoint_api_policy_mcp_transcoding.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_mcp_transcoding" "by_id" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  policy_id       = "12345"
}
```

## Schema

### Required

- `environment_id` (String) The environment ID where the API instance lives.
- `api_instance_id` (String) Numeric ID of the API instance this policy is applied to.

### Optional

- `organization_id` (String) Organization ID. Defaults to the provider's org ID if omitted.
- `policy_id` (String) The ID of the policy. If omitted, the data source looks up the policy by asset type on the API instance.

### Read-Only

- `id` (String) Unique identifier of the applied policy.
- `label` (String) A human-readable label for this policy instance.
- `configuration` (Attributes) Policy configuration with typed fields. See [`configuration`](#nestedschema--configuration) below.
- `order` (Number) Execution order of the policy.
- `disabled` (Boolean) Whether the policy is disabled.
- `policy_template_id` (String) Policy template ID assigned by the server.
- `asset_version` (String) Version of the policy asset.
- `upstream_ids` (List of String) The upstream IDs this policy applies to — for an MCP bridge, the upstream serving this policy's source API.
- `pointcut_data` (String) Pointcut definition as a JSON string.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Read-Only:

- `tools` (Dynamic) The full tool mapping for this upstream. Each entry describes one MCP tool and the HTTP operation it transcodes to.
