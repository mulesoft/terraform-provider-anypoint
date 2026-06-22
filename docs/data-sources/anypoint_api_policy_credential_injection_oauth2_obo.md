---
page_title: "anypoint_api_policy_credential_injection_oauth2_obo Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the Credential Injection OAuth2 On-Behalf-Of policy applied to an Anypoint API instance.
---

# anypoint_api_policy_credential_injection_oauth2_obo (Data Source)

Reads the Credential Injection OAuth2 On-Behalf-Of policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_credential_injection_oauth2_obo" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "credential_injection_oauth2_obo_configuration" {
  value = data.anypoint_api_policy_credential_injection_oauth2_obo.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_credential_injection_oauth2_obo" "by_id" {
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

- `client_id` (String) The OAuth 2.0 client ID.
- `client_secret` (String) The OAuth 2.0 client secret.
- `flow` (String) The OAuth 2.0 grant flow type.
- `token_endpoint` (String) URL of the OAuth 2.0 token endpoint.
- `ciba_enabled` (Boolean) Whether CIBA (Client-Initiated Backchannel Authentication) is enabled.
- `ciba_endpoint` (String) The backchannel authentication endpoint URL (used when `ciba_enabled` is true).
- `ciba_binding_message` (String) A human-readable binding message sent to the user's authentication device (used when `ciba_enabled` is true).
- `ciba_login_hint_claim` (String) The claim used to identify the end user in the CIBA flow (used when `ciba_enabled` is true).
- `distributed` (Boolean) Whether to use distributed token caching across cluster nodes.
- `requested_token_type` (String) The type of the token to be returned by the token endpoint.
- `scope` (String) Array or string of OAuth 2.0 scopes.
- `subject_token_type` (String) The type of the subject token (e.g. `urn:ietf:params:oauth:token-type:access_token`).
- `target_type` (String) The target resource type for on-behalf-of flow.
- `target_value` (String) The target resource value for on-behalf-of flow.
- `timeout` (Number) Timeout value in milliseconds.
