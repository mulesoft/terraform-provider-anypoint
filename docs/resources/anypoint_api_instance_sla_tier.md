---
page_title: "anypoint_api_instance_sla_tier Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages an SLA tier for an API instance in Anypoint API Manager.
---

# anypoint_api_instance_sla_tier (Resource)

Manages an SLA tier for an API instance in Anypoint API Manager.

-> **Authentication:** This resource calls **Gateway Manager / API Manager control-plane APIs** (the gateway lifecycle and/or the `gateway_id` pre-flight). A `client_credentials` Connected App works — grant it **Manage Servers**, **Read Servers**, and **View Organization** (plus API Manager scopes such as **Manage APIs Configuration**, **Manage Policies**, **Deploy API Proxies**, and **Exchange Viewer** for Omni Gateway operations). A Connected App missing these scopes is rejected with `HTTP 401`/`403` before anything is created; the fix is to add the scopes (or use `auth_type = "user"` with a user that has the equivalent permissions). See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

## Example Usage

```terraform
resource "anypoint_api_instance_sla_tier" "gold" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Gold"
  description = "Gold tier with high volume limits for premium customers"
  auto_approve = true
  status       = "ACTIVE"

  limits = [
    {
      time_period_in_milliseconds = 60000
      maximum_requests            = 1000
      visible                     = true
    },
    {
      time_period_in_milliseconds = 3600000
      maximum_requests            = 50000
      visible                     = true
    }
  ]
}
```

## Schema

### Required

- `environment_id` (String) Environment ID where the API instance lives.
- `api_instance_id` (String) Numeric ID of the API instance.
- `name` (String) Name of the SLA tier.
- `limits` (Block List) Rate limits for this SLA tier. See [below for nested schema](#nestedschema--limits).

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `description` (String) Description of the SLA tier.
- `auto_approve` (Boolean) Whether requests for this SLA tier are auto-approved. Defaults to `false`.
- `status` (String) Status of the SLA tier. Valid values: `ACTIVE`, `INACTIVE`.

### Read-Only

- `id` (String) Unique identifier of the SLA tier.

<a id="nestedschema--limits"></a>
### Nested Schema for `limits`

Required:

- `time_period_in_milliseconds` (Number) Time period for the rate limit in milliseconds.
- `maximum_requests` (Number) Maximum number of requests allowed in the time period.

Optional:

- `visible` (Boolean) Whether this limit is visible to API consumers. Defaults to `true`.

## Import

An existing SLA tier can be imported using its composite ID: `organization_id/environment_id/api_instance_id/tier_name_or_tier_id`.

The last segment accepts either the **tier name** (as shown in the Anypoint UI, e.g. `Gold`) or the numeric tier ID. Using the name is recommended — it is visible without API calls.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_instance_sla_tier.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<tier_name>"
}

resource "anypoint_api_instance_sla_tier" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  api_instance_id = "<api_instance_id>"
  name            = "<tier_name>"
  limits = [
    {
      time_period_in_milliseconds = 60000
      maximum_requests            = 100
    }
  ]
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

### Using the CLI (deprecated, Terraform < 1.5)

```shell
terraform import anypoint_api_instance_sla_tier.imported <organization_id>/<environment_id>/<api_instance_id>/<tier_name>
```
