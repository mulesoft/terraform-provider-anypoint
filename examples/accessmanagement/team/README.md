# Anypoint Platform Team Management Examples

This directory contains examples for managing teams and team role assignments in Anypoint Platform using the Terraform provider.

## Resources Demonstrated

### anypoint_team Resource

Manages teams in Anypoint Platform with support for:
- Team creation and management
- Team types (internal/external)
- Team membership

### anypoint_team_roles Resource

Manages role assignments for teams in Anypoint Platform with support for:
- Assigning multiple roles to a team
- Context parameters for role assignments (organization, environment, etc.)
- Intelligent updates with proper role removal handling

### anypoint_team_members Resource

Manages team membership in Anypoint Platform with support for:
- Adding users to teams with membership types (member, maintainer)
- Membership type changes for existing users
- Intelligent updates with proper member removal handling
- Computed user details for convenience

## Files

- `main.tf` - Basic team resource configuration
- `datasource.tf` - Team data source examples  
- `variables.tf` - Variable definitions
- `team_roles_example.tf` - Team role assignment examples
- `team_members_example.tf` - Team member management examples

## anypoint_team_roles Resource Schema

```hcl
resource "anypoint_team_roles" "example" {
  team_id = "team-id-here"
  
  roles = [
    {
      role_id = "role-id-here"
      context_params = {
        org = "organization-id"
        env = "environment-id"  # Optional, depends on role
      }
    }
  ]
}
```

### Attributes

- `team_id` (Required) - The ID of the team to assign roles to
- `roles` (Required) - List of role assignments
  - `role_id` (Required) - The ID of the role to assign
  - `context_params` (Optional) - Context parameters for the role assignment

## anypoint_team_members Resource Schema

```hcl
resource "anypoint_team_members" "example" {
  team_id = "team-id-here"
  
  members = [
    {
      id              = "user-id-here"
      membership_type = "member"  # or "maintainer"
    }
  ]
}
```

### Attributes

- `team_id` (Required) - The ID of the team to manage members for
- `members` (Required) - List of team members
  - `id` (Required) - The ID of the user to add to the team
  - `membership_type` (Required) - The membership type (member or maintainer)
- `users` (Computed) - List of team members with full user details
  - `id` - The ID of the user
  - `username` - The username of the user
  - `first_name` - The first name of the user
  - `last_name` - The last name of the user
  - `email` - The email of the user
  - `membership_type` - The membership type of the user

## Usage Examples

### Basic Team Role Assignment

```hcl
resource "anypoint_team_roles" "admin_team" {
  team_id = anypoint_team.my_team.id
  
  roles = [
    {
      role_id = "org-admin-role-id"
      context_params = {
        org = "your-org-id"
      }
    }
  ]
}
```

### Multiple Role Assignment

```hcl
resource "anypoint_team_roles" "dev_team" {
  team_id = anypoint_team.developers.id
  
  roles = [
    {
      role_id = "api-manager-role-id"
      context_params = {
        org = "your-org-id"
      }
    },
    {
      role_id = "environment-admin-role-id"  
      context_params = {
        org = "your-org-id"
        env = "dev-environment-id"
      }
    }
  ]
}
```

### Basic Team Member Management

```hcl
resource "anypoint_team_members" "my_team" {
  team_id = anypoint_team.example.id
  
  members = [
    {
      id              = "user-1-id"
      membership_type = "maintainer"
    },
    {
      id              = "user-2-id"
      membership_type = "member"
    }
  ]
}
```

### Team Members with Computed User Details

```hcl
resource "anypoint_team_members" "project_team" {
  team_id = anypoint_team.project.id
  
  members = [
    {
      id              = "user-1-id"
      membership_type = "maintainer"
    },
    {
      id              = "user-2-id"
      membership_type = "member"
    }
  ]
}

# Access computed user details
output "team_maintainers" {
  value = [
    for user in anypoint_team_members.project_team.users : user
    if user.membership_type == "maintainer"
  ]
}
```

## Import

### Team Resource
```bash
terraform import anypoint_team.example "team-id"
```

