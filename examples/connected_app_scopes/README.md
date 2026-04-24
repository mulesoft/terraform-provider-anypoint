# Connected App Scopes Management Example

This example demonstrates how to manage scopes for an Anypoint Connected Application using the `anypoint_connected_app_scopes` resource with user authentication.

## Prerequisites

- User account with appropriate permissions to manage connected application scopes
- Existing connected application ID
- Target organization ID where scopes will be granted

## Configuration

The example configures a connected app with the following scopes:
- `admin:cloudhub` - Administrative access to CloudHub
- `read:applications` - Read access to applications
- `write:applications` - Write access to applications  
- `admin:runtimefabric` - Administrative access to Runtime Fabric

Each scope is granted for a specific organization using the `context_params.org` parameter.

## Authentication

This resource requires **user authentication** because managing connected app scopes typically requires elevated privileges that are only available through user-based authentication.

The resource will automatically use user authentication with credentials provided via:

### Option 1: Environment Variables (Recommended for CI/CD)
```bash
export ANYPOINT_ADMIN_USERNAME="your-username"
export ANYPOINT_ADMIN_PASSWORD="your-password"
```

### Option 2: Provider Configuration
```hcl
provider "anypoint" {
  auth_type     = "user"
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
  username      = "your-username"
  password      = "your-password"
}
```

The resource uses the `UserAnypointClient` which will:
1. First check the provider configuration for username/password
2. Fall back to `ANYPOINT_USERNAME` and `ANYPOINT_PASSWORD` environment variables
3. Use password grant flow for authentication

## Usage

1. Copy the example terraform.tfvars file:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. Fill in your credentials and configuration:
   ```hcl
   anypoint_client_id     = "your-client-id"
   anypoint_client_secret = "your-client-secret"
   anypoint_username      = "your-username"
   anypoint_password      = "your-password"
   connected_app_id       = "your-connected-app-id"
   target_organization_id = "your-org-id"
   ```

3. Initialize and apply:
   ```bash   
   terraform plan
   terraform apply
   ```

## API Endpoint

This resource manages scopes using the Anypoint Platform API:
- **PATCH** `/accounts/api/connectedApplications/{id}/scopes`

The API expects a payload like:
```json
{
  "scopes": [
    {
      "scope": "admin:cloudhub",
      "context_params": {
        "org": "30aaff6f-c6d0-4555-b19e-ca6c72a6ef60"        
      }
    }
  ]
}
```

**API Response**: The PATCH operation returns a `204 No Content` response with no body. After a successful update, the client automatically fetches the current scopes using a GET request to ensure the Terraform state reflects the actual API state.

## Scope Types

Common scope types include:
- `admin:cloudhub` - CloudHub administrative access
- `read:applications` - Read applications
- `write:applications` - Write applications
- `admin:runtimefabric` - Runtime Fabric administrative access
- `read:environments` - Read environments
- `write:environments` - Write environments

## Notes

- The resource will replace all existing scopes with the ones defined in the configuration
- Removing the resource will clear all scopes from the connected app
- The `connected_app_id` cannot be changed after creation (requires replacement)
- Context parameters are currently limited to organization scoping via the `org` parameter