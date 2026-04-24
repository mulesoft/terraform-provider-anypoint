---
page_title: "anypoint_team_roles Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Resource to manage role assignments for a team in Anypoint Platform.
---

# anypoint_team_roles (Resource)

Resource to manage role assignments for a team in Anypoint Platform.

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

resource "anypoint_team" "example" {
  provider = anypoint.admin
  team_name      = "Example Team for Roles"
  parent_team_id = "root-team-id"
  team_type      = "internal"
}

resource "anypoint_team_roles" "example" {
  provider = anypoint.admin
  team_id  = anypoint_team.example.id

  roles = [
    {
      role_id = "98f87b9d-3e41-49cc-a171-f2580a742049"
      context_params = {
        org = "your-org-id"
      }
    }
  ]
}
```

## Schema

### Required

- `roles` (Block List) List of role assignments for the team. See [below for nested schema](#nestedschema--roles).
- `team_id` (String) The ID of the team to assign roles to.

### Optional

- `organization_id` (String) The organization ID where the team is located. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) Team roles identifier (same as team_id).

<a id="nestedschema--roles"></a>
### Nested Schema for `roles`

Required:

- `role_id` (String) The ID of the role to assign.

Optional:

- `context_params` (Map of String) Context parameters for the role assignment.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_team_roles.example <team_id>
```
