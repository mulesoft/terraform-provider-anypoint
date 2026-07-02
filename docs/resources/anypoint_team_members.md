---
page_title: "anypoint_team_members Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Assigns a user to a team.
---

# anypoint_team_members (Resource)

Assigns a user to a team. This creates a membership between a user and a team, granting the user access to resources visible to that team.

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

# Add user as a regular member
resource "anypoint_team_members" "example_member" {
  provider        = anypoint.admin
  team_id         = anypoint_team.example.id
  user_id         = data.anypoint_users.developer.users[0].id
  membership_type = "member"
}

# Add user as a maintainer (can manage team membership)
resource "anypoint_team_members" "example_maintainer" {
  provider        = anypoint.admin
  team_id         = anypoint_team.example.id
  user_id         = data.anypoint_users.admin.users[0].id
  membership_type = "maintainer"
}
```

## Schema

### Required

- `team_id` (String) The ID of the team to add the user to.
- `user_id` (String) The ID of the user to add to the team. Use the anypoint_users data source to look up by username.

### Optional

- `membership_type` (String) The membership type. Valid values: 'member' (default), 'maintainer'. Members inherit permissions from the team. Maintainers can additionally manage team membership and child teams.
- `organization_id` (String) The organization ID. Defaults to the provider's org.

### Read-Only

- `id` (String) Unique identifier for this team membership (format: {team_id}:{user_id}).

## Import

An existing team membership can be imported using a composite ID format: `{team_id}:{user_id}`.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_team_members.imported
  id       = "<team_id>:<user_id>"
}

resource "anypoint_team_members" "imported" {
  provider = anypoint.admin
  team_id  = "<team_id>"
  user_id  = "<user_id>"
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
terraform import anypoint_team_members.imported <team_id>:<user_id>
```
