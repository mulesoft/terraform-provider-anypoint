# Sub-Organization with Private Space Example

Complete Terraform example for creating a new sub-organization in Anypoint Platform with all necessary infrastructure.

## Quick Start

```bash
cd examples/comprehensive-e2e
./setup_suborg.sh
```

The setup script will:
1. ✅ Create `terraform.tfvars` from example
2. ✅ Prompt for authentication credentials
3. ✅ Initialize Terraform
4. ✅ Validate configuration

## What Gets Created

```
┌─────────────────────────────────────────────────────────────┐
│ Parent Organization (Salesforce)                            │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Sub-Organization (New)                                  │ │
│ │                                                         │ │
│ │ ├── Production Environment                             │ │
│ │ │   └── Private Space                                  │ │
│ │ │       └── Private Network (10.111.0.0/16)           │ │
│ │ │           ├── Inbound IPs: 4 static IPs             │ │
│ │ │           ├── Outbound IPs: 2 static IPs            │ │
│ │ │           └── DNS Target: VPC Endpoint              │ │
│ │                                                         │ │
│ │ └── Sandbox Environment                                │ │
│ │                                                         │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ Connected App (Existing: e5a776d9862a4f2d8f61ba8450803908)│
│ ├── Scopes Assigned:                                       │
│ │   ├── admin:cloudhub                                    │
│ │   ├── manage:runtime_fabrics                            │
│ │   ├── create:environment (prod)                         │
│ │   ├── create:environment (sandbox)                      │
│ │   ├── manage:private_spaces                             │
│ │   └── admin:api_manager                                 │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Files

| File | Purpose |
|------|---------|
| `suborg_with_privatespace_complete.tf` | Main Terraform configuration |
| `suborg_with_privatespace_variables.tf` | Variable definitions |
| `suborg_with_privatespace.tfvars.example` | Example values |
| `SUBORG_WITH_PRIVATESPACE_GUIDE.md` | Detailed documentation |
| `setup_suborg.sh` | Automated setup script |
| `SUBORG_EXAMPLE_README.md` | This file |

## Prerequisites

### Required
- ✅ Terraform >= 1.0
- ✅ Parent organization ID (Salesforce org)
- ✅ Connected app client ID and secret
- ✅ Owner user ID (existing user)

### Authentication
- ✅ Connected app credentials (for resource management)
- ✅ Admin user credentials (for scope assignment)

## Usage

### Option 1: Automated Setup (Recommended)

```bash
# Run the setup script
./setup_suborg.sh

# The script will guide you through:
# - Creating terraform.tfvars
# - Setting environment variables
# - Initializing Terraform
# - Running validation

# Review the plan
terraform plan -var-file=terraform.tfvars

# Apply the configuration
terraform apply -var-file=terraform.tfvars
```

### Option 2: Manual Setup

```bash
# 1. Create configuration file
cp suborg_with_privatespace.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

# 2. Set environment variables
export ANYPOINT_CLIENT_ID="e5a776d9862a4f2d8f61ba8450803908"
export ANYPOINT_CLIENT_SECRET="your-secret"
export ANYPOINT_ADMIN_USERNAME="admin@example.com"
export ANYPOINT_ADMIN_PASSWORD="password"

# 3. Initialize and apply
terraform init
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

## Configuration

### Minimum Required Variables

```hcl
# terraform.tfvars
parent_organization_id     = "parent-org-uuid"
owner_user_id              = "owner-user-uuid"
sub_org_name               = "my-suborg"
anypoint_admin_client_id     = "e5a776d9862a4f2d8f61ba8450803908"
anypoint_admin_client_secret = "your-client-secret"
anypoint_admin_username      = "your-admin-username"
anypoint_admin_password      = "your-admin-password"
```

### Optional Customizations

```hcl
# Private space region
private_space_region = "us-east-1"  # or us-west-2, eu-central-1, etc.

# Network CIDR
network_cidr_block = "10.111.0.0/16"

# Reserved CIDRs for VPN/TGW
network_reserved_cidrs = ["10.111.1.0/24", "10.111.2.0/24"]
```

## Outputs

After successful apply:

