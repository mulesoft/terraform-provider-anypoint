---
page_title: "anypoint_api_policy_header_injection Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the Header Injection policy applied to an Anypoint API instance.
---

# anypoint_api_policy_header_injection (Data Source)

Reads the Header Injection policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_header_injection" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "header_injection_configuration" {
  value = data.anypoint_api_policy_header_injection.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_header_injection" "by_id" {
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

- `inbound_headers` (Dynamic) Array of inbound headers to inject or remove.
- `outbound_headers` (Dynamic) Array of outbound headers to inject or remove.
