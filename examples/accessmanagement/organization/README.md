# Organization Management Example

This example demonstrates how to create and manage Anypoint Platform organizations using the Terraform Anypoint provider.

## Prerequisites

1. **Anypoint Platform Account**: You need access to an Anypoint Platform account with organization management permissions.
2. **Connected App Credentials**: Client ID and Client Secret for API access.
3. **User Credentials**: Username and password for user-based authentication (required for organization management).

## Authentication Requirements

Organization management requires **user-based authentication** (password grant flow) in addition to connected app credentials. This is because organization APIs require user-level permissions that are not available with client credentials alone.

### Required Environment Variables

Set the following environment variables before running Terraform:

```bash
# User authentication (required for organization management)
export ANYPOINT_USERNAME="your-username"
export ANYPOINT_PASSWORD="your-password"
```

### Connected App Configuration

Ensure your connected app has the appropriate scopes for organization management:
- `admin:organizations` - For creating and managing organizations

## Configuration

1. **Copy the example configuration**:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. **Update terraform.tfvars** with your actual values:
   - `anypoint_client_id`: Your connected app's client ID
   - `anypoint_client_secret`: Your connected app's client secret
   - `organization_name`: Name for the new organization
   - `parent_organization_id`: UUID of the parent organization
   - `owner_id`: UUID of the user who will own the organization

3. **Set environment variables** for user authentication (see above).

## Configurable Entitlements

This example demonstrates how to configure organization entitlements including:
- **Create Sub-Organizations**: Permission to create child organizations
- **Create Environments**: Permission to create new environments
- **Global Deployment**: Access to global deployment features
- **vCore Allocations**: Production, sandbox, and design vCore assignments
- **Infrastructure**: VPCs, network connections, and managed gateways

## Usage

1. **Initialize Terraform**:
   ```bash
   Not required since we are testing locally.
   ```

2. **Plan the deployment**:
   ```bash
   terraform plan
   ```

3. **Apply the configuration**:
   ```bash
   terraform apply
   ```

## API Request Structure

The organization will be created with the following request payload:
```json
{
  "name": "Your Organization Name",
  "parentOrganizationId": "parent-org-uuid",
  "ownerId": "owner-user-uuid",
  "entitlements": {
    "create_sub_orgs": false,
    "create_environments": false,
    "global_deployment": false,
    "vcores_production": {
      "assigned": 0
    },
    "vcores_sandbox": {
      "assigned": 0
    },
    "vcores_design": {
      "assigned": 0
    },
    "vpcs": {
      "assigned": 0
    },
    "network_connections": {
      "assigned": 0
    },
    "managed_gateway_small": {
      "assigned": 0
    },
    "managed_gateway_large": {
      "assigned": 0
    }
  }
}
```

## Important Notes

- **User Authentication**: Unlike other resources that use client credentials, organization management requires user authentication
- **Environment Variables**: Username and password must be provided via environment variables for security
- **Configurable Entitlements**: Entitlements can be fully configured through Terraform using the `jsonencode()` function
- **Parent Organization**: The parent organization must exist and you must have permissions to create sub-organizations in it
- **Owner User**: The owner user must exist in the Anypoint Platform and have appropriate permissions

## Importing an Existing Organization

If the organization already exists on Anypoint (e.g. created manually via the
UI or by a teammate), you can adopt it into Terraform state without
recreating it.

1. Declare the resource in HCL with the actual `name`, `parent_organization_id`,
   and `owner_id` the server already has. See [`import_example.tf`](./import_example.tf)
   for a commented-out stub — uncomment and edit it to your values.

2. Run `terraform import` with the organization's UUID:

   ```bash
   terraform import anypoint_organization.imported_org <organization-uuid>
   ```

   e.g.

   ```bash
   terraform import anypoint_organization.imported_org a02fab4f-4695-4325-882e-f326d1cef704
   ```

3. Run `terraform plan`. A clean plan means the import succeeded and your HCL
   matches the server. Any remaining diffs fall into one of two buckets:

   - **Entitlements / Read-Only fields**: those hydrate on the first refresh
     from the server, no action needed.
   - **`name` / `parent_organization_id` / `owner_id` differ**: the HCL you
     wrote in step 1 doesn't match the server. Update the HCL — do not apply
     blindly. `parent_organization_id` and `owner_id` have `RequiresReplace`,
     so applying with a mismatched value would destroy and recreate the
     organization.

On the first refresh the provider derives `parent_organization_id` from the
tail of the server-returned ancestor chain (`parent_organization_ids`). If
your organization sits deep in a hierarchy and the derivation picks a
different ancestor than you expected, just put the value you want into HCL
and plan again.

## Troubleshooting

### Authentication Errors
- Ensure `ANYPOINT_USERNAME` and `ANYPOINT_PASSWORD` environment variables are set
- Verify the user has organization management permissions
- Check that the connected app has the required scopes

### Permission Errors
- Verify the user has permissions to create organizations in the parent organization
- Ensure the owner user exists and has appropriate permissions
- Check that the parent organization ID is correct

### API Errors
- Verify all UUIDs are in the correct format
- Ensure the organization name meets platform requirements (minimum 3 characters)
- Check that the parent organization exists and is accessible