# Anypoint Provider Authentication Types

This directory contains examples for both authentication methods supported by the Anypoint Terraform Provider.

## Authentication Types

### 1. Connected App Authentication (Client Credentials)

**Path**: `connected_app_on_own_behalf/`

Uses the OAuth 2.0 client credentials flow with a connected app.

**Use Cases**:
- Service-to-service authentication
- CI/CD pipelines
- Automated deployments
- Single organization or pre-scoped connected apps

**Configuration**:
```hcl
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
  # auth_type defaults to "connected_app"
}
```

**Pros**:
- No user credentials needed
- Suitable for automation
- Token doesn't expire based on user sessions

**Cons**:
- Limited to connected app's organization scope
- Requires managing connected app permissions

### 2. User Authentication (Password Grant)

**Path**: `connected_app_on_user_behalf/`

Uses the OAuth 2.0 password grant flow with user credentials.

**Use Cases**:
- Multi-organization access
- Development/testing scenarios
- User-based operations
- Dynamic organization switching

**Configuration**:
```hcl
provider "anypoint" {
  auth_type     = "user"
  client_id     = var.anypoint_client_id     # Connected app that supports password grant
  client_secret = var.anypoint_client_secret
  username      = var.anypoint_username
  password      = var.anypoint_password
  base_url      = var.anypoint_base_url
}
```

**Pros**:
- Access to all organizations user has permissions for
- Can switch between organizations dynamically
- Uses user's active organization by default

**Cons**:
- Requires user credentials
- Less suitable for pure automation
- Token tied to user session

## Key Differences

| Feature | Connected App | User Auth |
|---------|---------------|-----------|
| Grant Type | `client_credentials` | `password` |
| Credentials | Client ID + Secret | Client ID + Secret + Username + Password |
| Organization Access | Single (connected app scope) | Multiple (user's organizations) |
| Automation Friendly | ✅ High | ⚠️ Medium |
| Multi-org Support | ❌ Limited | ✅ Native |
| Security | High (service account) | Medium (user credentials) |

## Organization ID Behavior

### Connected App
- Uses the organization where the connected app was created
- Can specify `organization_id` explicitly if connected app has cross-org permissions

### User Authentication
- Automatically detects user's "active organization" (selected in UI)
- Can specify `organization_id` for any organization user has access to
- Validates user has access before switching context

## Setup Requirements

### Connected App Setup
1. Create connected app in target organization
2. Configure required scopes (e.g., `write:private_spaces`)
3. Note the client ID and secret

### User Authentication Setup
1. Create connected app that supports password grant
2. Ensure connected app has `Resource Owner Password Credentials` grant type enabled
3. User must have appropriate permissions in target organizations

## Usage Examples

### Create resources in multiple organizations (User Auth)
```hcl
# Get all accessible organizations
data "anypoint_organizations" "accessible" {}

# Create spaces in all organizations
resource "anypoint_private_space" "multi_org" {
  for_each = data.anypoint_organizations.accessible.organizations
  
  name            = "${each.value.name}-space"
  region          = "us-east-1"
  organization_id = each.value.id
}
```

### Explicit organization targeting (Both auth types)
```hcl
resource "anypoint_private_space" "specific_org" {
  name            = "my-space"
  region          = "us-east-1"
  organization_id = "080f1918-0096-4cac-85b5-b1cd9cdf9260"
}
```

## Security Recommendations

### For Production (Connected App)
- Use connected app authentication
- Rotate credentials regularly
- Limit connected app scopes to minimum required
- Store credentials in secure credential management system

### For Development (User Auth)
- Use user authentication for development/testing
- Enable MFA on user accounts
- Avoid hardcoding credentials
- Use environment variables or secure credential storage

## Migration Path

1. **Start with User Auth**: For POC and development to access multiple organizations easily
2. **Create Organization-Specific Connected Apps**: As you productionize specific organization workflows
3. **Implement Provider Aliases**: Use multiple providers for different organizations in production