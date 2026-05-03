# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the **Anypoint Platform Terraform Provider** demo example - a comprehensive Infrastructure-as-Code (IaC) solution for managing MuleSoft Anypoint Platform resources. The provider enables automation of the complete API lifecycle including organizations, environments, APIs, policies, networking, security, and secrets management.

## Architecture

### Provider Structure

The codebase follows the Terraform Plugin Framework v1.19+ architecture with a modular design:

```
internal/
├── provider/         # Provider configuration and setup
├── resource/         # All Terraform resource implementations
│   ├── accessmanagement/    # Organizations, environments, users, teams, roles
│   ├── apimanagement/       # API instances, policies, SLA tiers, alerts
│   ├── cloudhub2/           # Private spaces, networks, VPNs, firewalls
│   ├── secretsmanagement/   # Keystores, truststores, certificates, TLS contexts
├── client/           # API client implementations per domain
├── datasource/       # Data source implementations
└── constants/        # Shared constants and enums
```

### Dual Provider Pattern

This provider uses a **dual authentication pattern** to handle different API authorization requirements:

1. **Default Provider** (`provider "anypoint"`): Uses Connected App credentials (OAuth2 client credentials flow) for standard API management operations
2. **Admin Provider** (`provider "anypoint" { alias = "admin" }`): Uses user authentication (username/password with Connected App) for privileged operations requiring user context:
   - Organization and environment creation
   - Connected App scope management
   - User and team management

**Critical**: Resources that manage organizational structure or permissions MUST use `provider = anypoint.admin`.

### Resource Dependencies

The demo follows a specific provisioning order due to API dependencies:

```
1. Organization & Environments (admin provider)
   ↓
2. Connected App Scopes (admin provider) - grants permissions to new org/envs
   ↓
3. Infrastructure (Private Space, Network)
   ↓
4. Secrets Management (Secret Groups, Keystores, Truststores, TLS Contexts)
   ↓
5. Flex Gateway (requires TLS context)
   ↓
6. API Instances (requires gateway)
   ↓
7. Policies & SLA Tiers (requires API instances)
   ↓
8. API Promotion (requires source API with all configurations)
   ↓
9. API Groups (requires API instances)
```

Use explicit `depends_on` to enforce this order when Terraform cannot infer dependencies automatically.

## Development Commands

### Building

```bash
# Build provider for current platform
make build

# Build for all platforms (macOS, Linux, Windows, AMD64, ARM64)
make build-all

# Package providers for distribution
make package-all

# Build and install locally for testing
make install
```

The provider binary is installed to: `~/.terraform.d/plugins/sf.com/mulesoft/anypoint/0.1.0/<platform>/`

### Testing

```bash
# Run unit tests
make test

# Run with coverage report
make test-coverage

# Run acceptance tests (requires valid Anypoint credentials)
make testacc

# Run linter
make lint
```

Acceptance tests require these environment variables:
- `ANYPOINT_CLIENT_ID`
- `ANYPOINT_CLIENT_SECRET`
- `ANYPOINT_USERNAME` (for admin operations)
- `ANYPOINT_PASSWORD` (for admin operations)
- `ANYPOINT_BASE_URL` (defaults to production)

### Terraform Operations (Demo Example)

```bash
cd examples/demo

# Initialize and download provider
terraform init

# Preview changes
terraform plan

# Apply configuration
terraform apply

# Destroy all resources
terraform destroy

# Target specific resources
terraform apply -target=anypoint_organization.commerce_bu
terraform destroy -target=anypoint_api_instance.orders_api
```

### Formatting

```bash
# Format Go code and Terraform files
make fmt
```

## Working with Credentials

### Never Commit Secrets

The `variables.tf` file in this demo contains **placeholder credentials** for demonstration purposes. In production:

1. **Use environment variables**:
   ```bash
   export TF_VAR_anypoint_client_id="your-id"
   export TF_VAR_anypoint_client_secret="your-secret"
   ```

2. **Use terraform.tfvars** (git-ignored):
   ```hcl
   anypoint_client_id     = "actual-client-id"
   anypoint_client_secret = "actual-client-secret"
   anypoint_username      = "actual-username"
   anypoint_password      = "actual-password"
   ```

