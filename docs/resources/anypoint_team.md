---
page_title: "anypoint_team Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages an Anypoint Platform team.
---

# anypoint_team (Resource)

Manages an Anypoint Platform team.

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

resource "anypoint_team" "example" {
  provider  = anypoint.admin
  name      = "Development Team"
  team_type = "internal"
  # parent_team is optional — omit to create under the org root team

  # Inline role assignments. Roles are referenced by their UI display name
  # (case-insensitive); the provider resolves each name to a role ID at apply time.
  roles = [
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
  provider    = anypoint.admin
  name        = "Frontend Team"
  parent_team = anypoint_team.example.name
  team_type   = "internal"
}
```

## Schema

### Required

- `name` (String) The name of the team.

### Optional

- `organization_id` (String) The organization ID where the team will be created. If not provided, the organization ID will be inferred from the connected app credentials.
- `parent_team` (String, Optional) The name of the parent team. The provider resolves it to an ID automatically (case-insensitive). If omitted, the org's root team is used as the parent — mirroring the Anypoint UI default.
- `team_type` (String) The type of the team. Optional; defaults to `internal` — the same default the Anypoint UI applies (its Create Team dialog only asks for a name and parent, and sends `team_type: "internal"` behind the scenes). Changing the type requires the target type to be enabled in the organization.
- `members` (Attributes Set) The set of members of this team. When set, this list is authoritative: members not listed here are removed on apply. Omit the attribute entirely to leave membership unmanaged. Members assigned via external groups (SAML/SCIM) are never modified. (see [below for nested schema](#nestedatt--members))
- `roles` (Attributes Set) The set of roles (permissions) assigned to this team. When set, this list is authoritative: roles not listed here are removed on apply. Omit the attribute entirely to leave role assignments unmanaged. System (internal) assignments are never modified. (see [below for nested schema](#nestedatt--roles))

### Read-Only

- `created_at` (String) The timestamp when the team was created.
- `id` (String) The unique identifier for the team.
- `updated_at` (String) The timestamp when the team was last updated.

<a id="nestedatt--roles"></a>
### Nested Schema for `roles`

Required:

- `name` (String) The role's display name as shown in the Anypoint UI (e.g., `Exchange Viewer`). Case-insensitive. Use the `anypoint_available_permissions` data source to discover valid names.

Optional:

- `context_params` (Map of String) Context parameters for the role. Typically includes `org` (organization ID) and, for environment-scoped roles, `envId`.

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
  provider = anypoint.admin
  to       = anypoint_team.imported
  id       = "<team_id>"
}

resource "anypoint_team" "imported" {
  provider        = anypoint.admin
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
