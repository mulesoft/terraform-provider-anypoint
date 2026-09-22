---
page_title: "anypoint_api_policy_graphql_static_query_complexity Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the GraphQL Static Query Complexity policy applied to an Anypoint API instance.
---

# anypoint_api_policy_graphql_static_query_complexity (Data Source)

Reads the GraphQL Static Query Complexity policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_graphql_static_query_complexity" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "complexity_config" {
  value = {
    maximum_complexity       = data.anypoint_api_policy_graphql_static_query_complexity.example.configuration.maximum_complexity
    default_field_cost       = data.anypoint_api_policy_graphql_static_query_complexity.example.configuration.default_field_cost
    block_operation          = data.anypoint_api_policy_graphql_static_query_complexity.example.configuration.block_operation
    reject_unbounded_lists   = data.anypoint_api_policy_graphql_static_query_complexity.example.configuration.reject_unbounded_lists
  }
}

# Or look up by explicit policy_id
data "anypoint_api_policy_graphql_static_query_complexity" "by_id" {
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
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.
- `pointcut_data` (String) Pointcut definition as a JSON string.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Read-Only:

- `maximum_complexity` (Number) Maximum allowed complexity score for a GraphQL operation.
- `default_field_cost` (Number) Default cost assigned to each field when no custom cost is specified.
- `block_operation` (Boolean) Whether to block operations that exceed the maximum complexity.
- `reject_unbounded_lists` (Boolean) Whether to reject queries with unbounded list selections.
- `directive_name` (String) Name of the GraphQL directive used to specify custom field costs in the schema.
- `value_argument` (String) Name of the directive argument that specifies the field cost value.
- `multipliers_argument` (String) Name of the directive argument that specifies complexity multipliers.
