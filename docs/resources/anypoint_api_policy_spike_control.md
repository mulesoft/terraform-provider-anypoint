---
page_title: "anypoint_api_policy_spike_control Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a Spike Control policy on an Anypoint API instance.
---

# anypoint_api_policy_spike_control (Resource)

Manages a Spike Control policy on an Anypoint API instance.

## Example Usage

```terraform
resource "anypoint_api_policy_spike_control" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    maximum_requests            = 100
    time_period_in_milliseconds = 1000
    delay_time_in_millis        = 500
    delay_attempts              = 3
    queuing_limit               = 5
    expose_headers              = false
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
- `asset_version` (String) The policy asset version. Defaults to `1.2.2`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.

### Read-Only

- `id` (String) The policy ID.
- `group_id` (String) The policy group ID.
- `asset_id` (String) The policy asset ID.
- `upstream_ids` (List of String) The upstream IDs this policy applies to.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `delay_attempts` (Number) The number of attempts to retry before rejecting.
- `delay_time_in_millis` (Number) The delay time in milliseconds before retrying queued requests.
- `maximum_requests` (Number) Maximum number of requests allowed in the time period.
- `time_period_in_milliseconds` (Number) The time period in milliseconds for the spike control window. Must be <= 5000.

Optional:

- `expose_headers` (Boolean) Whether to expose rate-limit headers in the response.
- `queuing_limit` (Number) Maximum number of requests that can be queued.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_api_policy_spike_control.example {organization_id}/{environment_id}/{api_instance_id}/{policy_id}
```
