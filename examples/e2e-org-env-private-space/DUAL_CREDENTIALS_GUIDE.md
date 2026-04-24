# Using Multiple Credentials in Terraform

This guide explains how to use two different sets of credentials (admin and normal user) in a single Terraform script for e2e testing.

## Overview

The Terraform configuration now supports two provider instances using **provider aliases**:

1. **`anypoint.admin`** - Admin credentials with user authentication (username/password)
   - Used for privileged operations: organization creation, scope assignment
   - Requires: `client_id`, `client_secret`, `username`, `password`

2. **`anypoint.normal_user`** - Normal user credentials with connected app authentication
   - Used for standard operations: private spaces, networks, etc.
   - Requires: `client_id`, `client_secret`

## Configuration

### Option 1: Using terraform.tfvars

Create a `terraform.tfvars` file:

```hcl
# Admin credentials
anypoint_admin_client_id     = "a66da37ba83d4c599264347952d4d533"
anypoint_admin_client_secret = "your-admin-secret"
anypoint_admin_username      = "admin@example.com"
anypoint_admin_password      = "admin-password"

# Normal user credentials
anypoint_normal_client_id     = "b77ea48ca94e5f3a9f72ba9561914644"
anypoint_normal_client_secret = "your-normal-secret"

# Common
anypoint_base_url = "https://stgx.anypoint.mulesoft.com"
```

### Option 2: Using Environment Variables

```bash
# Admin credentials
export TF_VAR_anypoint_admin_client_id="a66da37ba83d4c599264347952d4d533"
export TF_VAR_anypoint_admin_client_secret="your-admin-secret"
export TF_VAR_anypoint_admin_username="admin@example.com"
export TF_VAR_anypoint_admin_password="admin-password"

# Normal user credentials
export TF_VAR_anypoint_normal_client_id="b77ea48ca94e5f3a9f72ba9561914644"
export TF_VAR_anypoint_normal_client_secret="your-normal-secret"

# Common
export TF_VAR_anypoint_base_url="https://stgx.anypoint.mulesoft.com"
```

## Resource Provider Assignment

Each resource explicitly declares which provider to use:

```hcl
# Admin operations
resource "anypoint_organization" "sub_org" {
  provider = anypoint.admin
  # ...
}

resource "anypoint_connected_app_scopes" "app_scopes" {
  provider = anypoint.admin
  # ...
}

# Normal user operations (after scopes are granted)
resource "anypoint_private_space" "production_space" {
  provider = anypoint.normal_user
  # ...
}
```

## Use Cases

### E2E Testing Scenario

1. **Setup phase** (using admin credentials):
   - Create sub-organization
   - Create environments
   - Assign scopes to normal user's connected app

2. **Test phase** (using normal user credentials):
   - Create private spaces (validates scope assignment)
   - Create private networks
   - Deploy applications

### Single Credential Setup

If you only have admin credentials, you can:

1. Comment out the `normal_user` provider block
2. Change all `provider = anypoint.normal_user` to `provider = anypoint.admin`
3. Leave the normal user variables empty or remove them

## Running Terraform

```bash
# Initialize
terraform init

# Plan (review what will be created)
terraform plan

# Apply (create resources)
terraform apply

# Destroy (clean up)
terraform destroy
```

## Troubleshooting

### Authentication Errors

- **Admin provider**: Ensure username/password are correct and user has admin privileges
- **Normal user provider**: Ensure connected app has appropriate scopes assigned

### Provider Not Found

If you see "provider not found" errors for `anypoint.normal_user`:
- Check that the provider block is uncommented
- Verify credentials are provided
- Consider switching to `anypoint.admin` if you don't need separate credentials

## Benefits of This Approach

✅ **Clear separation** - Each provider instance has distinct credentials
✅ **Explicit control** - Each resource declares which provider to use
✅ **Standard pattern** - Follows Terraform best practices
✅ **Flexible** - Easy to add more provider instances if needed
✅ **Testable** - Perfect for e2e tests validating different permission levels
