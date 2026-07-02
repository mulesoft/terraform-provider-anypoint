---
page_title: "anypoint_role_permissions Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all permissions (roles) assigned to a specific role group.
---

# anypoint_role_permissions (Data Source)

Lists all permissions (roles) assigned to a specific role group.

~> **Note:** This is an Access Management data source and requires the **admin provider** (`anypoint.admin`), which uses admin user credentials along with the `client_id` and `client_secret` of a connected app to authenticate on behalf of the user (`auth_type = "user"`). You must set `provider = anypoint.admin` on this data source. The default provider (connected app credentials only) does not have sufficient privileges for Access Management operations.

-> **Connected App:** This data source requires an **admin connected app** configured with `auth_type = "user"` (user credentials + connected app client credentials). Use the `anypoint.admin` provider alias.

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

# List all permissions for a role group
data "anypoint_role_permissions" "example" {
  provider      = anypoint.admin
  role_group_id = "12345678-1234-1234-1234-123456789abc"
}

# Output all permissions
output "role_permissions" {
  value = data.anypoint_role_permissions.example.permissions
}

# Output permission count
output "permission_count" {
  value = length(data.anypoint_role_permissions.example.permissions)
}

# Output permission names
output "permission_names" {
  value = [for p in data.anypoint_role_permissions.example.permissions : p.name]
}
```

## Schema

### Required

- `role_group_id` (String) The ID of the role group to list permissions for.

### Optional

- `organization_id` (String) The organization ID. If not specified, uses the organization from provider credentials.

### Read-Only

- `permissions` (List of Object) List of permissions (roles) assigned to the role group. See [`permissions`](#nestedschema--permissions) below.

<a id="nestedschema--permissions"></a>
### Nested Schema for `permissions`

Read-Only:

- `id` (String) The role_group_assignment_id.
- `role_id` (String) The role (permission) ID.
- `name` (String) The name of the role (permission).
- `description` (String) The description of the role (permission).
- `context_params` (Map of String) Context parameters (org, envId).
- `created_at` (String) When the assignment was created.
