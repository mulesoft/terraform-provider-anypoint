---
page_title: "anypoint_api_instance_sla_tier Data Source - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Fetches a single SLA tier for an API instance in API Manager.
---

# anypoint_api_instance_sla_tier (Data Source)

Fetches a single SLA tier for an API instance in API Manager, including its rate-limit configuration.

## Example Usage

```terraform
data "anypoint_api_instance_sla_tier" "gold" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  tier_id         = var.tier_id
}

output "tier_name" {
  value = data.anypoint_api_instance_sla_tier.gold.name
}

output "tier_limits" {
  value = data.anypoint_api_instance_sla_tier.gold.limits
}
```

## Schema

### Required

- `environment_id` (String) The environment ID where the API instance exists.
- `api_instance_id` (String) The numeric ID of the API instance.
- `tier_id` (String) The ID of the SLA tier to look up.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>/<api_instance_id>/<tier_id>`.
- `name` (String) The name of the SLA tier.
- `description` (String) The description of the SLA tier.
- `auto_approve` (Boolean) Whether requests for this tier are auto-approved.
- `status` (String) The status of the SLA tier.
- `limits` (List of Object) List of SLA limits for this tier. See [`limits`](#nestedschema--limits) below.

<a id="nestedschema--limits"></a>
### Nested Schema for `limits`

Read-Only:

- `time_period_in_milliseconds` (Number) The time period for the limit in milliseconds.
- `maximum_requests` (Number) The maximum number of requests allowed in the time period.
- `visible` (Boolean) Whether this limit is visible to API consumers.
