---
page_title: "anypoint_agent_instance Data Source - terraform-provider-anypoint"
subcategory: "Agents Tools"
description: |-
  Fetches the full details of a single agent instance by ID.
---

# anypoint_agent_instance (Data Source)

Fetches the full details of a single agent instance by ID, including its Exchange asset specification, endpoint, deployment target, and routing rules.

## Example Usage

```terraform
data "anypoint_agent_instance" "example" {
  id              = var.agent_instance_id
  environment_id  = var.environment_id
  organization_id = var.organization_id
}

output "agent_instance_status" {
  value = data.anypoint_agent_instance.example.status
}

output "agent_instance_consumer_endpoint" {
  value = data.anypoint_agent_instance.example.consumer_endpoint
}
```

## Schema

### Required

- `id` (String) The numeric identifier of the agent instance (as a string).
- `environment_id` (String) The environment ID where the agent instance is registered.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `technology` (String) The gateway technology (e.g., `omniGateway`).
- `provider_id` (String) The identity provider ID for the agent.
- `instance_label` (String) A human-readable label for this agent instance.
- `approval_method` (String) Client approval method (e.g., `manual`).
- `status` (String) The current status of the agent instance.
- `asset_id` (String) The Exchange asset ID.
- `asset_version` (String) The Exchange asset version.
- `product_version` (String) The product version.
- `consumer_endpoint` (String) Consumer-facing endpoint URI (the public URL clients use to reach the agent).
- `spec` (Object) The Exchange asset specification backing this agent instance. See [`spec`](#nestedschema--spec) below.
- `endpoint` (Object) Endpoint / proxy configuration for the agent instance. See [`endpoint`](#nestedschema--endpoint) below.
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

- `deployment_type` (String) Deployment type (`HY`, `CH`, `RF`).
- `type` (String) Endpoint protocol type (e.g., `a2a` for Agent-to-Agent).
- `base_path` (String) Agent base path for Omni Gateway (e.g., `my-agent`).
- `uri` (String) Direct implementation URI.
- `response_timeout` (Number) Response timeout in milliseconds.

<a id="nestedschema--deployment"></a>
### Nested Schema for `deployment`

Read-Only:

- `environment_id` (String) The environment ID for deployment.
- `type` (String) Deployment type (`HY`, `CH`, `RF`).
- `expected_status` (String) Expected deployment status (`deployed`, `undeployed`).
- `overwrite` (Boolean) Whether to overwrite an existing deployment.
- `target_id` (String) The target gateway ID.
- `target_name` (String) The target gateway name.
- `gateway_version` (String) The Omni Gateway runtime version.

<a id="nestedschema--routing"></a>
### Nested Schema for `routing`

Read-Only:

- `label` (String) A label for this route.
- `rules` (Object) Match conditions for this route. See [`rules`](#nestedschema--routing--rules) below.
- `upstreams` (List of Object) Weighted upstream backends for this route. See [`upstreams`](#nestedschema--routing--upstreams) below.

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
