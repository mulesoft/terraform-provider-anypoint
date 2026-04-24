# Anypoint Private Space Association Resource and Data Source

This resource creates and manages associations between a CloudHub 2.0 private space and environments. The data source fetches existing associations for a private space.

## API Reference

**Create Method:** POST  
**Create URL:** `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/associations`

**Delete Method:** DELETE  
**Delete URL:** `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/associations/{associationId}`

### Create Request Payload

```json
{
  "associations": [
    {
      "organizationId": "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b",
      "environment": "2ea3eff9-569a-4495-b7b6-1e28c6440aeb"
    },
    {
      "organizationId": "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b",
      "environment": "3e9d5bde-6ef7-4947-9f98-0f7c32d48909"
    }
  ]
}
```

### Create Response

The API returns an array of created associations:

```json
[
  {
    "id": "d282e462-07af-4146-b7d8-8bd7e6b18d83",
    "environmentId": "2ea3eff9-569a-4495-b7b6-1e28c6440aeb",
    "organizationId": "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b"
  },
  {
    "id": "79810eb2-c999-4104-b650-31ea2962a200",
    "environmentId": "3e9d5bde-6ef7-4947-9f98-0f7c32d48909",
    "organizationId": "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b"
  }
]
```

### Delete Response

The DELETE API returns:
- **Status 200 OK** or **Status 204 No Content** on successful deletion
- **Status 404 Not Found** if the association doesn't exist (treated as success)
- **Error status codes** with error details for failures

## Usage

```hcl
resource "anypoint_private_space_association" "example" {
  private_space_id = var.private_space_id
  
  associations = [
    {
      organization_id = "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b"
      environment_id  = "2ea3eff9-569a-4495-b7b6-1e28c6440aeb"
    },
    {
      organization_id = "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b"
      environment_id  = "3e9d5bde-6ef7-4947-9f98-0f7c32d48909"
    }
  ]
}
```

## Configuration Arguments

### Required Arguments

- `private_space_id` - (Required) The ID of the private space where associations will be created
- `associations` - (Required) List of associations to create. Each association contains:
  - `organization_id` - (Required) The organization ID for the association
  - `environment_id` - (Required) The environment ID for the association

## Computed Attributes

- `id` - The unique identifier for the Private Space Association resource
- `created_associations` - List of created associations with their details:
  - `id` - The ID of the created association
  - `organization_id` - The organization ID of the association
  - `environment_id` - The environment ID of the association

## Example

```hcl
terraform {
  required_providers {
    anypoint = {
      source = "example.com/ankitsarda/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

resource "anypoint_private_space_association" "example" {
  private_space_id = var.private_space_id
  
  associations = [
    {
      organization_id = var.organization_id
      environment_id  = var.environment_id_1
    },
    {
      organization_id = var.organization_id
      environment_id  = var.environment_id_2
    }
  ]
}

# Access created associations
output "association_ids" {
  value = [for assoc in anypoint_private_space_association.example.created_associations : assoc.id]
}

output "environment_associations" {
  value = {
    for assoc in anypoint_private_space_association.example.created_associations : 
    assoc.environment_id => assoc.id
  }
}
```

## Import

Private Space Associations can be imported using the private space ID:

```sh
terraform import anypoint_private_space_association.example private-space-id
```

## Notes

- This resource creates multiple associations in a single API call
- **Create and Delete**: Supports creating associations in bulk and deleting them individually
- **No Updates**: Updates require recreation of the resource (delete + create)
- **No Individual Management**: Individual associations cannot be managed separately - they are managed as a group
- The resource generates a unique ID based on the private space ID and number of associations
- All associations are created atomically in a single API request
- Deletion removes each association individually, which may result in partial failures

## Lifecycle Management

### Create
- Sends all associations to the API in a single request
- Maps the response to store individual association IDs

### Read
- Currently maintains existing state (no read API implemented yet)
- TODO: Implement read logic when API becomes available

### Update
- **Not Supported**: Updates will result in an error
- To modify associations, delete and recreate the resource

### Delete
- Deletes all associations individually using the DELETE API
- Each association is deleted by its ID using the endpoint: `DELETE /organizations/{orgId}/privatespaces/{privateSpaceId}/associations/{associationId}`
- If any association fails to delete, the operation reports an error with details of all failed deletions
- Successfully deleted associations are removed from the private space

## Common Use Cases

1. **Environment Association**: Associate multiple environments with a private space for deployment
2. **Multi-Environment Setup**: Configure development, staging, and production environments together
3. **Organizational Setup**: Associate environments from the same organization with a private space
4. **Bulk Association**: Create multiple associations efficiently in a single operation

## Prerequisites

1. **Private Space**: Must have an existing CloudHub 2.0 private space
2. **Environment Access**: Must have access to the environments being associated
3. **Organization Permissions**: Must have proper permissions in the target organization
4. **Valid Environment IDs**: Environment IDs must exist and be accessible

