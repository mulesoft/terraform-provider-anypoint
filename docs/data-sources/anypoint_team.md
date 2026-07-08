---
page_title: "anypoint_team Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Fetches information about an Anypoint Platform team.
---

# anypoint_team (Data Source)

Fetches information about an Anypoint Platform team.

## Example Usage

```terraform
data "anypoint_team" "ops" {
  id              = "team-uuid-here"
  organization_id = var.organization_id
}

output "team_name" {
  value = data.anypoint_team.ops.name
}

output "team_roles" {
  value = data.anypoint_team.ops.roles
}

output "team_members" {
  value = data.anypoint_team.ops.members
}
```

## Schema

### Required

- `id` (String) The unique identifier for the team.

### Optional

- `organization_id` (String) The organization ID where the team is located. If not specified, uses the organization from provider credentials.

### Read-Only

- `name` (String) The name of the team.
- `parent_team_id` (String) The parent team ID.
- `team_type` (String) The type of the team.
- `roles` (Attributes List) The roles (permissions) assigned to this team. Excludes system/internal assignments. (see [below for nested schema](#nestedatt--roles))
- `members` (Attributes List) The members of this team. (see [below for nested schema](#nestedatt--members))
- `created_at` (String) The timestamp when the team was created.
- `updated_at` (String) The timestamp when the team was last updated.

<a id="nestedatt--roles"></a>
### Nested Schema for `roles`

Read-Only:

- `name` (String) The role's display name.
- `context_params` (Map of String) Context parameters for the role (e.g., `org`, `envId`).

<a id="nestedatt--members"></a>
### Nested Schema for `members`

Read-Only:

- `username` (String) The member's username.
- `membership_type` (String) The membership type (`member` or `maintainer`).
- `is_assigned_via_external_groups` (Boolean) Whether the membership was assigned via external groups (e.g., SAML/SCIM).
