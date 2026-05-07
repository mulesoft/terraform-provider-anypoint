# Sub-Organization with Private Space - Complete Setup Guide

## Overview

This example demonstrates a complete end-to-end flow for setting up a new sub-organization within an Anypoint Platform parent organization (Salesforce org) with:

1. ✅ New sub-organization with entitlements
2. ✅ Admin user for the sub-organization
3. ✅ Two environments (Production and Sandbox)
4. ✅ Connected app scopes assignment
5. ✅ Private space in the production environment
6. ✅ Private network within the private space

## Architecture

```
Parent Organization (Salesforce)
└── Sub-Organization (Created)
    ├── Production Environment (Created)
    │   └── Private Space (Created)
    │       └── Private Network (Created)
    ├── Sandbox Environment (Created)
    └── Connected App Scopes (Assigned)
        ├── admin:cloudhub
        ├── manage:runtime_fabrics
        ├── create:environment (both envs)
        ├── manage:private_spaces
        └── admin:api_manager
```

## Prerequisites

### 1. Existing Resources
- **Parent Organization ID** (Salesforce org)
- **Owner User ID** (existing user who will own the sub-org)
- **Connected App** with client ID `e5a776d9862a4f2d8f61ba8450803908`

### 2. Authentication Requirements

#### For All Resources (User Authentication Required):
```bash
export ANYPOINT_ADMIN_USERNAME="your.admin@email.com"
export ANYPOINT_ADMIN_PASSWORD="your-password"
```

**OR** configure in the provider block:
```hcl
provider "anypoint" {
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  auth_type     = "user"
}
```

**Note**: Connected app scope assignment requires user authentication because it's a privileged operation.

### 3. Permissions Required

The user/connected app must have:
- **Organization Administrator** role in the parent organization
- Permission to create sub-organizations
- Permission to create environments
- Permission to manage connected apps
- Permission to create private spaces

## File Structure

```
examples/comprehensive-e2e/
├── suborg_with_privatespace_complete.tf          # Main configuration
├── suborg_with_privatespace_variables.tf         # Variable definitions
├── suborg_with_privatespace.tfvars.example       # Example values
└── SUBORG_WITH_PRIVATESPACE_GUIDE.md            # This file
```

## Setup Instructions

### Step 1: Prepare Configuration

1. Copy the example tfvars file:
```bash
cd examples/comprehensive-e2e
cp suborg_with_privatespace.tfvars.example terraform.tfvars
```

2. Edit `terraform.tfvars` with your values:
```hcl
parent_organization_id     = "your-parent-org-id"
owner_user_id              = "your-owner-user-id"
sub_org_name               = "my-test-suborg"
anypoint_admin_client_id     = "e5a776d9862a4f2d8f61ba8450803908"
anypoint_admin_client_secret = "your-client-secret"
anypoint_admin_username      = "your-admin-username"
anypoint_admin_password      = "your-admin-password"
```

### Step 2: Set Environment Variables

```bash
# For resource management (connected app)
export ANYPOINT_CLIENT_ID="e5a776d9862a4f2d8f61ba8450803908"
export ANYPOINT_CLIENT_SECRET="your-client-secret"

# For scope assignment (user authentication)
export ANYPOINT_ADMIN_USERNAME="your.admin@email.com"
export ANYPOINT_ADMIN_PASSWORD="your-password"
```

### Step 3: Initialize Terraform

```bash
terraform init
```

### Step 4: Plan and Apply

```bash
# Review the execution plan
terraform plan -var-file=terraform.tfvars

# Apply the configuration
terraform apply -var-file=terraform.tfvars
```

### Step 5: Verify Outputs

After successful apply, Terraform will output:

```
Outputs:

sub_organization = {
  id         = "new-org-uuid"
  name       = "my-test-suborg"
  client_id  = "org-client-id"
  created_at = "2026-03-28T..."
}

environments = {
  production = {
    id            = "prod-env-uuid"
    name          = "my-test-suborg-production"
    client_id     = "env-client-id"
    arc_namespace = "namespace"
  }
  sandbox = {
    id            = "sandbox-env-uuid"
    name          = "my-test-suborg-sandbox"
    client_id     = "env-client-id"
    arc_namespace = "namespace"
  }
}

private_space = {
  id              = "space-uuid"
  name            = "my-test-suborg-prod-space"
  region          = "us-east-1"
  status          = "active"
  organization_id = "new-org-uuid"
}

private_network = {
  id                  = "network-uuid"
  cidr_block          = "10.111.0.0/16"
  inbound_static_ips  = ["1.2.3.4", "5.6.7.8"]
  outbound_static_ips = ["9.10.11.12", "13.14.15.16"]
  dns_target          = "vpce-xxx.vpce-svc-xxx.us-east-1.vpce.amazonaws.com"
}
```

## Resource Details

### Sub-Organization Entitlements

The sub-organization is created with the following entitlements:

| Resource | Assigned | Purpose |
|----------|----------|---------|
| vCores Production | 2 | Production app deployments |
| vCores Sandbox | 1 | Sandbox app deployments |
| vCores Design | 0.5 | Design center usage |
| Static IPs | 2 | Outbound connections |
| VPCs | 1 | Private space |
| VPNs | 2 | Site-to-site connections |
| Network Connections | 3 | VPN, etc. |
| Runtime Fabric | Enabled | On-premises runtime |
| Omni Gateway | Enabled | API gateway |

You can adjust these in the `entitlements` section of the configuration.

