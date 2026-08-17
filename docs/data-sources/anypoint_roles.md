---
page_title: "anypoint_roles Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all role groups in an Anypoint Platform organization.
---

# anypoint_roles (Data Source)

Lists all role groups in an Anypoint Platform organization.

~> **Authentication:** This Access Management data source works with a standard **client_credentials** connected app configured in the provider block — a separate admin/user provider is **not** required. What matters is that the connected app holds the right **scopes / organization permissions** to read this data; a missing scope surfaces as `HTTP 401`/`403` (a missing-scope error, not a wrong auth mode). See [Authentication](../index.md).

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

# List all role groups in the organization
data "anypoint_roles" "all" {
  provider = anypoint.admin
}

# Output all role groups
output "all_roles" {
  value = data.anypoint_roles.all.roles
}

# Output role group count
output "role_count" {
  value = length(data.anypoint_roles.all.roles)
}

# Filter to find a specific role by name
output "admin_role" {
  value = [for r in data.anypoint_roles.all.roles : r if r.name == "Organization Administrators"]
}
```

## Schema

### Optional

- `organization_id` (String) The organization ID. If not specified, uses the organization from provider credentials.

### Read-Only

- `roles` (List of Object) The list of role groups. See [`roles`](#nestedschema--roles) below.

<a id="nestedschema--roles"></a>
### Nested Schema for `roles`

Read-Only:

- `id` (String) The unique identifier for the role group.
- `name` (String) The name of the role group.
- `description` (String) A description of the role group.
- `editable` (Boolean) Whether the role group can be edited.
- `external_names` (List of String) External group names mapped to this role group.
- `created_at` (String) The timestamp when the role group was created.
- `updated_at` (String) The timestamp when the role group was last updated.
