---
page_title: "anypoint_teams Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all teams in the organization.
---

# anypoint_teams (Data Source)

Lists all teams in the organization. Use this to find the root team or to look up a team by name.

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

# List all teams in the organization
data "anypoint_teams" "all" {
  provider = anypoint.admin
}

# Output all teams
output "all_teams" {
  value = data.anypoint_teams.all.teams
}

# Find the root team
output "root_team_name" {
  value = [for t in data.anypoint_teams.all.teams : t.name if t.is_root_team][0]
}

# Filter teams by name
data "anypoint_teams" "engineering" {
  provider    = anypoint.admin
  name_filter = "Engineering"
}

# Output filtered teams
output "engineering_teams" {
  value = data.anypoint_teams.engineering.teams
}
```

## Schema

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider's org.
- `name_filter` (String) Optional filter to match teams by name (case-insensitive substring match). If not provided, all teams are returned.

### Read-Only

- `teams` (List of Object) List of teams matching the filter. See [`teams`](#nestedschema--teams) below.

<a id="nestedschema--teams"></a>
### Nested Schema for `teams`

Read-Only:

- `id` (String) The unique team ID.
- `name` (String) The team name. Use this as `parent_team` in other resources when nesting teams.
- `team_type` (String) The type of team (e.g. 'internal').
- `ancestor_team_ids` (List of String) List of ancestor team IDs (empty for the root team).
- `is_root_team` (Boolean) True if this is the organization's root team (has no ancestors). Top-level teams (with `parent_team` omitted) are automatically parented under the root.
- `created_at` (String) When the team was created.
- `updated_at` (String) When the team was last updated.
