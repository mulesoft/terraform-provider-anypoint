---
page_title: "anypoint_role Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Fetches information about an Anypoint Platform role group, including its permissions and members.
---

# anypoint_role (Data Source)

Fetches information about a specific Anypoint Platform role group by ID, including
its permissions (role assignments) and members.

~> **Authentication:** This Access Management data source works with a standard **client_credentials** connected app configured in the provider block — a separate admin/user provider is **not** required. What matters is that the connected app holds the right **scopes / organization permissions** to read this data; a missing scope surfaces as `HTTP 401`/`403` (a missing-scope error, not a wrong auth mode). See [Authentication](../index.md).

## Example Usage

```terraform
# A standard client_credentials connected app is enough for this read.
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Retrieve information about a specific role group
data "anypoint_role" "example" {
  id = "12345678-1234-1234-1234-123456789abc"
}

# Output role group details
output "role_name" {
  value = data.anypoint_role.example.name
}

output "role_description" {
  value = data.anypoint_role.example.description
}

output "is_editable" {
  value = data.anypoint_role.example.editable
}

# The permissions (role assignments) granted by this role group
output "role_permissions" {
  value = data.anypoint_role.example.permissions
}

# The usernames of the role group's members
output "role_members" {
  value = data.anypoint_role.example.members
}
```

## Schema

### Required

- `id` (String) The unique identifier for the role group.

### Optional

- `organization_id` (String) The organization ID where the role group is located. If not specified, uses the organization from provider credentials.

### Read-Only

- `name` (String) The name of the role group.
- `description` (String) A description of the role group.
- `editable` (Boolean) Whether the role group can be edited.
- `external_names` (List of String) External group names mapped to this role group.
- `permissions` (List of Object) The permissions (role assignments) granted by this role group. Excludes system/internal assignments and platform-injected side-effect grants (e.g. the auto-added org-scoped "Business Group Viewer"). See [`permissions`](#nestedatt--permissions) below.
- `members` (List of String) The usernames of members in this role group.
- `created_at` (String) The timestamp when the role group was created.
- `updated_at` (String) The timestamp when the role group was last updated.

<a id="nestedatt--permissions"></a>
### Nested Schema for `permissions`

Read-Only:

- `name` (String) The permission's display name.
- `context_params` (Map of String) Context parameters for the permission (e.g., `org`, `envId`).
