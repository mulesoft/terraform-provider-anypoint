# Anypoint Platform Scope Constants

This package contains constants for all valid Anypoint Platform scope names used in Connected Applications.

## Overview

The `scopes.go` file defines:
- **Constants** for all valid scope names
- **ValidScopes** map for validation
- **IsValidScope()** function to check if a scope is valid
- **GetAllScopes()** function to retrieve all valid scopes

## Usage

### Validating Scope Names

```go
import "github.com/mulesoft/terraform-provider-anypoint/internal/constants"

// Check if a scope is valid
if constants.IsValidScope("admin:cloudhub") {
    // Scope is valid
}

// Using constants
if constants.IsValidScope(constants.ScopeAdminCloudHub) {
    // Scope is valid
}
```

### Using Scope Constants

```go
import "github.com/mulesoft/terraform-provider-anypoint/internal/constants"

// Use in resource definitions
scope := constants.ScopeAdminCloudHub         // "admin:cloudhub"
scope := constants.ScopeManageRuntimeFabrics  // "manage:runtime_fabrics"
scope := constants.ScopeCreateEnvironment     // "create:environment"
```

### Getting All Scopes

```go
import "github.com/mulesoft/terraform-provider-anypoint/internal/constants"

// Get all valid scopes
allScopes := constants.GetAllScopes()
for _, scope := range allScopes {
    fmt.Println(scope)
}
```

## Scope Categories

Scopes are organized by action prefix:

### Admin Scopes
Full administrative access to resources:
- `admin:ang_governance_profiles`
- `admin:api_manager`
- `admin:api_query`
- `admin:cloudhub`
- `admin:data_exporter_configurations`
- `admin:data_exporter_connections`
- `admin:partner_manager`

### Create Scopes
Permission to create new resources:
- `create:environment`
- `create:exchange_genai`
- `create:generations`

### Manage Scopes
Permission to manage existing resources:
- `manage:activity`
- `manage:api_query`
- `manage:clients`
- `manage:data_gateway`
- `manage:host`
- `manage:partners`
- `manage:private_spaces`
- `manage:runtime_fabrics`
- `manage:store`
- `manage:store_clients`
- `manage:store_data`

### Read Scopes
Read-only access to resources:
- `read:activity`
- `read:api_query`
- `read:data_gateway`
- `read:host_partners`
- `read:stats`
- `read:store`
- `read:store_clients`
- `read:store_metrics`

### Edit Scopes
Permission to modify resources:
- `edit:api_catalog`
- `edit:api_query`
- `edit:monitoring`
- `edit:rpa`
- `edit:visualizer`

### View Scopes
View access (typically for UI):
- `view:ang_governance_profiles`
- `view:clients`
- `view:destinations`
- `view:metering`
- `view:monitoring`

### Other Scopes
- `administer:destinations`
- `aeh_admin`
- `clear:destinations`
- `execute:document_actions`
- `promote:api_query`
- `publish:destinations`
- `restart:applications`
- `subscribe:destinations`

## Testing

Run tests with:
```bash
go test ./internal/constants/... -v
```

## Integration Example

Example usage in the Connected App Scopes resource:

```go
import (
    "github.com/mulesoft/terraform-provider-anypoint/internal/constants"
)

func (r *ConnectedAppScopesResource) validateScope(scope string) error {
    if !constants.IsValidScope(scope) {
        return fmt.Errorf("invalid scope: %s", scope)
    }
    return nil
}
```

## Scope Format

All scopes follow the format: `action:resource`

Where:
- **action**: The operation type (admin, create, read, edit, manage, view, etc.)
- **resource**: The platform resource or service (cloudhub, api_manager, environment, etc.)

## Context Parameters

Scopes often require context parameters:
- `org`: Organization ID
- `envId`: Environment ID (for environment-specific scopes)

Example in Terraform:
```hcl
resource "anypoint_connected_app_scopes" "example" {
  connected_app_id = var.connected_app_id

  scopes = [
    {
      scope = "admin:cloudhub"
      context_params = {
        org = var.org_id
      }
    }
  ]
}
```
