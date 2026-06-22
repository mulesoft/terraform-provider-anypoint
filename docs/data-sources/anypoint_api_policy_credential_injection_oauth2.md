---
page_title: "anypoint_api_policy_credential_injection_oauth2 Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the Credential Injection OAuth2 policy applied to an Anypoint API instance.
---

# anypoint_api_policy_credential_injection_oauth2 (Data Source)

Reads the Credential Injection OAuth2 policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_credential_injection_oauth2" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "credential_injection_oauth2_configuration" {
  value = data.anypoint_api_policy_credential_injection_oauth2.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_credential_injection_oauth2" "by_id" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  policy_id       = "12345"
}
```

## Schema

### Required

- `environment_id` (String) The environment ID where the API instance lives.
- `api_instance_id` (String) Numeric ID of the API instance this policy is applied to.

### Optional

- `organization_id` (String) Organization ID. Defaults to the provider's org ID if omitted.
- `policy_id` (String) The ID of the policy. If omitted, the data source looks up the policy by asset type on the API instance.

### Read-Only

- `id` (String) Unique identifier of the applied policy.
- `label` (String) A human-readable label for this policy instance.
- `configuration` (Attributes) Policy configuration with typed fields. See [`configuration`](#nestedschema--configuration) below.
- `order` (Number) Execution order of the policy.
- `disabled` (Boolean) Whether the policy is disabled.
- `policy_template_id` (String) Policy template ID assigned by the server.
- `asset_version` (String) Version of the policy asset.
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.
- `pointcut_data` (String) Pointcut definition as a JSON string.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Read-Only:

- `oauth_service` (String) URL of the OAuth 2.0 token service.
- `client_id` (String) The OAuth 2.0 client ID.
- `client_secret` (String) The OAuth 2.0 client secret.
- `scope` (Dynamic) Array of OAuth 2.0 scopes.
- `token_fetch_timeout` (Number) Timeout in milliseconds for fetching the OAuth token. Defaults to `10000`. The provider always sends this field (defaulting to `10000`) — omitting it does not cause HTTP 400.
- `overwrite` (Boolean) Whether to overwrite an existing credential header on the request. Defaults to `false`. The provider always sends this field.
- `allow_request_without_credential` (Boolean) Whether to allow requests to pass through without injected credentials. Defaults to `false`. The provider always sends this field.
