---
page_title: "anypoint_api_policy_spike_control Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a Spike Control policy on an Anypoint API instance.
---

# anypoint_api_policy_spike_control (Resource)

Manages a Spike Control policy on an Anypoint API instance.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

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
- `label` (String) A human-readable label for this policy instance.
- `order` (Number) The order of policy execution.
- `asset_version` (String) The policy asset version. Defaults to `1.2.2`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.

### Read-Only

- `id` (String) The policy ID.
- `policy_template_id` (String) The policy template ID assigned by the server.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `maximum_requests` (Number) Maximum number of requests allowed in the time period.
- `time_period_in_milliseconds` (Number) The time period in milliseconds for the spike control window.
- `delay_time_in_millis` (Number) The delay time in milliseconds before retrying queued requests.
- `delay_attempts` (Number) The number of attempts to retry before rejecting.

Optional:

- `queuing_limit` (Number) Maximum number of requests that can be queued.
- `expose_headers` (Boolean) Whether to expose rate-limit headers in the response.

## Import

An existing `anypoint_api_policy_spike_control` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_spike_control.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_spike_control" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  api_instance_id = "<api_instance_id>"
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
terraform import anypoint_api_policy_spike_control.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
