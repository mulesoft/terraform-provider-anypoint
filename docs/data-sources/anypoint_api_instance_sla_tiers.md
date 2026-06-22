---
page_title: "anypoint_api_instance_sla_tiers Data Source - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Lists all SLA tiers for an API instance in API Manager.
---

# anypoint_api_instance_sla_tiers (Data Source)

Lists all SLA tiers configured for an API instance in API Manager.

## Example Usage

```terraform
data "anypoint_api_instance_sla_tiers" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "tier_names" {
  value = [for t in data.anypoint_api_instance_sla_tiers.all.tiers : t.name]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID where the API instance exists.
- `api_instance_id` (String) The numeric ID of the API instance.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>/<api_instance_id>`.
- `tiers` (List of Object) List of SLA tiers for this API instance. See [`tiers`](#nestedschema--tiers) below.

<a id="nestedschema--tiers"></a>
### Nested Schema for `tiers`

Read-Only:

- `id` (String) The numeric ID of the SLA tier.
- `name` (String) The name of the SLA tier.
- `description` (String) The description of the SLA tier.
- `auto_approve` (Boolean) Whether requests for this tier are auto-approved.
- `status` (String) The status of the SLA tier.
- `limits` (List of Object) List of SLA limits for this tier. See [`limits`](#nestedschema--tiers--limits) below.

<a id="nestedschema--tiers--limits"></a>
### Nested Schema for `tiers.limits`

Read-Only:

- `time_period_in_milliseconds` (Number) The time period for the limit in milliseconds.
- `maximum_requests` (Number) The maximum number of requests allowed in the time period.
- `visible` (Boolean) Whether this limit is visible to API consumers.