## Monitoring and Troubleshooting

Monitor created associations through outputs:

```hcl
# Check number of associations created
output "association_count" {
  value = length(anypoint_private_space_association.example.created_associations)
}

# List all association IDs
output "all_associations" {
  value = anypoint_private_space_association.example.created_associations
}

# Create lookup map
output "environment_lookup" {
  value = {
    for assoc in anypoint_private_space_association.example.created_associations : 
    assoc.environment_id => {
      association_id  = assoc.id
      organization_id = assoc.organization_id
    }
  }
}
```

## Deletion Behavior

When you run `terraform destroy` or remove the resource from your configuration:

1. **Individual Deletion**: Each association is deleted individually using its ID
2. **Error Handling**: If any association fails to delete, the operation reports errors with details
3. **Partial Failures**: Some associations may be deleted while others fail
4. **Idempotent**: Attempting to delete an already-deleted association (404 Not Found) is treated as success

Example of potential delete error output:
```
Error: Error deleting Private Space Associations

Could not delete some Private Space Associations:
Failed to delete association d282e462-07af-4146-b7d8-8bd7e6b18d83: failed to delete private space association with status 500: Internal Server Error
Failed to delete association 79810eb2-c999-4104-b650-31ea2962a200: failed to delete private space association with status 403: Forbidden
```

## Data Source: anypoint_private_space_associations

The `anypoint_private_space_associations` data source fetches all existing associations for a CloudHub 2.0 private space.

### API Reference

**Method:** GET  
**URL:** `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/associations`

### Response

The API returns an array of existing associations:

```json
[
  {
    "id": "d282e462-07af-4146-b7d8-8bd7e6b18d83",
    "environmentId": "2ea3eff9-569a-4495-b7b6-1e28c6440aeb",
    "organizationId": "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b"
  },
  {
    "id": "79810eb2-c999-4104-b650-31ea2962a200",
    "environmentId": "3e9d5bde-6ef7-4947-9f98-0f7c32d48909",
    "organizationId": "1ffdaa42-4702-4747-aeb9-aecab0f5ac1b"
  }
]
```

### Usage

```hcl
data "anypoint_private_space_associations" "example" {
  private_space_id = var.private_space_id
}
```

### Configuration Arguments

#### Required Arguments

- `private_space_id` - (Required) The ID of the private space to fetch associations for

### Computed Attributes

- `id` - The unique identifier for the data source
- `associations` - List of associations with their details:
  - `id` - The ID of the association
  - `organization_id` - The organization ID of the association
  - `environment_id` - The environment ID of the association

### Example

```hcl
terraform {
  required_providers {
    anypoint = {
      source = "example.com/ankitsarda/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Fetch all associations for a private space
data "anypoint_private_space_associations" "example" {
  private_space_id = var.private_space_id
}

# Output all associations
output "associations" {
  value = data.anypoint_private_space_associations.example.associations
}

# Filter associations by environment ID
locals {
  production_associations = [
    for association in data.anypoint_private_space_associations.example.associations :
    association if association.environment_id == var.environment_id_1
  ]
}

# Output filtered associations
output "production_associations" {
  value = local.production_associations
}

# Create a map of environment ID to association ID
output "environment_association_map" {
  value = {
    for assoc in data.anypoint_private_space_associations.example.associations : 
    assoc.environment_id => assoc.id
  }
}
```

### Common Use Cases

1. **Audit Associations**: Check which environments are associated with a private space
2. **Conditional Logic**: Use existing associations to make decisions about new resources
3. **Data Validation**: Verify that expected associations exist before creating dependent resources
4. **Environment Mapping**: Create mappings between environments and their associations

### Working with the Data Source

The data source can be used in combination with the resource for advanced scenarios:

```hcl
# First, fetch existing associations
data "anypoint_private_space_associations" "current" {
  private_space_id = var.private_space_id
}

# Check if a specific environment is already associated
locals {
  is_env_associated = length([
    for assoc in data.anypoint_private_space_associations.current.associations :
    assoc if assoc.environment_id == var.new_environment_id
  ]) > 0
}

# Conditionally create associations if environment is not already associated
resource "anypoint_private_space_association" "conditional" {
  count = local.is_env_associated ? 0 : 1
  
  private_space_id = var.private_space_id
  
  associations = [
    {
      organization_id = var.organization_id
      environment_id  = var.new_environment_id
    }
  ]
}
```

## Limitations

- **No Individual Updates**: Cannot update individual associations
- **No Partial Operations**: All associations are managed together
- **Create Only**: Currently limited to create operations
- **No Validation**: No client-side validation of organization/environment IDs
- **No Filtering**: Cannot filter or conditionally create associations 