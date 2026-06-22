---
page_title: "anypoint_api_policy_external_oauth2_access_token_enforcement Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the External OAuth 2.0 Access Token Enforcement policy applied to an Anypoint API instance.
---

# anypoint_api_policy_external_oauth2_access_token_enforcement (Data Source)

Reads the External OAuth 2.0 Access Token Enforcement policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_external_oauth2_access_token_enforcement" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "external_oauth2_access_token_enforcement_configuration" {
  value = data.anypoint_api_policy_external_oauth2_access_token_enforcement.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_external_oauth2_access_token_enforcement" "by_id" {
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

- `token_url` (String) URL of the OAuth 2.0 token endpoint.
- `authentication_timeout` (Number) Authentication request timeout in milliseconds.
- `expose_headers` (Boolean) Whether to expose rate-limit headers in the response.
- `max_cache_entries` (Number) Maximum number of entries in the cache.
- `scope_validation_criteria` (String) How scopes are validated (`AND` or `OR`).
- `scopes` (String) Space-separated list of required OAuth scopes.
- `secure_trust_store` (Boolean) Whether to use a secure trust store for token validation.
- `skip_client_id_validation` (Boolean) Whether to skip client ID validation.
