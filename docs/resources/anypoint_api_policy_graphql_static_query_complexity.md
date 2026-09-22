---
page_title: "anypoint_api_policy_graphql_static_query_complexity Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a GraphQL Static Query Complexity policy on an Anypoint API instance.
---

# anypoint_api_policy_graphql_static_query_complexity (Resource)

Manages a GraphQL Static Query Complexity policy on an Anypoint API instance. This policy calculates and enforces a static complexity score for GraphQL queries to prevent excessively complex or expensive operations.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_api_policy_graphql_static_query_complexity" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    maximum_complexity    = 100
    default_field_cost    = 1
    block_operation       = true
    reject_unbounded_lists = true
  }

  order = 1
}

# Advanced configuration with custom complexity directives
resource "anypoint_api_policy_graphql_static_query_complexity" "advanced" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    maximum_complexity       = 250
    default_field_cost       = 2
    block_operation          = true
    reject_unbounded_lists   = false
    directive_name           = "cost"
    value_argument           = "weight"
    multipliers_argument     = "multipliers"
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

Required:

- `maximum_complexity` (Number) Maximum allowed complexity score for a GraphQL operation. Operations exceeding this threshold will be rejected if `block_operation` is `true`.

Optional:

- `default_field_cost` (Number) Default cost assigned to each field when no custom cost is specified. Defaults to `1`.
- `block_operation` (Boolean) Whether to block operations that exceed the maximum complexity. Defaults to `true`.
- `reject_unbounded_lists` (Boolean) Whether to reject queries with unbounded list selections (selections without limits). Defaults to `true`.
- `directive_name` (String) Name of the GraphQL directive used to specify custom field costs in the schema (e.g. `@cost`).
- `value_argument` (String) Name of the directive argument that specifies the field cost value.
- `multipliers_argument` (String) Name of the directive argument that specifies complexity multipliers.


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

An existing `anypoint_api_policy_graphql_static_query_complexity` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_graphql_static_query_complexity.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_graphql_static_query_complexity" "imported" {
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
terraform import anypoint_api_policy_graphql_static_query_complexity.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
