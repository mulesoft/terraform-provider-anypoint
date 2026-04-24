---
page_title: "anypoint_rolegroup Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages an Anypoint Platform role group.
---

# anypoint_rolegroup (Resource)

Manages an Anypoint Platform role group.

~> **Note:** This is an Access Management resource and requires the **admin provider** (`anypoint.admin`), which uses admin user credentials along with the `client_id` and `client_secret` of a connected app to authenticate on behalf of the user (`auth_type = "user"`). You must set `provider = anypoint.admin` on this resource. The default provider (connected app credentials only) does not have sufficient privileges for Access Management operations.

## Example Usage

```terraform
# Admin provider – authenticates on behalf of a user using connected app credentials
provider "anypoint" {
  alias         = "admin"
  auth_type     = "user"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  base_url      = var.anypoint_base_url
}

# Simple role group
resource "anypoint_rolegroup" "example" {
  provider = anypoint.admin
  name        = "Organization Administrators"
  description = "Administrators for the organization"
}

# Role group with external names
resource "anypoint_rolegroup" "external_example" {
  provider = anypoint.admin
  name            = "External Administrators"
  description     = "External group administrators"
  organization_id = "your-org-id"

  external_names = [
    {
      external_group_name = "administrators"
      provider_id         = "idp-provider-id"
    }
  ]
}
```

## Schema

### Required

- `description` (String) The description of the role group.
- `name` (String) The name of the role group.

### Optional

- `external_names` (Block List) List of external names for the role group. See [below for nested schema](#nestedschema--external_names).
- `organization_id` (String) The organization ID where the role group will be created. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `created_at` (String) The creation timestamp of the role group.
- `editable` (Boolean) Whether the role group is editable.
- `id` (String) The unique identifier for the role group.
- `updated_at` (String) The last update timestamp of the role group.

<a id="nestedschema--external_names"></a>
### Nested Schema for `external_names`

Required:

- `external_group_name` (String) The external group name.
- `provider_id` (String) The provider ID.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_rolegroup.example <rolegroup_id>
```
