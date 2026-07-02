---
page_title: "anypoint_users Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists users in the organization.
---

# anypoint_users (Data Source)

Lists users in the organization. Use this to look up user IDs by username or email instead of hardcoding UUIDs in anypoint_role_users resources.

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

# List all users in the organization
data "anypoint_users" "all" {
  provider = anypoint.admin
}

# Output all users
output "all_users" {
  value = data.anypoint_users.all.users
}

# Filter users by name or email
data "anypoint_users" "filtered" {
  provider    = anypoint.admin
  name_filter = "john.doe"
}

# Output filtered users
output "filtered_users" {
  value = data.anypoint_users.filtered.users
}

# Find a specific user by email
output "user_id_by_email" {
  value = [for u in data.anypoint_users.all.users : u.id if u.email == "john.doe@example.com"][0]
}
```

## Schema

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider's org.
- `name_filter` (String) Optional filter to match users by username, email, first name, or last name (case-insensitive substring match).

### Read-Only

- `users` (List of Object) List of users in the organization. See [`users`](#nestedschema--users) below.

<a id="nestedschema--users"></a>
### Nested Schema for `users`

Read-Only:

- `id` (String) The unique ID of the user. Use this as the user_id in anypoint_role_users.
- `username` (String) The username.
- `first_name` (String) The first name.
- `last_name` (String) The last name.
- `email` (String) The email address.
- `enabled` (Boolean) Whether the user is enabled.
