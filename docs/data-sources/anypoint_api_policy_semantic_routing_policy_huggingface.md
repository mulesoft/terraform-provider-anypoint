---
page_title: "anypoint_api_policy_semantic_routing_policy_huggingface Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the Semantic Routing (HuggingFace) policy applied to an Anypoint API instance.
---

# anypoint_api_policy_semantic_routing_policy_huggingface (Data Source)

Reads the Semantic Routing (HuggingFace) policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_semantic_routing_policy_huggingface" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "semantic_routing_policy_huggingface_configuration" {
  value = data.anypoint_api_policy_semantic_routing_policy_huggingface.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_semantic_routing_policy_huggingface" "by_id" {
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

- `huggingface_api_key` (String) API key for the HuggingFace service.
- `huggingface_url` (String) URL of the HuggingFace inference API.
- `routes` (Dynamic) Array of routing rules.
- `fallback_route` (Dynamic) Fallback route configuration when no semantic match is found.
- `threshold` (Dynamic) Threshold configuration object for similarity scoring.
- `timeout` (Number) Timeout value in milliseconds.
