---
page_title: "anypoint_api_policy_jwt_validation Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the JWT Validation policy applied to an Anypoint API instance.
---

# anypoint_api_policy_jwt_validation (Data Source)

Reads the JWT Validation policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_jwt_validation" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "jwt_validation_configuration" {
  value = data.anypoint_api_policy_jwt_validation.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_jwt_validation" "by_id" {
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

- `jwt_origin` (String) Where the JWT token is extracted from (e.g. `httpBearerAuthenticationHeader`).
- `claims_to_headers` (Dynamic) Array mapping JWT claims to response headers.
- `client_id_expression` (String) Expression to extract the client ID from the request.
- `custom_key_expression` (String) Custom expression to resolve the signing key.
- `jwks_service_connection_timeout` (Number) Connection timeout in milliseconds for JWKS endpoint.
- `jwks_service_time_to_live` (Number) TTL in seconds for cached JWKS keys.
- `jwks_url` (String) URL to the JWKS endpoint for key retrieval.
- `jwt_expression` (String) Custom expression to extract the JWT token.
- `jwt_key_origin` (String) Source of the signing key (e.g. `jwks`, `text`).
- `mandatory_aud_claim` (Boolean) Whether the `aud` claim is mandatory.
- `mandatory_custom_claims` (Dynamic) Array of custom claims that must be present.
- `mandatory_exp_claim` (Boolean) Whether the `exp` (expiration) claim is mandatory.
- `mandatory_nbf_claim` (Boolean) Whether the `nbf` (not before) claim is mandatory.
- `non_mandatory_custom_claims` (Dynamic) Array of optional custom claims to validate if present.
- `signing_key_length` (Number) The key length for the signing algorithm.
- `signing_method` (String) The signing algorithm (e.g. `rsa`, `hmac`).
- `skip_client_id_validation` (Boolean) Whether to skip client ID validation.
- `supported_audiences` (String) Comma-separated list of supported audience values.
- `text_key` (String) The inline signing key when `jwt_key_origin` is `text`.
- `validate_aud_claim` (Boolean) Whether to validate the `aud` (audience) claim.
- `validate_custom_claim` (Boolean) Whether to validate custom claims.