3. **Use a secrets manager**: AWS Secrets Manager, HashiCorp Vault, etc.

### Environment URLs

- Production US: `https://anypoint.mulesoft.com`
- Production EU: `https://eu1.anypoint.mulesoft.com`
- Government: `https://gov.anypoint.mulesoft.com`
- Staging: `https://stgx.anypoint.mulesoft.com` (used in this demo)

## Key Implementation Patterns

### Resource Lifecycle

All resources follow the Terraform Plugin Framework CRUD pattern:

```go
type ResourceName struct{}

func (r *ResourceName) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse)
func (r *ResourceName) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse)
func (r *ResourceName) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse)
func (r *ResourceName) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse)
```

### Client Architecture

Each domain has a dedicated client package under `internal/client/`:
- API calls are centralized in client methods
- Resources call client methods, never make direct HTTP calls
- Client handles authentication, retries, and error mapping

### Error Handling

Use the standardized error handling from `internal/client/errors.go`:
- Wrap client errors with context
- Map HTTP status codes to Terraform diagnostics
- Provide actionable error messages

## Policy Management

The provider supports managed policies through dedicated resources:

- `anypoint_api_policy_jwt_validation` - JWT token validation
- `anypoint_api_policy_rate_limiting` - Request rate limits
- `anypoint_api_policy_ip_allowlist` - IP-based access control
- And more...

**Policy Order**: The `order` attribute determines policy execution sequence. Lower numbers execute first (1, 2, 3...).

## Secrets Management Workflow

Secrets follow a strict hierarchy:

1. Create `anypoint_secret_group`
2. Add secrets to the group:
   - `anypoint_secret_group_keystore` (PEM, JKS, PKCS12, JCEKS)
   - `anypoint_secret_group_truststore` (PEM, JKS)
   - `anypoint_secret_group_certificate`
3. Reference in TLS contexts:
   - `anypoint_flex_tls_context` (for Flex Gateway)
   - `anypoint_secret_group_tls_context` (for CloudHub)

**Base64 Encoding**: Certificate and key files must be base64-encoded: `base64encode(file("path/to/cert.pem"))`

## API Promotion

Use `anypoint_api_instance_promotion` to promote APIs across environments:

```hcl
resource "anypoint_api_instance_promotion" "api_to_prod" {
  source_api_id    = anypoint_api_instance.source.id
  environment_id   = var.source_environment_id
  instance_label   = "api-production"
  
  include_policies = true  # Copy all policies
  include_tiers    = true  # Copy SLA tiers
  include_alerts   = true  # Copy alert configurations
}
```

This creates a complete copy of the API configuration in the target environment.

## Testing Guidelines

When adding new resources:

1. **Unit tests**: Test schema validation and data transformations
2. **Acceptance tests**: Test full CRUD lifecycle against real API
3. **Use test fixtures**: Reuse common test data from `internal/testutil/`
4. **Clean up**: Always destroy resources in acceptance tests

See `TESTING_FRAMEWORK.md` and `TEST_ORGANIZATION.md` for detailed patterns.

## Common Issues

### Provider Not Found
```bash
# Reinstall provider locally
make install
cd examples/demo
terraform init -upgrade
```

### Authentication Failures
- Verify credentials are valid in the Anypoint Platform UI
- Check that Connected App has required scopes
- Ensure correct `base_url` for your control plane

### Resource Dependencies
- Add explicit `depends_on` when implicit dependencies aren't detected
- Check resource ordering (see "Resource Dependencies" above)
- Review Terraform plan for cycle errors

### State Management
- State files contain sensitive data - never commit `terraform.tfstate`
- Use remote state (S3, Terraform Cloud) for team collaboration
- Lock state during operations to prevent conflicts

## Documentation

Additional documentation in the repository:
- `README.md` - Provider overview and quick start
- `CLIENT_QUICK_START.md` - Client development guide
- `CLIENT_SOP.md` - Standard operating procedures
- `TESTING_FRAMEWORK.md` - Testing patterns and practices
- `examples/` - Resource-specific examples by category
