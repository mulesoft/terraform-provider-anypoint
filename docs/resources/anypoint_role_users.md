---
page_title: "anypoint_role_users Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Assigns a user to a role group.
---

# anypoint_role_users (Resource)

Assigns a user to a role group. This creates a membership between a user and a role group, granting the user all permissions assigned to that role group.

~> **Note:** This is an Access Management resource and requires the **admin provider** (`anypoint.admin`), which uses admin user credentials along with the `client_id` and `client_secret` of a connected app to authenticate on behalf of the user (`auth_type = "user"`). You must set `provider = anypoint.admin` on this resource. The default provider (connected app credentials only) does not have sufficient privileges for Access Management operations.

-> **Connected App:** This resource requires an **admin connected app** configured with `auth_type = "user"` (user credentials + connected app client credentials). Use the `anypoint.admin` provider alias. A standard connected app (client credentials only) does not have sufficient privileges for Access Management operations.

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

# Look up user by username
data "anypoint_users" "developer" {
  provider = anypoint.admin
  username = "john.doe@example.com"
}

resource "anypoint_role_users" "example" {
  provider      = anypoint.admin
  role_group_id = anypoint_role.example.id
  user_id       = data.anypoint_users.developer.users[0].id
}
```

## Schema

### Required

- `role_group_id` (String) The ID of the role group to add the user to.
- `user_id` (String) The ID of the user to add to the role group. Use the anypoint_users data source to look up by username.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider's org.

### Read-Only

- `email` (String) The email of the assigned user.
- `first_name` (String) The first name of the assigned user.
- `id` (String) Unique identifier for this user-role-group assignment (format: {role_group_id}:{user_id}).
- `last_name` (String) The last name of the assigned user.
- `username` (String) The username of the assigned user (computed after creation).

## Import

An existing role-user assignment can be imported using a composite ID format: `{role_group_id}:{user_id}`.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_role_users.imported
  id       = "<role_group_id>:<user_id>"
}

resource "anypoint_role_users" "imported" {
  provider      = anypoint.admin
  role_group_id = "<role_group_id>"
  user_id       = "<user_id>"
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

### Using the CLI (deprecated, Terraform < 1.5)

```shell
terraform import anypoint_role_users.imported <role_group_id>:<user_id>
```
