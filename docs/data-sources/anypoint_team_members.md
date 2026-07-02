---
page_title: "anypoint_team_members Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all members of a specific team.
---

# anypoint_team_members (Data Source)

Lists all members of a specific team.

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

# List all members of a team
data "anypoint_team_members" "example" {
  provider = anypoint.admin
  team_id  = "12345678-1234-1234-1234-123456789abc"
}

# Output all members
output "team_members" {
  value = data.anypoint_team_members.example.members
}

# Output member count
output "member_count" {
  value = data.anypoint_team_members.example.total
}

# Output maintainers only
output "maintainers" {
  value = [for m in data.anypoint_team_members.example.members : m if m.membership_type == "maintainer"]
}
```

## Schema

### Required

- `team_id` (String) The ID of the team to list members for.

### Optional

- `organization_id` (String) The organization ID. If not specified, uses the organization from provider credentials.

### Read-Only

- `members` (List of Object) List of members in the team. See [`members`](#nestedschema--members) below.
- `total` (Number) Total number of members in the team.

<a id="nestedschema--members"></a>
### Nested Schema for `members`

Read-Only:

- `id` (String) The user ID.
- `membership_type` (String) The membership type (member or maintainer).
- `is_assigned_via_external_groups` (Boolean) Whether the membership was assigned via external groups (e.g., SAML/SCIM).
- `created_at` (String) When the membership was created.