### Team Roles Resource
```bash
terraform import anypoint_team_roles.example "team-id"
```

### Team Members Resource
```bash
terraform import anypoint_team_members.example "team-id"
```

## API Mapping

### anypoint_team Resource

- **Create**: `POST /accounts/api/organizations/{orgId}/teams`
- **Read**: `GET /accounts/api/organizations/{orgId}/teams/{teamId}`
- **Update**: `PUT /accounts/api/organizations/{orgId}/teams/{teamId}`
- **Delete**: `DELETE /accounts/api/organizations/{orgId}/teams/{teamId}`

### anypoint_team_roles Resource

- **Create**: `POST /accounts/api/organizations/{orgId}/teams/{teamId}/roles`
- **Read**: `GET /accounts/api/organizations/{orgId}/teams/{teamId}/roles`
- **Update**: `DELETE` (for removed roles) + `POST` (for new role list)
- **Delete**: `DELETE /accounts/api/organizations/{orgId}/teams/{teamId}/roles`

### anypoint_team_members Resource

- **Create**: `PATCH /accounts/api/organizations/{orgId}/teams/{teamId}/members`
- **Read**: `GET /accounts/api/organizations/{orgId}/teams/{teamId}/members`
- **Update**: `DELETE` (for removed members) + `PATCH` (for new member list)
- **Delete**: `DELETE /accounts/api/organizations/{orgId}/teams/{teamId}/members`

## Notes

### anypoint_team_roles Resource Notes

- Each role assignment requires a `role_id` and optional `context_params`
- Context parameters are environment-specific (e.g., organization ID, environment ID)
- Updates are handled intelligently:
  - Roles removed from the `roles` list are explicitly removed via DELETE API
  - Roles added to the `roles` list are assigned via POST API
  - Role comparison includes both `role_id` and `context_params` for accurate matching
  - This ensures proper state synchronization when roles are added or removed
- Role assignments are managed independently from team membership

### anypoint_team_members Resource Notes

- Each member requires a `id` (user ID) and `membership_type` (member or maintainer)
- Membership type changes are handled automatically via the PATCH API
- Updates are handled intelligently:
  - Members removed from the `members` list are explicitly removed via DELETE API
  - Members added to the `members` list are assigned via PATCH API
  - Membership type changes for existing members are handled via PATCH API
  - This ensures proper state synchronization when members are added, removed, or their roles change
- The `users` field provides computed user details (username, email, etc.) fetched from the API for convenience
- Member assignments are managed independently from role assignments

## Running the Examples

1. Set your Anypoint Platform credentials:
   ```bash
   export TF_VAR_anypoint_client_id="your-client-id"
   export TF_VAR_anypoint_client_secret="your-client-secret"
   export TF_VAR_anypoint_base_url="https://anypoint.mulesoft.com"  # or your environment URL
   ```

2. Initialize and apply:
   ```bash   
   terraform plan
   terraform apply
   ```

3. To test updates:
   - Modify the `roles` list in `team_roles_example.tf` to test role changes
   - Modify the `members` list in `team_members_example.tf` to test membership changes
   - Run `terraform apply` again to see the intelligent update behavior

## Example Workflow

```hcl
# 1. Create a team
resource "anypoint_team" "api_team" {
  name = "API Development Team"
  type = "internal"
}

# 2. Assign roles to the team
resource "anypoint_team_roles" "api_team_roles" {
  team_id = anypoint_team.api_team.id
  
  roles = [
    {
      role_id = "api-manager-role-id"
      context_params = {
        org = var.organization_id
      }
    }
  ]
}

# 3. Assign members to the team  
resource "anypoint_team_members" "api_team_members" {
  team_id = anypoint_team.api_team.id
  
  members = [
    {
      id              = "user-1-id"
      membership_type = "maintainer"
    },
    {
      id              = "user-2-id"
      membership_type = "member"
    }
  ]
}

# 4. Update roles and members by modifying the lists
# Terraform will automatically handle additions, removals, and membership changes
``` 