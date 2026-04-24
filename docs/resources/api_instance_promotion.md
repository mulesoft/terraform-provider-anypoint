---
page_title: "anypoint_api_instance_promotion Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Promotes an API instance from one environment to another. This copies the API definition (and optionally its policies, SLA tiers, and alerts) into the target environment.
---

# anypoint_api_instance_promotion (Resource)

Promotes an API instance from one environment to another. This copies the API definition (and optionally its policies, SLA tiers, and alerts) into the target environment.

## Example Usage

```terraform
resource "anypoint_api_instance_promotion" "example" {
  environment_id = "target-environment-id"
  source_api_id  = 12345

  instance_label   = "promoted-api"
  include_alerts   = true
  include_policies = true
  include_tiers    = true
}
```

## Schema

### Required

- `environment_id` (String) The target environment ID where the API instance will be promoted to.
- `source_api_id` (Number) The numeric ID of the source API instance to promote from.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `instance_label` (String) Optional label for the promoted API instance.
- `include_alerts` (Boolean) Whether to copy alerts from the source API instance. Defaults to `true`.
- `include_policies` (Boolean) Whether to copy policies from the source API instance. Defaults to `true`.
- `include_tiers` (Boolean) Whether to copy SLA tiers from the source API instance. Defaults to `true`.

### Read-Only

- `id` (String) The numeric ID of the promoted API instance in the target environment.
- `asset_id` (String) The Exchange asset ID of the promoted API instance.
- `asset_version` (String) The Exchange asset version of the promoted API instance.
- `product_version` (String) The product version of the promoted API instance.
- `group_id` (String) The Exchange group (organization) ID of the promoted API instance.
- `technology` (String) The gateway technology of the promoted API instance.
- `status` (String) The current status of the promoted API instance.
- `autodiscovery_instance_name` (String) The autodiscovery instance name of the promoted API instance.
