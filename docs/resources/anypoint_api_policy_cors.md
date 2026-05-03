---
page_title: "anypoint_api_policy_cors Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a CORS policy on an Anypoint API instance.
---

# anypoint_api_policy_cors (Resource)

Manages a CORS policy on an Anypoint API instance.

## Example Usage

```terraform
resource "anypoint_api_policy_cors" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    public_resource     = true
    support_credentials = false
    origin_groups = [
      {
        origins               = ["https://example.com"]
        access_control_max_age = 600
        methods               = ["GET", "POST", "PUT"]
        headers               = ["Content-Type", "Authorization"]
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
- `asset_version` (String) The policy asset version. Defaults to `1.3.2`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.

### Read-Only

- `id` (String) The policy ID.
- `group_id` (String) The policy group ID.
- `asset_id` (String) The policy asset ID.
- `upstream_ids` (List of String) The upstream IDs this policy applies to.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `origin_groups` (Dynamic) Array of origin group configurations for CORS.

Optional:

- `public_resource` (Boolean) Whether the resource is publicly accessible (no CORS preflight required).
- `support_credentials` (Boolean) Whether to allow credentials in CORS requests.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_api_policy_cors.example {organization_id}/{environment_id}/{api_instance_id}/{policy_id}
```
