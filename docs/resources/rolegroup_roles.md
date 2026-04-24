---
page_title: "anypoint_rolegroup_roles Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages role assignments for an Anypoint Platform role group. This resource manages all roles assigned to a role group as a single unit.
---

# anypoint_rolegroup_roles (Resource)

Manages role assignments for an Anypoint Platform role group. This resource manages all roles assigned to a role group as a single unit.

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

resource "anypoint_rolegroup" "example" {
  provider = anypoint.admin
  name        = "Example Role Group"
  description = "Example role group for demonstrating role assignments"
}

resource "anypoint_rolegroup_roles" "example" {
  provider        = anypoint.admin
  rolegroup_id    = anypoint_rolegroup.example.id
  organization_id = "your-org-id"

  roles = [
    {
      role_id = "d74ef94a-4292-4896-b860-b05bd7f90d6d"
      context_params = {
        org = "your-org-id"
      }
    },
    {
      role_id = "ceeabcd5-eb31-41c9-b387-01a0e9095620"
      context_params = {
        org = "your-org-id"
      }
    }
  ]
}
```

## Schema

### Required

- `rolegroup_id` (String) The ID of the role group to assign roles to.
- `roles` (Block List) List of roles to assign to the role group. See [below for nested schema](#nestedschema--roles).

### Optional

- `organization_id` (String) The organization ID where the role group is located. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) The unique identifier for this resource (same as rolegroup_id).

<a id="nestedschema--roles"></a>
### Nested Schema for `roles`

Required:

- `context_params` (Map of String) Context parameters for the role assignment (e.g., organization, environment).
- `role_id` (String) The ID of the role to assign.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_rolegroup_roles.example <rolegroup_id>
```
