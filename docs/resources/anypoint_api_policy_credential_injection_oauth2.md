---
page_title: "anypoint_api_policy_credential_injection_oauth2 Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a Credential Injection OAuth2 policy on an Anypoint API instance.
---

# anypoint_api_policy_credential_injection_oauth2 (Resource)

Manages a Credential Injection OAuth2 policy on an Anypoint API instance.

## Example Usage

```terraform
resource "anypoint_api_policy_credential_injection_oauth2" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    oauth_service                    = "https://auth.example.com/oauth2/token"
    client_id                        = "my-client-id"
    client_secret                    = "my-client-secret"
    scope                            = ["read", "write"]
    overwrite                        = true
    token_fetch_timeout              = 5000
    allow_request_without_credential = false
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

- `client_id` (String) The OAuth 2.0 client ID.
- `client_secret` (String) The OAuth 2.0 client secret.
- `oauth_service` (String) URL of the OAuth 2.0 token service.

Optional:

- `allow_request_without_credential` (Boolean) Whether to allow requests without credentials.
- `overwrite` (Boolean) Whether to overwrite existing credentials.
- `scope` (Dynamic) Array or string of OAuth 2.0 scopes.
- `token_fetch_timeout` (Number) Timeout in milliseconds for fetching tokens.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_api_policy_credential_injection_oauth2.example {organization_id}/{environment_id}/{api_instance_id}/{policy_id}
```
