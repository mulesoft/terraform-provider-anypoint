---
page_title: "anypoint_teams Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all teams in the organization.
---

# anypoint_teams (Data Source)

Lists all teams in the organization. Use this to find the root team or to look up a team by name.

-> **Authentication:** This data source uses client_credentials authentication via the connected app configured in the provider block. No username/password is required.

## Example Usage

```terraform
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# List all teams in the organization
data "anypoint_teams" "all" {}

# Find the root team ID (use as parent_team_id when creating top-level teams)
output "root_team_id" {
  value = [for t in data.anypoint_teams.all.teams : t.id if t.is_root_team][0]
}

# Filter teams by name
data "anypoint_teams" "engineering" {
  name_filter = "Engineering"
}

# Use a team's ID as parent_team_id in a team resource
resource "anypoint_team" "sub_team" {
  name           = "Sub Team"
  parent_team_id = data.anypoint_teams.engineering.teams[0].id
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
- `name` (String) The team name.
- `team_type` (String) The type of team (e.g. 'internal').
- `ancestor_team_ids` (List of String) List of ancestor team IDs (empty for the root team).
- `is_root_team` (Boolean) True if this is the organization's root team (has no ancestors). Use this team's ID as `parent_team_id` when creating top-level teams.
- `created_at` (String) When the team was created.
- `updated_at` (String) When the team was last updated.