```hcl
Outputs:

sub_organization = {
  id         = "new-org-uuid"
  name       = "my-suborg"
  client_id  = "org-client-id"
  created_at = "2026-03-28T..."
}

environments = {
  production = { ... }
  sandbox    = { ... }
}

private_space = {
  id     = "space-uuid"
  status = "active"
  ...
}

private_network = {
  id                  = "network-uuid"
  cidr_block          = "10.111.0.0/16"
  inbound_static_ips  = ["1.2.3.4", ...]
  outbound_static_ips = ["5.6.7.8", ...]
  dns_target          = "vpce-xxx...."
}
```

## Post-Deployment

### 1. Connect On-Premises Network

```hcl
resource "anypoint_vpn_connection" "vpn" {
  private_space_id = anypoint_private_space.production_space.id
  organization_id  = anypoint_organization.sub_org.id
  name             = "on-prem-vpn"
  vpns = [{ ... }]
}
```

### 2. Deploy Applications

```hcl
resource "anypoint_application" "app" {
  environment_id = anypoint_environment.production.id
  name           = "my-app"
  target_id      = anypoint_private_space.production_space.id
  # ... app configuration
}
```

## Network Planning

### CIDR Block Guidelines

**Choose non-overlapping ranges**:
- ✅ `10.111.0.0/16` (recommended)
- ✅ `172.20.0.0/16`
- ✅ `192.168.100.0/22`

**Avoid overlaps with**:
- ❌ On-premises networks
- ❌ Other AWS VPCs
- ❌ Docker default (`172.17.0.0/16`)
- ❌ Home networks (`192.168.0.0/16`)

### Reserved CIDRs

Reserve subnets for specific purposes:

```hcl
network_reserved_cidrs = [
  "10.111.1.0/24",   # VPN
  "10.111.2.0/24"    # Future use
]
```

## Entitlements

Sub-organization includes:

| Resource | Allocation |
|----------|-----------|
| vCores Production | 2.0 |
| vCores Sandbox | 1.0 |
| vCores Design | 0.5 |
| Static IPs | 2 |
| VPCs | 1 |
| VPNs | 2 |
| Network Connections | 3 |
| Runtime Fabric | Enabled |
| Omni Gateway | Enabled |

Adjust in `entitlements` block as needed.

## Troubleshooting

### Issue: Scope Assignment Fails (401)

**Cause**: User authentication not configured

**Fix**:
```bash
export ANYPOINT_ADMIN_USERNAME="your-admin@email.com"
export ANYPOINT_ADMIN_PASSWORD="your-password"
```

### Issue: Private Space Takes Long

**Normal**: 10-15 minutes to provision AWS infrastructure

**Check**: `terraform show anypoint_private_space.production_space`

### Issue: Owner User Not Found

**Cause**: Invalid `owner_user_id`

**Fix**: Use an existing user ID from parent organization

### Issue: Insufficient Entitlements

**Cause**: Parent org doesn't have enough resources

**Fix**: Request additional entitlements or reduce allocation

## Cost Estimate

| Resource | Monthly Cost (USD) |
|----------|-------------------|
| Private Space | ~$1,000 |
| vCores (2 prod + 1 sandbox) | ~$1,800 |
| Static IPs | Included |
| Data Transfer | Variable |
| **Total** | **~$2,800/month** |

Actual costs depend on usage and region.

## Cleanup

```bash
terraform destroy -var-file=terraform.tfvars
```

Destroys in reverse order:
1. Private network
2. Private space
3. Scopes
4. Environments
5. User
6. Sub-org

## Security

- ✅ Never commit `terraform.tfvars`
- ✅ Use environment variables for secrets
- ✅ Enable MFA for production users
- ✅ Audit permissions regularly
- ✅ Rotate credentials periodically

## Additional Examples

See related examples in this directory:

- `main.tf` - OmniGateway API instance with policies
- `api_instance_mule4_example.tf` - Mule4 API instance
- `MULE4_SUPPORT.md` - Mule4 vs OmniGateway comparison

## Support

- 📖 [Detailed Guide](SUBORG_WITH_PRIVATESPACE_GUIDE.md)
- 🔧 [Terraform Provider Docs](https://github.com/mulesoft/terraform-provider-anypoint)
- 💬 [MuleSoft Help Center](https://help.mulesoft.com)

## Related Resources

- [Anypoint Private Spaces](https://docs.mulesoft.com/runtime-manager/cloudhub-2-architecture)
- [Environment Management](https://docs.mulesoft.com/access-management/environments)
- [Connected Apps](https://docs.mulesoft.com/access-management/connected-apps)
- [VPN Connections](https://docs.mulesoft.com/runtime-manager/cloudhub-2-vpn)
