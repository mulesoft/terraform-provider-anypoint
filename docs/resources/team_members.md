---
page_title: "anypoint_team_members Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Resource to manage team member assignments in Anypoint Platform.
---

# anypoint_team_members (Resource)

Resource to manage team member assignments in Anypoint Platform.

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
  team_name      = "Example Team"
  parent_team_id = "root-team-id"
  team_type      = "internal"
}

resource "anypoint_team_members" "example" {
  provider = anypoint.admin
  team_id  = anypoint_team.example.id

  members = [
    {
      id              = "f7f43384-b33e-470c-ad4c-285aa0c01212"
      membership_type = "member"
    },
    {
      id              = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      membership_type = "maintainer"
    }
  ]
}
```

## Schema

### Required

- `members` (Block List) List of team members with their membership types. See [below for nested schema](#nestedschema--members).
- `team_id` (String) The ID of the team to manage members for.

### Optional

- `organization_id` (String) The organization ID where the team is located. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) Team members identifier (same as team_id).
- `users` (Block List) Computed list of team members with full user details. See [below for nested schema](#nestedschema--users).

<a id="nestedschema--members"></a>
### Nested Schema for `members`

Required:

- `id` (String) The ID of the user to add to the team.
- `membership_type` (String) The membership type for the user (member or maintainer).

<a id="nestedschema--users"></a>
### Nested Schema for `users`

Read-Only:

- `email` (String) The email of the user.
- `first_name` (String) The first name of the user.
- `id` (String) The ID of the user.
- `last_name` (String) The last name of the user.
- `membership_type` (String) The membership type of the user.
- `username` (String) The username of the user.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_team_members.example <team_id>
```
