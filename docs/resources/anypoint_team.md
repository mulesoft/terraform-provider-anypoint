---
page_title: "anypoint_team Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages an Anypoint Platform team.
---

# anypoint_team (Resource)

Manages an Anypoint Platform team.

~> **Managing `permissions` requires `auth_type = "user"`.** Create, `members`, read and import
work with a `client_credentials` connected app; assigning most `permissions` returns `403
Forbidden` and **adding scopes does not fix it**. Only *Exchange Viewer*, *Exchange
Administrator* and *View Organization* can be assigned under `client_credentials` —
Runtime Manager, CloudHub, Design Center, API Manager, Monitoring and others all fail,
whether organization- or environment-scoped. The symptom is an apply that creates the
team and then fails with `failed to assign permission "...": ... status 403: Forbidden`.
See [Authentication](../index.md).

-> The same permissions assign fine to a **role group** via
[`anypoint_role`](anypoint_role.md) under `client_credentials`. If you only need to grant
permissions, that path has no such restriction.

## Example Usage

```terraform
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Look up the root team to use as parent
data "anypoint_teams" "all" {}

locals {
  root_team_id = [for t in data.anypoint_teams.all.teams : t.id if t.is_root_team][0]
}

resource "anypoint_team" "example" {
  name           = "Development Team"
  team_type      = "internal"
  parent_team_id = local.root_team_id  # optional — omit to default to root team

  # Inline permissions. Permissions are referenced by their UI display name
  # (case-insensitive); the provider resolves each name to a role ID at apply time.
  permissions = [
    {
      name = "Exchange Viewer"
      context_params = {
        org = var.organization_id
      }
    },
  ]

  # Inline members. Members are referenced by username (case-insensitive).
  # membership_type is optional and defaults to "member".
  members = [
    {
      username        = "jdoe"
      membership_type = "maintainer"
    },
    {
      username = "asmith" # membership_type omitted → "member"
    },
  ]
}

resource "anypoint_team" "sub_team" {
  name           = "Frontend Team"
  parent_team_id = anypoint_team.example.id
  team_type      = "internal"
}
```

## Schema

### Required

- `name` (String) The name of the team.

### Optional

- `organization_id` (String) The organization ID where the team will be created. If not provided, the organization ID will be inferred from the connected app credentials.
- `parent_team_id` (String, Optional) The ID of the parent team. If omitted, the org's root team is used as the parent — mirroring the Anypoint UI default. Use the `anypoint_teams` data source to look up team IDs by name.
- `team_type` (String) The type of the team. Optional; defaults to `internal` — the same default the Anypoint UI applies (its Create Team dialog only asks for a name and parent, and sends `team_type: "internal"` behind the scenes). Changing the type requires the target type to be enabled in the organization.
- `members` (Attributes Set) The set of members of this team. When set, this list is authoritative: members not listed here are removed on apply. Omit the attribute entirely to leave membership unmanaged. Members assigned via external groups (SAML/SCIM) are never modified. (see [below for nested schema](#nestedatt--members))
- `permissions` (Attributes Set) The set of permissions assigned to this team, matching what the Anypoint UI calls Permissions and the same shape as `anypoint_role`'s `permissions`. When set, this list is authoritative: permissions not listed here are removed on apply. Omit the attribute entirely to leave assignments unmanaged. System (internal) assignments are never modified. (see [below for nested schema](#nestedatt--permissions))

### Read-Only

- `created_at` (String) The timestamp when the team was created.
- `id` (String) The unique identifier for the team.
- `updated_at` (String) The timestamp when the team was last updated.

<a id="nestedatt--permissions"></a>
### Nested Schema for `permissions`

Required:

- `name` (String) The permission's display name as shown in the Anypoint UI (e.g., `Exchange Viewer`). Case-insensitive. Use the `anypoint_available_permissions` data source to discover valid names.

Optional:

- `context_params` (Map of String) Context parameters for the permission. Typically includes `org` (organization ID) and, for environment-scoped permissions, `envId`.

<a id="nestedatt--members"></a>
### Nested Schema for `members`

Required:

- `username` (String) The member's username. Case-insensitive; use the `anypoint_users` data source to discover usernames.

Optional:

- `membership_type` (String) The membership type: `member` (default) or `maintainer`. Maintainers can additionally manage team membership and child teams. Omit to default to `member`.

## Import

An existing team can be imported using its team ID (UUID).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_team.imported
  id = "<team_id>"
}

resource "anypoint_team" "imported" {
  organization_id = "<organization_id>"
  name            = "<team_name>"
  team_type       = "internal"
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
terraform import anypoint_team.imported <team_id>
```
