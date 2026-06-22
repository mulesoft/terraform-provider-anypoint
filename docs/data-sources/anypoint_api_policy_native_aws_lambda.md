---
page_title: "anypoint_api_policy_native_aws_lambda Data Source - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Reads the Native AWS Lambda policy applied to an Anypoint API instance.
---

# anypoint_api_policy_native_aws_lambda (Data Source)

Reads the Native AWS Lambda policy applied to an Anypoint API instance. If `policy_id` is omitted, the data source automatically finds the policy by its asset type on the given API instance.

## Example Usage

```terraform
# Auto-discover by asset type (recommended when only one such policy exists on the instance)
data "anypoint_api_policy_native_aws_lambda" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "native_aws_lambda_configuration" {
  value = data.anypoint_api_policy_native_aws_lambda.example.configuration
}

# Or look up by explicit policy_id
data "anypoint_api_policy_native_aws_lambda" "by_id" {
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

- `arn` (String) The ARN of the AWS Lambda function.
- `payload_passthrough` (Boolean) Whether to pass the request payload directly to Lambda.
- `invocation_mode` (String) Lambda invocation mode (`synchronous` or `asynchronous`).
- `authentication_mode` (String) AWS authentication mode (e.g. `static_credentials`, `iam_role`).
- `credentials` (Dynamic) AWS credentials object with `access_key_id`, `secret_access_key`, and optional `session_token`.
