---
page_title: "anypoint_role Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages an Anypoint Platform role group (custom or default), including its inline permissions and members.
---

# anypoint_role (Resource)

Manages an Anypoint Platform role group (custom or default). Requires Organization Administrator privileges.

This single resource also manages the role group's **permissions** (role
assignments) and **members** inline. You do not need separate resources for those
— assign permissions by their UI display name and add members by username,
directly on this resource.

~> **Authentication:** This resource works fully with a standard **client_credentials**
connected app — role groups, `permissions` (including environment-scoped ones) and
`members`; a separate admin/user provider is **not** required. What matters is that the connected app holds the right **scopes / organization permissions** for the operation; a missing scope surfaces as `HTTP 401`/`403` (that is a missing-scope error, not a wrong auth mode). For role groups specifically, grant the connected app **`admin:access_controls`**, **`view:access_controls`**, and **`read:organization`** (each with `context_params` set to the target `org`). Note that `admin:access_controls` alone is enough to read/write teams but **not** role groups — reading or writing role groups also requires `view:access_controls`. See [Authentication](../index.md).

-> **User-based alternative (optional):** If you would rather rely on a user's permissions than grant the connected app the scopes directly, you can use an admin connected app with `auth_type = "user"` via the `anypoint.admin` provider alias (shown in the example below). This is optional — a client_credentials app with the right scopes is sufficient.

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

resource "anypoint_role" "example" {
  provider    = anypoint.admin
  name        = "API Developers"
  description = "Role group for the API development team"

  # Inline permissions (role assignments). Each permission is referenced by its
  # UI display name (case-insensitive); the provider resolves the name to a role
  # ID at apply time. Use the anypoint_available_permissions data source to discover
  # valid names. When set, this list is authoritative — permissions not listed
  # here are removed on apply.
  permissions = [
    {
      # Organization-scoped permission (context_params carries only the org).
      name = "Exchange Viewer"
      context_params = {
        org = var.organization_id
      }
    },
    {
      # Environment-scoped permission (also carries envId). Some permissions can
      # only be applied to Sandbox/Production environments.
      name = "Read Applications"
      context_params = {
        org   = var.organization_id
        envId = var.environment_id
      }
    },
  ]

  # Inline members, referenced by username (case-insensitive). When set, this
  # list is authoritative — members not listed here are removed on apply. Use the
  # anypoint_users data source to discover usernames.
  members = [
    "jdoe",
    "asmith",
  ]
}
```

Omit `permissions` (or `members`) entirely to leave that aspect **unmanaged** —
the provider will not read, add, or remove permissions (or members) it is not
told to manage. Supplying an empty list (`permissions = []`) is different: it
declares that the role group should have **no** managed permissions and removes
any that exist.

-> **Platform side-effect grants:** When you assign an environment-scoped
permission, the platform may auto-add an organization-scoped "Business Group
Viewer" grant so the grantee can navigate to the business group. That grant is
not part of the assignable catalog and cannot be expressed in configuration; the
provider deliberately ignores it (it is never surfaced in state and never
removed), so it does not cause a perpetual diff. System (internal) assignments
are likewise never modified.

## Schema

### Required

- `name` (String) The name of the role group.

### Optional

- `description` (String) A description of the role group. This attribute is Optional and Computed: if you omit it, the server-resolved value is retained (an omitted `description` is **not** wiped when you change another field such as `name`).
- `organization_id` (String) The organization ID where the role group will be created. If not specified, uses the organization from provider credentials.
- `permissions` (Attributes Set) The set of permissions granted by this role group. When set, this list is authoritative: permissions not listed here are removed on apply. Omit the attribute entirely to leave permissions unmanaged. System (internal) assignments are never modified. (see [below for nested schema](#nestedatt--permissions))
- `members` (Set of String) The set of usernames that are members of this role group. When set, this list is authoritative: members not listed here are removed on apply. Omit the attribute entirely to leave membership unmanaged. Usernames are case-insensitive; use the `anypoint_users` data source to discover usernames.

### Read-Only

- `created_at` (String) The timestamp when the role group was created.
- `editable` (Boolean) Whether the role group can be edited. Default (system) role groups are not editable.
- `external_names` (List of String) External group names mapped to this role group (for SSO/SAML integration). Read-only.
- `id` (String) The unique identifier for the role group.
- `updated_at` (String) The timestamp when the role group was last updated.

<a id="nestedatt--permissions"></a>
### Nested Schema for `permissions`

Required:

- `name` (String) The permission's display name as shown in the Anypoint UI (e.g., `Exchange Viewer`). Case-insensitive. Use the `anypoint_available_permissions` data source to discover valid names.

Optional:

- `context_params` (Map of String) Context parameters for the permission. Typically includes `org` (organization ID) and, for environment-scoped permissions, `envId`.

## Import

An existing role group can be imported using its role group ID (UUID). After a
passthrough import, the `permissions` and `members` sets are populated on the
first `terraform plan`/`apply` that reconciles them against your configuration.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_role.imported
  id       = "<role_group_id>"
}

resource "anypoint_role" "imported" {
  provider        = anypoint.admin
  organization_id = "<organization_id>"
  name            = "<role_name>"
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
terraform import anypoint_role.imported <role_group_id>
```
