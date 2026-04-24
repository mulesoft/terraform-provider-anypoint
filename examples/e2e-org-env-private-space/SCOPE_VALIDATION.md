# Scope Validation Implementation

This document describes the scope validation feature added to the `anypoint_connected_app_scopes` resource.

## Overview

The scope validation feature ensures that only valid Anypoint Platform scope names are accepted when configuring connected application scopes. This prevents typos and configuration errors before they reach the API.

## Implementation Details

### 1. Constants Package (`/internal/constants/scopes.go`)

Created a new constants package that defines:
- **47 scope constants** for all valid Anypoint Platform scopes
- **ValidScopes map** for efficient O(1) lookup validation
- **IsValidScope()** function to validate scope names
- **GetAllScopes()** function to retrieve all valid scopes

Example constants:
```go
const (
    ScopeAdminCloudHub         = "admin:cloudhub"
    ScopeManageRuntimeFabrics  = "manage:runtime_fabrics"
    ScopeCreateEnvironment     = "create:environment"
    ScopeManagePrivateSpaces   = "manage:private_spaces"
    ScopeAdminAPIManager       = "admin:api_manager"
    // ... and 42 more
)
```

### 2. Validation in Resource (`/internal/resource/accessmanagement/connectedappscopes.go`)

Added validation to the `anypoint_connected_app_scopes` resource:

#### Changes Made:
1. **Import constants package** for scope validation
2. **Added `validateScopes()` method** that checks each scope in the set
3. **Integrated validation** into Create() and Update() methods
4. **Error messages** include the invalid scope name and suggestions for valid scopes

#### Validation Flow:
```
User Configuration → Terraform Plan → validateScopes() → API Call
                                            ↓
                                    (Rejects invalid scopes)
```

### 3. Comprehensive Tests

Added extensive test coverage in `/internal/resource/accessmanagement/connectedappscopes_test.go`:

- ✅ **TestConnectedAppScopesResource_validateScopes_ValidScopes** - Tests valid scopes pass
- ✅ **TestConnectedAppScopesResource_validateScopes_InvalidScopes** - Tests invalid scopes fail
- ✅ **TestConnectedAppScopesResource_validateScopes_MixedScopes** - Tests mixed valid/invalid
- ✅ **TestConnectedAppScopesResource_validateScopes_AllKnownScopes** - Tests all 47 constants

## Valid Scope Categories

### Admin Scopes (7)
- `admin:ang_governance_profiles`
- `admin:api_manager`
- `admin:api_query`
- `admin:cloudhub`
- `admin:data_exporter_configurations`
- `admin:data_exporter_connections`
- `admin:partner_manager`

### Create Scopes (3)
- `create:environment`
- `create:exchange_genai`
- `create:generations`

### Manage Scopes (11)
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

### Read Scopes (8)
- `read:activity`
- `read:api_query`
- `read:data_gateway`
- `read:host_partners`
- `read:stats`
- `read:store`
- `read:store_clients`
- `read:store_metrics`

### Edit Scopes (5)
- `edit:api_catalog`
- `edit:api_query`
- `edit:monitoring`
- `edit:rpa`
- `edit:visualizer`

### View Scopes (5)
- `view:ang_governance_profiles`
- `view:clients`
- `view:destinations`
- `view:metering`
- `view:monitoring`

### Other Scopes (8)
- `administer:destinations`
- `aeh_admin`
- `clear:destinations`
- `execute:document_actions`
- `promote:api_query`
- `publish:destinations`
- `restart:applications`
- `subscribe:destinations`

## Usage Example

### Valid Configuration
```hcl
resource "anypoint_connected_app_scopes" "app_scopes" {
  provider = anypoint.admin

  connected_app_id = var.connected_app_client_id

  scopes = [
    {
      scope = "admin:cloudhub"  # ✅ Valid
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    },
    {
      scope = "manage:runtime_fabrics"  # ✅ Valid
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    }
  ]
}
```

### Invalid Configuration (Will Be Rejected)
```hcl
resource "anypoint_connected_app_scopes" "app_scopes" {
  provider = anypoint.admin

  connected_app_id = var.connected_app_client_id

  scopes = [
    {
      scope = "admin:cloudhb"  # ❌ Invalid - typo in "cloudhub"
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    }
  ]
}
```

### Error Message Example
```
Error: Invalid Scope Name

The scope 'admin:cloudhb' at index 0 is not a valid Anypoint Platform scope.
Please check the scope name for typos. Valid scopes include: admin:cloudhub,
manage:runtime_fabrics, create:environment, manage:private_spaces,
admin:api_manager, read:api_query, edit:api_query, manage:api_query, etc.
For a complete list of valid scopes, see the provider documentation.
```

## Benefits

1. **Early Error Detection** - Catches typos and invalid scopes before API calls
2. **Better User Experience** - Clear error messages with suggestions
3. **Documentation** - Constants serve as documentation for valid scopes
4. **Type Safety** - Use constants instead of strings for better IDE support
5. **Maintainability** - Single source of truth for valid scope names

## Testing

Run tests with:
```bash
# Test constants package
go test ./internal/constants/... -v

# Test resource validation
go test ./internal/resource/accessmanagement/... -v -run TestConnectedAppScopesResource_validateScopes
```

## Future Enhancements

Potential improvements:
- Add scope descriptions/documentation in constants
- Group scopes by service/feature
- Add validation for required context_params based on scope type
- Create custom Terraform validators for compile-time checking
- Add scope suggestion based on Levenshtein distance for typos

## Files Modified/Created

### Created:
- `/internal/constants/scopes.go` - Scope constants and validation functions
- `/internal/constants/scopes_test.go` - Comprehensive tests for constants
- `/internal/constants/README.md` - Documentation for constants package
- `/examples/e2e/SCOPE_VALIDATION.md` - This document

### Modified:
- `/internal/resource/accessmanagement/connectedappscopes.go` - Added validation
- `/internal/resource/accessmanagement/connectedappscopes_test.go` - Added validation tests

## Summary

The scope validation feature provides robust validation for Anypoint Platform scope names, preventing configuration errors and improving the user experience. All 47 valid scopes are now defined as constants and validated before API calls.
