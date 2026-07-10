---
page_title: "anypoint_available_permissions Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all available permissions that can be assigned to role groups.
---

# anypoint_available_permissions (Data Source)

Lists all available permissions that can be assigned to role groups. Use this to look up permission IDs by name instead of hardcoding UUIDs.

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

# List all available permissions
data "anypoint_available_permissions" "all" {
  provider = anypoint.admin
}

# Output all available permissions
output "all_available_permissions" {
  value = data.anypoint_available_permissions.all.permissions
}

# Filter permissions by name
data "anypoint_available_permissions" "read_apps" {
  provider    = anypoint.admin
  name_filter = "Read Applications"
}

# Output filtered permissions
output "read_apps_permissions" {
  value = data.anypoint_available_permissions.read_apps.permissions
}
```

## Schema

### Optional

- `name_filter` (String) Optional filter to match permissions by name (case-insensitive substring match). For example, 'Read Applications' returns only permissions with that name.

### Read-Only

- `permissions` (List of Object) List of available permissions. See [`permissions`](#nestedschema--permissions) below.

<a id="nestedschema--permissions"></a>
### Nested Schema for `permissions`

Read-Only:

- `role_id` (String) The unique ID of the permission.
- `name` (String) The human-readable name of the permission. Use this value in the `permissions[].name` field of the `anypoint_role` resource (permissions are referenced by name, case-insensitive; the provider resolves the name to a role ID at apply time).
- `description` (String) A description of what the permission grants.
