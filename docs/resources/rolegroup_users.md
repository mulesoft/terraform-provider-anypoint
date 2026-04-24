---
page_title: "anypoint_rolegroup_users Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages user assignments for an Anypoint Platform role group. This resource manages all users assigned to a role group as a single unit.
---

# anypoint_rolegroup_users (Resource)

Manages user assignments for an Anypoint Platform role group. This resource manages all users assigned to a role group as a single unit.

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
  description = "Example role group for demonstrating user assignments"
}

resource "anypoint_rolegroup_users" "example" {
  provider        = anypoint.admin
  rolegroup_id    = anypoint_rolegroup.example.id
  organization_id = "your-org-id"

  user_ids = [
    "e0102052-4e55-4e61-985b-c284c97f3688",
    "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  ]
}
```

## Schema

### Required

- `rolegroup_id` (String) The ID of the role group to assign users to.
- `user_ids` (List of String) List of user IDs to assign to the role group.

### Optional

- `organization_id` (String) The organization ID where the role group is located. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) The unique identifier for this resource (same as rolegroup_id).
- `users` (Block List) List of users assigned to the role group (computed). See [below for nested schema](#nestedschema--users).

<a id="nestedschema--users"></a>
### Nested Schema for `users`

Read-Only:

- `email` (String) The user's email address.
- `enabled` (Boolean) Whether the user is enabled.
- `first_name` (String) The user's first name.
- `id` (String) The user ID.
- `idprovider_id` (String) The identity provider ID.
- `last_name` (String) The user's last name.
- `organization_id` (String) The organization ID.
- `username` (String) The username.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_rolegroup_users.example <rolegroup_id>
```
