---
page_title: "anypoint_role_users Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all users assigned to a specific role group.
---

# anypoint_role_users (Data Source)

Lists all users assigned to a specific role group.

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

# List all users assigned to a role group
data "anypoint_role_users" "example" {
  provider      = anypoint.admin
  role_group_id = "12345678-1234-1234-1234-123456789abc"
}

# Output all users
output "role_users" {
  value = data.anypoint_role_users.example.users
}

# Output user count
output "user_count" {
  value = data.anypoint_role_users.example.total
}

# Output user emails
output "user_emails" {
  value = [for u in data.anypoint_role_users.example.users : u.email]
}
```

## Schema

### Required

- `role_group_id` (String) The ID of the role group to list users for.

### Optional

- `organization_id` (String) The organization ID. If not specified, uses the organization from provider credentials.

### Read-Only

- `users` (List of Object) List of users assigned to the role group. See [`users`](#nestedschema--users) below.
- `total` (Number) Total number of users assigned to the role group.

<a id="nestedschema--users"></a>
### Nested Schema for `users`

Read-Only:

- `id` (String) The user ID.
- `username` (String) The username.
- `first_name` (String) The user's first name.
- `last_name` (String) The user's last name.
- `email` (String) The user's email address.
