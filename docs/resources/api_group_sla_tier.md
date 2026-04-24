---
page_title: "anypoint_api_group_sla_tier Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages an SLA tier for an API Group instance in Anypoint API Manager. The group_instance_id is the numeric ID of the group instance (visible in the groupInstances URL path in API Manager).
---

# anypoint_api_group_sla_tier (Resource)

Manages an SLA tier for an API Group instance in Anypoint API Manager. The `group_instance_id` is the numeric ID of the group instance (visible in the groupInstances URL path in API Manager).

## Example Usage

```terraform
resource "anypoint_api_group_sla_tier" "bronze" {
  organization_id   = var.organization_id
  environment_id    = var.environment_id
  group_instance_id = var.group_instance_id

  name        = "Bronze"
  description = "Standard tier for regular consumers"
  status      = "ACTIVE"
  auto_approve = true

  default_limits = [
    {
      maximum_requests            = 100
      time_period_in_milliseconds = 60000
      visible                     = true
    },
    {
      maximum_requests            = 5000
      time_period_in_milliseconds = 3600000
      visible                     = true
    }
  ]
}
```

## Schema

### Required

- `environment_id` (String) Environment ID where the API Group instance lives.
- `group_instance_id` (String) Numeric ID of the API Group instance to attach this SLA tier to.
- `name` (String) Name of the SLA tier.
- `default_limits` (Block List) Rate limits for this SLA tier. Maps to the 'defaultLimits' field in the API. See [below for nested schema](#nestedschema--default_limits).

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `description` (String) Optional description of the SLA tier.
- `auto_approve` (Boolean) Whether subscription requests for this tier are auto-approved. Defaults to `false`.
- `status` (String) Status of the SLA tier. Valid values: `ACTIVE`, `INACTIVE`.

### Read-Only

- `id` (String) Unique identifier of the SLA tier (assigned by the platform).

<a id="nestedschema--default_limits"></a>
### Nested Schema for `default_limits`

Required:

- `time_period_in_milliseconds` (Number) Duration of the rate-limit window in milliseconds.
- `maximum_requests` (Number) Maximum number of requests allowed within the window.

Optional:

- `visible` (Boolean) Whether this limit is visible to API consumers. Defaults to `true`.

## Import

Import is supported using the following format:

```shell
terraform import anypoint_api_group_sla_tier.example organization_id/environment_id/group_instance_id/tier_id
```
