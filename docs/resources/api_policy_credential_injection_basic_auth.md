---
page_title: "anypoint_api_policy_credential_injection_basic_auth Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a Credential Injection Basic Auth policy on an Anypoint API instance.
---

# anypoint_api_policy_credential_injection_basic_auth (Resource)

Manages a Credential Injection Basic Auth policy on an Anypoint API instance.

## Example Usage

```terraform
resource "anypoint_api_policy_credential_injection_basic_auth" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    username  = "service-account"
    password  = "service-password"
    overwrite = true
  }

  upstream_ids = [anypoint_api_upstream.example.id]
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
- `asset_version` (String) The policy asset version. Defaults to `1.0.1`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.

### Read-Only

- `id` (String) The policy ID.
- `group_id` (String) The policy group ID.
- `asset_id` (String) The policy asset ID.
- `upstream_ids` (List of String) The upstream IDs this policy applies to.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `password` (String) The password for authentication.
- `username` (String) The username for authentication.

Optional:

- `custom_header` (String) Custom header name for injecting credentials.
- `overwrite` (Boolean) Whether to overwrite existing credentials.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_api_policy_credential_injection_basic_auth.example {organization_id}/{environment_id}/{api_instance_id}/{policy_id}
```
