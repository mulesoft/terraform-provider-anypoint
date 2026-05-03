# Anypoint Platform Team Management Examples

This directory contains examples for managing teams in Anypoint Platform using the Terraform provider.

## Resources Demonstrated

### anypoint_team Resource

Manages teams in Anypoint Platform with support for:
- Team creation and management
- Team types (internal/external)

## Files

- `main.tf` - Basic team resource configuration
- `datasource.tf` - Team data source examples
- `variables.tf` - Variable definitions
- `team_example.tf` - Team resource examples

## Usage Examples

### Basic Team Creation

```hcl
resource "anypoint_team" "my_team" {
  team_name = "My Team"
  team_type = "internal"
}
```

### Team with Parent

```hcl
resource "anypoint_team" "parent" {
  team_name = "Parent Team"
  team_type = "internal"
}

resource "anypoint_team" "child" {
  team_name      = "Child Team"
  team_type      = "internal"
  parent_team_id = anypoint_team.parent.id
}
```

## Import

### Team Resource
```bash
terraform import anypoint_team.example "team-id"
```

## API Mapping

### anypoint_team Resource

- **Create**: `POST /accounts/api/organizations/{orgId}/teams`
- **Read**: `GET /accounts/api/organizations/{orgId}/teams/{teamId}`
- **Update**: `PUT /accounts/api/organizations/{orgId}/teams/{teamId}`
- **Delete**: `DELETE /accounts/api/organizations/{orgId}/teams/{teamId}`

## Running the Examples

1. Set your Anypoint Platform credentials:
   ```bash
   export TF_VAR_anypoint_client_id="your-client-id"
   export TF_VAR_anypoint_client_secret="your-client-secret"
   export TF_VAR_anypoint_base_url="https://anypoint.mulesoft.com"
   ```

2. Initialize and apply:
   ```bash
   terraform plan
   terraform apply
   ```
