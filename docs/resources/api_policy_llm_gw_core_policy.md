---
page_title: "anypoint_api_policy_llm_gw_core_policy Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a LLM Gateway Core Policy policy on an Anypoint API instance.
---

# anypoint_api_policy_llm_gw_core_policy (Resource)

Manages a LLM Gateway Core Policy policy on an Anypoint API instance.

## Example Usage

```terraform
resource "anypoint_api_policy_llm_gw_core_policy" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    header_name           = "X-LLM-Vendor"
    vendor_header_mapping = [
      {
        vendor       = "openai"
        header_value = "openai"
      }
    ]
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
- `order` (Number) The order of policy execution.
- `asset_version` (String) The policy asset version. Defaults to `1.0.0-20251230075635`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.

### Read-Only

- `id` (String) The policy ID.
- `group_id` (String) The policy group ID.
- `asset_id` (String) The policy asset ID.
- `upstream_ids` (List of String) The upstream IDs this policy applies to.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `header_name` (String) Name of the header used for vendor routing.
- `vendor_header_mapping` (Dynamic) Array mapping vendor names to header values.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_api_policy_llm_gw_core_policy.example {organization_id}/{environment_id}/{api_instance_id}/{policy_id}
```
