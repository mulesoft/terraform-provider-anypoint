---
page_title: "anypoint_api_policy_graphql_introspection_control Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a GraphQL Introspection Control policy on an Anypoint API instance.
---

# anypoint_api_policy_graphql_introspection_control (Resource)

Manages a GraphQL Introspection Control policy on an Anypoint API instance. This policy controls which GraphQL introspection queries are allowed, enabling you to block schema introspection, type introspection, or typename queries.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
# Block all introspection queries
resource "anypoint_api_policy_graphql_introspection_control" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    block_schema   = true
    block_type     = true
    block_typename = true
  }

  order = 1
}

# Allow introspection (default behavior)
resource "anypoint_api_policy_graphql_introspection_control" "allow" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    block_schema   = false
    block_type     = false
    block_typename = false
  }

  order = 1
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
- `order` (Number) The order of policy execution.
- `asset_version` (String) The policy asset version. Defaults to `1.0.0`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.
- `pointcut_data` (String) Pointcut definition as a JSON string. Restricts the policy to specific resources (methods and/or URIs). When null the policy applies to all resources. Use `jsonencode()` to set this. See [Pointcut Data](#pointcut-data) below.

### Read-Only

- `id` (String) The policy ID.
- `policy_template_id` (String) The policy template ID assigned by the server.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Optional:

- `block_schema` (Boolean) Whether to block `__schema` introspection queries. Defaults to `false`.
- `block_type` (Boolean) Whether to block `__type` introspection queries. Defaults to `false`.
- `block_typename` (Boolean) Whether to block `__typename` queries. Defaults to `false`.


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

# Multiple conditions (logical OR — policy applies if any condition matches)
pointcut_data = jsonencode([
  {
    methodRegex      = "GET"
    uriTemplateRegex = "/api/v1/read/.*"
  },
  {
    methodRegex      = "POST|PUT"
    uriTemplateRegex = "/api/v1/write/.*"
  }
])
```

## Import

An existing `anypoint_api_policy_graphql_introspection_control` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_graphql_introspection_control.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_graphql_introspection_control" "imported" {
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
terraform import anypoint_api_policy_graphql_introspection_control.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
