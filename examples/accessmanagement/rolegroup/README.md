# Anypoint Platform Role Group Resources

This example demonstrates how to use the `anypoint_rolegroup`, `anypoint_rolegroup_roles`, and `anypoint_rolegroup_users` resources to manage role groups, role assignments, and user assignments in Anypoint Platform.

## Features

The `anypoint_rolegroup` resource supports full CRUD operations:
- ✅ **Create**: Create new role groups with optional external names
- ✅ **Read**: Retrieve role group details and state refresh
- ✅ **Update**: Modify name, description, and external names
- ✅ **Delete**: Remove role groups
- ✅ **Import**: Import existing role groups by ID

## Resource Schema

### Required Arguments

- `name` (String) - The name of the role group
- `description` (String) - The description of the role group

### Optional Arguments

- `external_names` (List of Objects) - List of external group mappings
  - `external_group_name` (String, Required) - The external group name
  - `provider_id` (String, Required) - The provider ID for the external group

### Computed Arguments

- `id` (String) - Unique identifier for the role group
- `org_id` (String) - Organization ID where the role group belongs
- `editable` (Boolean) - Whether the role group can be edited
- `created_at` (String) - Creation timestamp
- `updated_at` (String) - Last update timestamp

## anypoint_rolegroup_roles Resource Schema

### Required Arguments

- `rolegroup_id` (String) - The ID of the role group to assign roles to
- `roles` (List of Objects) - List of role assignments
  - `role_id` (String, Required) - The ID of the role to assign
  - `context_params` (Map of Strings, Required) - Context parameters for the role (e.g., org, env)

### Computed Arguments

- `id` (String) - Unique identifier for this resource (same as rolegroup_id)

## anypoint_rolegroup_users Resource Schema

### Required Arguments

- `rolegroup_id` (String) - The ID of the role group to assign users to
- `user_ids` (List of Strings) - List of user IDs to assign to the role group

### Computed Arguments

- `id` (String) - Unique identifier for this resource (same as rolegroup_id)
- `users` (List of Objects) - List of users assigned to the role group with full details
  - `id` (String) - The user ID
  - `username` (String) - The username  
  - `first_name` (String) - The user's first name
  - `last_name` (String) - The user's last name
  - `email` (String) - The user's email address
  - `organization_id` (String) - The organization ID
  - `enabled` (Boolean) - Whether the user is enabled
  - `idprovider_id` (String) - The identity provider ID

## Usage Examples

### Basic Role Group

```hcl
resource "anypoint_rolegroup" "simple_example" {
  name        = "Organization Administrators"
  description = "Administrators for the organization"
}
```

### Role Group with External Names

```hcl
resource "anypoint_rolegroup" "external_example" {
  name        = "External Administrators"
  description = "External group administrators"
  
  external_names = [
    {
      external_group_name = "administrators"
      provider_id         = "2e50e859-0042-46ff-8cf8-1ad6f0c78b67"
    },
    {
      external_group_name = "admins"
      provider_id         = "2e50e859-0042-46ff-8cf8-1ad6f0c78b67"
    }
  ]
}
```

### Update Operations

The resource supports in-place updates for all fields. For example, to update the description:

```hcl
resource "anypoint_rolegroup" "example" {
  name        = "Organization Administrators"
  description = "Updated description for administrators" # This will trigger an update
}
```

### Role Assignments

Use the `anypoint_rolegroup_roles` resource to manage role assignments:

```hcl
resource "anypoint_rolegroup_roles" "example" {
  rolegroup_id = anypoint_rolegroup.example.id
  
  roles = [
    {
      role_id = "d74ef94a-4292-4896-b860-b05bd7f90d6d"
      context_params = {
        org = "68ef9520-24e9-4cf2-b2f5-620025690913"
      }
    },
    {
      role_id = "e85f0a5b-5393-5907-c971-c16ce6a95e7e"
      context_params = {
        org = "68ef9520-24e9-4cf2-b2f5-620025690913"
        env = "production"
      }
    }
  ]
}
```

### User Assignments

Use the `anypoint_rolegroup_users` resource to manage user assignments:

```hcl
resource "anypoint_rolegroup_users" "example" {
  rolegroup_id = anypoint_rolegroup.example.id
  
  user_ids = [
    "1c0d1a43-4d91-4b52-bcc6-cff8ecf0c3a3",
    "dc52bf66-b0d8-4cda-9ae8-45a5f085ead4",
    "5314c476-1cd4-49be-ba4e-1d67f6c052c3"
  ]
}

# Access computed user details
output "user_details" {
  value = anypoint_rolegroup_users.example.users
}

# Example: To update user assignments, just modify the user_ids list
# Terraform will automatically:
# 1. Remove users no longer in the list via DELETE API
# 2. Add new users via POST API
# 3. Keep existing users unchanged
```

### Import Existing Resources

```bash
# Import role group
terraform import anypoint_rolegroup.example "67f3b9a6-75f1-4b94-af44-7fbe2a9b7b57"

# Import role assignments (use the rolegroup_id)
terraform import anypoint_rolegroup_roles.example "67f3b9a6-75f1-4b94-af44-7fbe2a9b7b57"

# Import user assignments (use the rolegroup_id)
terraform import anypoint_rolegroup_users.example "67f3b9a6-75f1-4b94-af44-7fbe2a9b7b57"
```

## API Mapping

### anypoint_rolegroup Resource

- **Create**: `POST /accounts/api/organizations/{orgId}/rolegroups`
- **Read**: `GET /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}`
- **Update**: `PUT /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}`
- **Delete**: `DELETE /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}`

### anypoint_rolegroup_roles Resource

- **Create**: `POST /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}/roles`
- **Read**: `GET /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}/roles`
- **Update**: `DELETE` (for removed roles) + `POST` (for new role list)
- **Delete**: `DELETE /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}/roles`

### anypoint_rolegroup_users Resource

- **Create**: `POST /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}/users`
- **Read**: `GET /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}/users`
- **Update**: `DELETE` (for removed users) + `POST` (for new user list)
- **Delete**: `DELETE /accounts/api/organizations/{orgId}/rolegroups/{roleGroupId}/users`

## Notes

- The API response for `external_names` returns simple strings, while the request expects objects with `external_group_name` and `provider_id`. The provider handles this mapping automatically.
- Updates require all fields to be specified (full replacement), not partial updates.
- The `provider_id` field in `external_names` may not be preserved in the state after creation/update due to API response format limitations.

### anypoint_rolegroup_roles Resource Notes

- Each role assignment requires a `role_id` and optional `context_params`.
- Context parameters are environment-specific (e.g., organization ID).
- Updates are handled intelligently:
  - Roles removed from the `roles` list are explicitly removed via DELETE API
  - Roles added to the `roles` list are assigned via POST API
  - Role comparison includes both `role_id` and `context_params` for accurate matching
  - This ensures proper state synchronization when roles are added or removed
- Role assignments are managed independently from user assignments.

### anypoint_rolegroup_users Resource Notes

- The API request expects a simple array of user IDs, while the response returns full user objects with details.
- Updates are handled intelligently:
  - Users removed from the `user_ids` list are explicitly removed via DELETE API
  - Users added to the `user_ids` list are assigned via POST API
  - This ensures proper state synchronization when users are added or removed
- The `users` field provides computed user details fetched from the API for convenience.
- User assignments are managed independently from role assignments.

## Running the Example

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

3. To test updates, modify the configuration and run `terraform apply` again. 