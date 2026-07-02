---
page_title: "anypoint_team_roles Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all roles (permissions) assigned to a specific team.
---

# anypoint_team_roles (Data Source)

Lists all roles (permissions) assigned to a specific team.

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

# List all roles assigned to a team
data "anypoint_team_roles" "example" {
  provider = anypoint.admin
  team_id  = "12345678-1234-1234-1234-123456789abc"
}

# Output all roles
output "team_roles" {
  value = data.anypoint_team_roles.example.roles
}

# Output role count
output "role_count" {
  value = data.anypoint_team_roles.example.total
}

# Output role names
output "role_names" {
  value = [for r in data.anypoint_team_roles.example.roles : r.name]
}
```

## Schema

### Required

- `team_id` (String) The ID of the team to list roles for.

### Optional

- `organization_id` (String) The organization ID. If not specified, uses the organization from provider credentials.

### Read-Only

- `roles` (List of Object) List of roles (permissions) assigned to the team. See [`roles`](#nestedschema--roles) below.
- `total` (Number) Total number of roles assigned to the team.

<a id="nestedschema--roles"></a>
### Nested Schema for `roles`

Read-Only:

- `id` (String) The role_group_assignment_id.
- `role_id` (String) The role (permission) ID.
- `name` (String) The name of the role.
- `description` (String) The description of the role.
- `context_params` (Map of String) Context parameters (org, envId).
- `created_at` (String) When the assignment was created.
