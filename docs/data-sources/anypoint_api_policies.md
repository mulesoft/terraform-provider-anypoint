---
page_title: "anypoint_api_policies Data Source - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Lists all API policies for an API instance in API Manager.
---

# anypoint_api_policies (Data Source)

Lists all API policies applied to an API instance in API Manager.

## Example Usage

```terraform
data "anypoint_api_policies" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "policy_ids" {
  value = [for p in data.anypoint_api_policies.all.policies : p.id]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID where the API instance exists.
- `api_instance_id` (String) The API instance ID.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>/<api_instance_id>`.
- `policies` (List of Object) List of API policies. See [`policies`](#nestedschema--policies) below.

<a id="nestedschema--policies"></a>
### Nested Schema for `policies`

Read-Only:

- `id` (String) The numeric ID of the policy.
- `policy_template_id` (String) The policy template ID.
- `group_id` (String) The Exchange group (organization) ID.
- `asset_id` (String) The Exchange asset ID.
- `asset_version` (String) The Exchange asset version.
- `configuration_json` (String) JSON-encoded policy configuration.
- `order` (Number) The execution order of the policy.
- `disabled` (Boolean) Whether the policy is disabled.
- `pointcut_json` (String) JSON-encoded pointcut data.
