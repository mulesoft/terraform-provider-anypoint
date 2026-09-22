---
page_title: "anypoint_api_policy_mcp_transcoding Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a MCP Transcoding policy on an Anypoint API instance.
---

# anypoint_api_policy_mcp_transcoding (Resource)

Manages a MCP Transcoding policy on an Anypoint API instance.

This is the per-upstream **outbound** transcoding policy that maps MCP tool calls onto the
HTTP operations of one source API. It is the companion of
[`anypoint_api_policy_mcp_transcoding_router`](anypoint_api_policy_mcp_transcoding_router.md),
which is the single **inbound** policy that routes each tool call to the right upstream.

~> **Usually you do not need this resource.** [`anypoint_mcp_bridge`](anypoint_mcp_bridge.md)
attaches one `mcp-transcoding` policy per entry in its `source_apis` automatically, along with
the inbound `mcp-support`, `mcp-schema-validation` and `mcp-transcoding-router` policies. Manage
this resource directly only when you are assembling a bridge by hand; declaring it against an
instance that `anypoint_mcp_bridge` also manages will cause the two to fight over the same policy.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_api_policy_mcp_transcoding" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  # Attach the policy to the upstream that serves this source API.
  upstream_ids = [var.upstream_id]

  configuration = {
    tools = [
      {
        name        = "get_pet_by_id"
        description = "Fetch a single pet by its identifier"
        method      = "GET"
        path        = "/pets/{petId}"
        hasBody     = false
      }
    ]
  }
}
```

## Schema

### Required

- `environment_id` (String) The environment ID.
- `api_instance_id` (String) The API instance ID.
- `configuration` (Block) The policy configuration. See [Configuration](#nestedschema--configuration) below.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `label` (String) A human-readable label for this policy instance.
- `asset_version` (String) The policy asset version. Defaults to `1.0.0`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.
- `upstream_ids` (List of String) The upstream IDs this policy applies to. Because `mcp-transcoding` is an outbound policy, this is how the tool mapping is bound to a specific source API.
- `pointcut_data` (String) Pointcut definition as a JSON string. Restricts the policy to specific resources (methods and/or URIs). When null the policy applies to all resources. Use `jsonencode()` to set this. See [Pointcut Data](#pointcut-data) below.

### Read-Only

- `id` (String) The policy ID.
- `policy_template_id` (String) The policy template ID assigned by the server.
- `order` (Number) The order of policy execution. Assigned by the server — the outbound policy endpoint does not accept a caller-supplied order.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `tools` (Dynamic) The full tool mapping for this upstream. Each entry describes one MCP tool and the HTTP operation it transcodes to — typically `name`, `description`, `method`, `path`, and whether the operation takes a request body.

## Pointcut Data

The optional `pointcut_data` attribute restricts the policy to specific HTTP methods and/or URI patterns, matching what is configured under "Apply configurations to specific methods & resources" in the Anypoint Platform UI.

Each element in the array maps to one condition row in the UI:

- `methodRegex` — pipe-separated HTTP methods (e.g. `GET`, `GET|POST`). Omit or set to `.*` to match all methods.
- `uriTemplateRegex` — regex for the URI path (e.g. `/api/v1/.*`). Omit or set to `.*` to match all paths.

```hcl
# Apply policy to GET and POST requests on /api/v1/* only
pointcut_data = jsonencode([
  {
    methodRegex      = "GET|POST"
    uriTemplateRegex = "/api/v1/.*"
  }
])
```

## Import

An existing `anypoint_api_policy_mcp_transcoding` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_mcp_transcoding.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_mcp_transcoding" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  api_instance_id = "<api_instance_id>"
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

### Using the CLI (deprecated, Terraform < 1.5)

```shell
terraform import anypoint_api_policy_mcp_transcoding.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
