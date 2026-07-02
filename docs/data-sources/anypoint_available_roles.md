---
page_title: "anypoint_available_roles Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all available roles (permissions) that can be assigned to role groups.
---

# anypoint_available_roles (Data Source)

Lists all available roles (permissions) that can be assigned to role groups. Use this to look up role IDs by name instead of hardcoding UUIDs.

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

# List all available roles
data "anypoint_available_roles" "all" {
  provider = anypoint.admin
}

# Output all available roles
output "all_available_roles" {
  value = data.anypoint_available_roles.all.roles
}

# Filter roles by name
data "anypoint_available_roles" "read_apps" {
  provider    = anypoint.admin
  name_filter = "Read Applications"
}

# Output filtered roles
output "read_apps_roles" {
  value = data.anypoint_available_roles.read_apps.roles
}
```

## Schema

### Optional

- `name_filter` (String) Optional filter to match roles by name (case-insensitive substring match). For example, 'Read Applications' returns only roles with that name.

### Read-Only

- `roles` (List of Object) List of available roles (permissions). See [`roles`](#nestedschema--roles) below.

<a id="nestedschema--roles"></a>
### Nested Schema for `roles`

Read-Only:

- `role_id` (String) The unique ID of the role. Use this as the role_id in anypoint_role_permission.
- `name` (String) The human-readable name of the role.
- `description` (String) A description of what the role grants.
