---
page_title: "anypoint_api_policy_rate_limiting Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a Rate Limiting policy on an Anypoint API instance.
---

# anypoint_api_policy_rate_limiting (Resource)

Manages a Rate Limiting policy on an Anypoint API instance.

## Example Usage

```terraform
resource "anypoint_api_policy_rate_limiting" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    rate_limits = [
      {
        maximum_requests            = 100
        time_period_in_milliseconds = 60000
      }
    ]
    expose_headers = false
    clusterizable  = true
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
- `asset_version` (String) The policy asset version. Defaults to `1.4.1`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.

### Read-Only

- `id` (String) The policy ID.
- `group_id` (String) The policy group ID.
- `asset_id` (String) The policy asset ID.
- `upstream_ids` (List of String) The upstream IDs this policy applies to.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `rate_limits` (Dynamic) Array of rate limit rules with `maximum_requests` and `time_period_in_milliseconds`.

Optional:

- `clusterizable` (Boolean) Whether the rate limit counters are shared across a cluster.
- `expose_headers` (Boolean) Whether to expose rate-limit headers in the response.
- `key_selector` (String) Expression to extract the rate limit key from the request.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_api_policy_rate_limiting.example {organization_id}/{environment_id}/{api_instance_id}/{policy_id}
```
