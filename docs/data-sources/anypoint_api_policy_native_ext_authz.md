---
page_title: "anypoint_api_policy_native_ext_authz Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the Native External Authorization policy applied to an Anypoint API instance.
---

# anypoint_api_policy_native_ext_authz (Data Source)

Reads the Native External Authorization policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_native_ext_authz" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "native_ext_authz_configuration" {
  value = data.anypoint_api_policy_native_ext_authz.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_native_ext_authz" "by_id" {
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

- `server_type` (String) The server type (e.g. `grpc`, `http`).
- `uri` (String) The URI of the external service.
- `allowed_headers` (Dynamic) Array of headers to forward to the external service.
- `include_peer_certificate` (Boolean) Whether to include the peer certificate in the authorization request.
- `path_prefix` (String) Path prefix for the external authorization request.
- `request_timeout` (Number) Request timeout in milliseconds.
- `server_api_version` (String) The API version of the external authorization server.
- `service_request_headers_to_add` (Dynamic) Array of headers to add to the authorization request.
- `service_response_client_headers` (Dynamic) Array of headers from the authorization response to send to the client.
- `service_response_client_headers_on_success` (Dynamic) Array of headers to send on successful authorization.
- `service_response_upstream_headers` (Dynamic) Array of headers from the authorization response to send upstream.
- `service_response_upstream_headers_to_append` (Dynamic) Array of headers from the authorization response to append upstream.
