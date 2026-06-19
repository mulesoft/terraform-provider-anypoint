---
page_title: "anypoint_api_policy Data Source - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Retrieves a single API policy by ID from API Manager.
---

# anypoint_api_policy (Data Source)

Retrieves a single API policy applied to an API instance in API Manager.

## Example Usage

```terraform
data "anypoint_api_policy" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  policy_id       = var.policy_id
}

output "policy_template_id" {
  value = data.anypoint_api_policy.example.policy_template_id
}

output "policy_configuration" {
  value = data.anypoint_api_policy.example.configuration_json
}
```

## Schema

### Required

- `environment_id` (String) The environment ID where the API instance exists.
- `api_instance_id` (String) The API instance ID.
- `policy_id` (String) The ID of the policy to retrieve.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>/<api_instance_id>/<policy_id>`.
- `policy_template_id` (String) The policy template ID.
- `group_id` (String) The Exchange group (organization) ID.
- `asset_id` (String) The Exchange asset ID.
- `asset_version` (String) The Exchange asset version.
- `configuration_json` (String) JSON-encoded policy configuration.
- `order` (Number) The execution order of the policy.
- `disabled` (Boolean) Whether the policy is disabled.
- `pointcut_json` (String) JSON-encoded pointcut data.