### Connected App Scopes

The following scopes are assigned to the connected app:

```hcl
admin:cloudhub              # CloudHub 2.0 administration
manage:runtime_fabrics      # Runtime Fabric management
create:Environment          # Environment creation (both prod & sandbox)
manage:private_spaces       # Private space management
admin:api_manager           # API Manager administration
```

### Private Network Configuration

- **CIDR Block**: `10.111.0.0/16` (customizable)
- **Reserved CIDRs**: Optional ranges for VPN/TGW
- **Static IPs**: Auto-assigned (4 inbound, 2 outbound)
- **DNS Target**: Auto-generated VPC endpoint

## Post-Deployment Tasks

### 1. Configure VPN

Connect your on-premises network to the private space:

```hcl
# Option 1: VPN Connection
resource "anypoint_vpn_connection" "site_to_site" {
  private_space_id = anypoint_private_space.production_space.id
  organization_id  = anypoint_organization.sub_org.id
  name             = "on-prem-vpn"

  vpns = [
    {
      remote_ip_address = "203.0.113.1"
      local_asn         = 64512
      remote_asn        = 64513
      vpn_tunnels       = [...]
    }
  ]
}
```

### 3. Deploy Applications

Deploy Mule applications to the private space:

```hcl
resource "anypoint_application" "my_app" {
  environment_id = anypoint_environment.production.id
  name           = "my-mule-app"
  target_id      = anypoint_private_space.production_space.id
  # ... application configuration
}
```

### 4. Set Up API Management

Create API instances and apply policies:

```hcl
resource "anypoint_api_instance" "my_api" {
  environment_id = anypoint_environment.production.id
  technology     = "omniGateway"
  # ... API configuration
}
```

## Network Planning

### CIDR Block Selection

**Important**: Choose a CIDR block that doesn't overlap with:
- Your on-premises networks
- Other private spaces
- AWS VPC CIDR blocks you might peer with

**Recommended ranges**:
- `10.111.0.0/16` (65,536 IPs)
- `172.20.0.0/16` (65,536 IPs)
- `192.168.100.0/22` (1,024 IPs)

**Avoid**:
- `10.0.0.0/8` (often used by AWS)
- `172.16.0.0/12` (often used by Docker)
- `192.168.0.0/16` (common home networks)

### Reserved CIDRs

Reserve CIDR ranges for specific purposes:

```hcl
network_reserved_cidrs = [
  "10.111.1.0/24",   # Reserved for VPN
  "10.111.2.0/24"    # Reserved for future use
]
```

## Troubleshooting

### Connected App Scopes Assignment Fails

**Error**: `401 Unauthorized`

**Solution**: Ensure user authentication is configured:
```bash
export ANYPOINT_ADMIN_USERNAME="your.admin@email.com"
export ANYPOINT_ADMIN_PASSWORD="your-password"
```

The connected app scopes resource requires user authentication, not just client credentials.

### Private Space Creation Takes Long

**Normal**: Private space creation typically takes 10-15 minutes as it provisions AWS infrastructure (VPC, subnets, NAT gateways, etc.).

**Check status**: Use `terraform show` to see the current status.

### Owner User Not Found

**Error**: `Owner user ID not found`

**Solution**: Ensure the `owner_user_id` refers to an existing user in the parent organization. You cannot use a newly created user as the owner in the same run.

### Insufficient Entitlements

**Error**: `Organization doesn't have enough entitlements`

**Solution**: Check the parent organization has sufficient entitlements to allocate to the sub-organization.

## Cleanup

To destroy all resources:

```bash
terraform destroy -var-file=terraform.tfvars
```

**Note**: Resources are destroyed in reverse dependency order:
1. Private network
2. Private space
3. Connected app scopes
4. Environments
5. Admin user
6. Sub-organization

## Security Best Practices

1. **Secrets Management**:
   - Never commit `terraform.tfvars` to version control
   - Use environment variables for sensitive values
   - Consider using a secrets manager (AWS Secrets Manager, HashiCorp Vault)

2. **Password Requirements**:
   - Minimum 8 characters
   - At least one uppercase letter
   - At least one lowercase letter
   - At least one number
   - At least one special character

3. **MFA Enforcement**:
   - Set `mfa_verification_excluded = false` for production users
   - Enable MFA in organization settings

4. **Role-Based Access**:
   - Assign minimal required permissions
   - Use role groups for access control
   - Regularly audit user permissions

## Cost Considerations

- **Private Space**: ~$1,000/month per region (AWS infrastructure)
- **vCores**: Charged per vCore hour
- **Static IPs**: Included with private space
- **Data Transfer**: Outbound data transfer charges apply

## Next Steps

1. ✅ Configure firewall rules for the private space
2. ✅ Set up VPN connectivity
3. ✅ Deploy Mule applications
4. ✅ Create API instances and policies
5. ✅ Configure monitoring and alerting
6. ✅ Set up CI/CD pipelines

## Support

For issues or questions:
- Anypoint Platform: https://help.mulesoft.com
- Terraform Provider: https://github.com/mulesoft/terraform-provider-anypoint

## References

- [Anypoint Platform Docs](https://docs.mulesoft.com/)
- [Private Spaces Documentation](https://docs.mulesoft.com/runtime-manager/cloudhub-2-architecture)
- [Connected Apps](https://docs.mulesoft.com/access-management/connected-apps)
- [Environment Management](https://docs.mulesoft.com/access-management/environments)
