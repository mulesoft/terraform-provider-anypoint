---
page_title: "anypoint_api_group Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages an API Group in Anypoint API Manager. An API Group bundles multiple API instances across environments under a shared versioned contract.
---

# anypoint_api_group (Resource)

Manages an API Group in Anypoint API Manager. An API Group bundles multiple API instances across environments under a shared versioned contract.

## Example Usage

```terraform
resource "anypoint_api_group" "example" {
  organization_id = var.organization_id
  name            = "Order Processing APIs"

  versions = [
    {
      name = "v1"
      instances = [
        {
          environment_id       = var.environment_id
          group_instance_label = "production"
          api_instances        = [12345, 67890]
        }
      ]
    },
    {
      name = "v2"
      instances = [
        {
          environment_id       = var.staging_environment_id
          group_instance_label = "staging"
          api_instances        = [11111]
        }
      ]
    }
  ]
}
```

## Schema

### Required

- `name` (String) Display name of the API Group.
- `versions` (Block List) List of named versions defined in this API Group. See [below for nested schema](#nestedschema--versions).

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) Unique identifier of the API Group (numeric, assigned by the platform).

<a id="nestedschema--versions"></a>
### Nested Schema for `versions`

Required:

- `name` (String) Name of the version (e.g. 'v1', 'v2').
- `instances` (Block List) API instances associated with this version. See [below for nested schema](#nestedschema--versions--instances).

Read-Only:

- `id` (String) Version ID assigned by the platform (computed on create).

<a id="nestedschema--versions--instances"></a>
### Nested Schema for `versions.instances`

Required:

- `environment_id` (String) Environment ID that owns the API instances.
- `api_instances` (List of Number) Numeric IDs of the API instances to include.

Optional:

- `group_instance_label` (String) Optional label for this instance group. Defaults to `""`.

## Import

Import is supported using the following format:

```shell
terraform import anypoint_api_group.example organization_id/group_id
```
