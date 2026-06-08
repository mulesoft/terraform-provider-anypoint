---
page_title: "anypoint_api_policy Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages a policy applied to an API instance in Anypoint API Manager. Use policy_type for known policies (auto-resolves group_id, asset_id, and default version), or provide group_id + asset_id + asset_version directly for custom policies.
---

# anypoint_api_policy (Resource)

Manages a policy applied to an API instance in Anypoint API Manager. Use `policy_type` for known policies (auto-resolves group_id, asset_id, and default version), or provide `group_id` + `asset_id` + `asset_version` directly for custom policies.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

### Using policy_type for a known policy

```terraform
resource "anypoint_api_policy" "rate_limit" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  policy_type = "rate-limiting"
  label       = "rate-limit-100rpm"
  order       = 1

  configuration_data = jsonencode({
    key_selector = "#[attributes.queryParams['identifier']]"
    rate_limits = [
      {
        maximum_requests            = 100
        time_period_in_milliseconds = 60000
      }
    ]
    expose_headers = true
    clusterizable  = true
  })
}
```

### Using explicit group_id, asset_id, asset_version for a custom policy

```terraform
resource "anypoint_api_policy" "custom" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  group_id      = "my-org-id"
  asset_id      = "my-custom-policy"
  asset_version = "1.0.0"
  label         = "custom-policy"
  order         = 2

  configuration_data = jsonencode({
    custom_field = "value"
  })
}
```

## Schema

### Required

- `environment_id` (String) Environment ID where the API instance lives.
- `api_instance_id` (String) Numeric ID of the API instance this policy is applied to.
- `configuration_data` (String) Policy configuration as a JSON string. Use `jsonencode()` to set this. Fields vary by policy type; the provider validates known policies at plan time.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `policy_type` (String) Known policy type name (e.g. 'rate-limiting', 'cors', 'jwt-validation'). When set, group_id, asset_id, and asset_version are auto-resolved from the built-in registry. You can still override asset_version to pin a specific version. For custom policies not in the registry, omit this and set group_id + asset_id + asset_version directly.
- `group_id` (String) Exchange group ID for the policy asset. Auto-resolved when policy_type is set.
- `asset_id` (String) Exchange asset ID that identifies the policy type. Auto-resolved when policy_type is set.
- `asset_version` (String) Version of the policy asset from Exchange. Auto-resolved to default when policy_type is set, but can be overridden.
- `label` (String) A human-readable label for this policy instance.
- `order` (Number) Execution order of the policy. Lower numbers execute first.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.
- `pointcut_data` (String) Pointcut definition as a JSON string. Restricts the policy to specific resources (methods and/or URIs). When null the policy applies to all resources. Use `jsonencode()` to set this. See [Pointcut Data](#pointcut-data) below.

### Read-Only

- `id` (String) Unique identifier of the applied policy.
- `policy_template_id` (String) Policy template ID assigned by the server.


## Pointcut Data

The optional `pointcut_data` attribute restricts the policy to specific HTTP methods and/or URI patterns, matching what is configured under "Apply configurations to specific methods & resources" in the Anypoint Platform UI.

Each element in the array maps to one condition row in the UI:

- `methodRegex` — pipe-separated HTTP methods (e.g. `GET`, `GET|POST`). Omit or set to `.*` to match all methods.
- `uriTemplateRegex` — regex for the URI path (e.g. `/api/v1/.*`). Omit or set to `.*` to match all paths.

```hcl
# Apply policy to GET and POST requests on /api/v1/* only
pointcut_data = jsonencode([
  {
    methodRegex      = "GET|POST"
    uriTemplateRegex = "/api/v1/.*"
  }
])

# Multiple conditions (logical OR — policy applies if any condition matches)
pointcut_data = jsonencode([
  {
    methodRegex      = "GET"
    uriTemplateRegex = "/api/v1/read/.*"
  },
  {
    methodRegex      = "POST|PUT"
    uriTemplateRegex = "/api/v1/write/.*"
  }
])
```

## Import

An existing API policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in the Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  api_instance_id = "<api_instance_id>"
  policy_type     = "<policy_type>"
  configuration_data = jsonencode({})
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
terraform import anypoint_api_policy.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
