---
page_title: "anypoint_users Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists users in the organization.
---

# anypoint_users (Data Source)

Lists users in the organization. Use this to discover usernames for the `members` field of the `anypoint_role` and `anypoint_team` resources (members are referenced by username, case-insensitive).

~> **Authentication:** This Access Management data source works with a standard **client_credentials** connected app configured in the provider block — a separate admin/user provider is **not** required. What matters is that the connected app holds the right **scopes / organization permissions** to read this data; a missing scope surfaces as `HTTP 401`/`403` (a missing-scope error, not a wrong auth mode). See [Authentication](../index.md).

## Example Usage

```terraform
# A standard client_credentials connected app is enough for this read.
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# List all users in the organization
data "anypoint_users" "all" {}

# Output all users
output "all_users" {
  value = data.anypoint_users.all.users
}

# Filter users by name or email
data "anypoint_users" "filtered" {
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

- `id` (String) The unique ID of the user.
- `username` (String) The username. Use this value in the `members` field of the `anypoint_role` or `anypoint_team` resource.
- `first_name` (String) The first name.
- `last_name` (String) The last name.
- `email` (String) The email address.
- `enabled` (Boolean) Whether the user is enabled.
