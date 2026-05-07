# Complete Examples Index

This directory contains comprehensive end-to-end examples for Anypoint Platform Terraform Provider.

## Available Examples

### 1. Mule4 API Instance Support ✨ NEW

**Files**:
- `api_instance_mule4_example.tf` - Mule4 API instance example
- `MULE4_SUPPORT.md` - Complete Mule4 support documentation
- `CHANGES_SUMMARY.md` - Summary of Mule4 implementation changes

**What it demonstrates**:
- Creating API instances with `technology = "mule4"`
- Direct URI endpoint configuration (vs OmniGateway base_path)
- Differences between Mule4 and OmniGateway patterns
- Using autodiscovery instance name in Mule applications

**Use when**: You need to manage API instances for Mule4 applications

**Key changes**:
- ✅ Enhanced `endpoint` block with `uri` field
- ✅ Technology-aware endpoint handling
- ✅ Backward compatible with existing OmniGateway configs

---

### 2. Sub-Organization with Private Space ✨ NEW

**Files**:
- `suborg_with_privatespace_complete.tf` - Complete infrastructure setup
- `suborg_with_privatespace_variables.tf` - Variable definitions
- `suborg_with_privatespace.tfvars.example` - Example configuration
- `SUBORG_WITH_PRIVATESPACE_GUIDE.md` - Detailed 20-page guide
- `SUBORG_EXAMPLE_README.md` - Quick reference
- `setup_suborg.sh` - Automated setup script

**What it demonstrates**:
- Creating a new sub-organization within parent org
- Creating multiple environments (production & sandbox)
- Assigning connected app scopes to specific environments
- Provisioning private spaces and networks
- Complete network configuration with CIDR planning
- Using user authentication for scope assignment

**Use when**: You need to set up a new organizational unit with isolated infrastructure

**Architecture**:
```
Parent Org (Salesforce)
└── Sub-Organization
    ├── Production Environment
    │   └── Private Space
    │       └── Private Network
    ├── Sandbox Environment
    └── Connected App Scopes
```

---

### 3. OmniGateway with Full Policy Suite

**Files**:
- `main.tf` - Complete OmniGateway deployment
- `variables.tf` - Variable definitions
- `terraform.tfvars.example` - Example values

**What it demonstrates**:
- Creating private space and network
- Setting up VPN connections
- Deploying managed Omni Gateway
- Creating API instance with routing
- Applying 33 different API policies
- Configuring SLA tiers
- Setting up alerts
- Promoting API instances between environments

**Use when**: You need a complete API management setup with OmniGateway

**Resources created**:
1. Secret management (keystore, truststore, TLS context)
2. Managed OmniGateway
3. API instance with weighted routing
4. 33 API policies (rate limiting, JWT, CORS, etc.)
5. SLA tiers
6. Alerts
7. API instance promotion

---

## Quick Start Guide

### For Mule4 API Instance

```bash
# View the example
cat api_instance_mule4_example.tf

# Read the documentation
open MULE4_SUPPORT.md

# Use in your config
terraform apply -target=anypoint_api_instance.mule4_api
```

### For Sub-Organization Setup

```bash
# Option 1: Automated (Recommended)
./setup_suborg.sh
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars

# Option 2: Manual
cp suborg_with_privatespace.tfvars.example terraform.tfvars
# Edit terraform.tfvars
terraform init
terraform apply -var-file=terraform.tfvars
```

### For OmniGateway Deployment

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars
terraform init
terraform plan
terraform apply
```

---

## Comparison Matrix

| Feature | OmniGateway Example | Mule4 Example | Sub-Org Example |
|---------|-------------------|---------------|-----------------|
| **API Gateway** | Omni Gateway | Mule Runtime | N/A |
| **Endpoint Config** | `base_path` | `uri` | N/A |
| **Routing** | ✅ Weighted | ❌ App-managed | N/A |
| **TLS Context** | ✅ Required | ❌ Runtime config | N/A |
| **Policies** | ✅ 33 types | ✅ Same policies | N/A |
| **Private Space** | ✅ Optional | ❌ N/A | ✅ Created |
| **Sub-Org** | ❌ Uses existing | ❌ Uses existing | ✅ Created |
| **Environments** | ❌ Uses existing | ❌ Uses existing | ✅ 2 created |
| **Scopes** | ❌ N/A | ❌ N/A | ✅ Assigned |
| **Auth Type** | client_credentials | client_credentials | user |

---

## Resource Dependencies

### OmniGateway Example
```
Secret Group → TLS Context
                    ↓
Private Space → Private Network → VPN → Omni Gateway → API Instance → Policies
                                                                         ↓
                                                               SLA Tiers + Alerts
                                                                         ↓
                                                                     Promotion
```

### Mule4 Example
```
Exchange Asset → API Instance (no gateway needed) → Policies
                                                        ↓
                                              SLA Tiers + Alerts
```

### Sub-Org Example
```
Parent Org → Sub-Org → User
                ↓
            Environments
                ↓
        Connected App Scopes
                ↓
          Private Space → Private Network
```

---

## Authentication Requirements

### OmniGateway & Mule4 Examples
```bash
# Connected App (Client Credentials)
export ANYPOINT_CLIENT_ID="your-client-id"
export ANYPOINT_CLIENT_SECRET="your-secret"
```

### Sub-Org Example
```bash
# User Authentication - required for all resources including scope assignment
export ANYPOINT_ADMIN_USERNAME="admin@example.com"
export ANYPOINT_ADMIN_PASSWORD="your-password"

# Can also be set in provider config with auth_type = "user"
```

---

## Common Variables

All examples use these variables:

```hcl
# Provider (OmniGateway & Mule4 examples)
anypoint_client_id     = "your-client-id"
anypoint_client_secret = "your-secret"
anypoint_base_url      = "https://stgx.anypoint.mulesoft.com"

# Provider (Sub-Org example - requires user auth)
anypoint_admin_client_id     = "your-client-id"
anypoint_admin_client_secret = "your-secret"
anypoint_admin_username      = "your-username"
anypoint_admin_password      = "your-password"
anypoint_base_url            = "https://stgx.anypoint.mulesoft.com"

# Organization & Environment
organization_id = "your-org-id"
environment_id  = "your-env-id"

# API Specification
api_asset_id      = "your-api-asset"
api_asset_version = "1.0.0"
```

---

## Documentation Files

### Getting Started
1. **SUBORG_EXAMPLE_README.md** - Quick start for sub-org setup
2. **MULE4_SUPPORT.md** - Quick start for Mule4 API instances

### Detailed Guides
1. **SUBORG_WITH_PRIVATESPACE_GUIDE.md** - 20-page comprehensive guide
   - Prerequisites
   - Step-by-step instructions
   - Network planning
   - Troubleshooting
   - Cost estimates
   - Security best practices

2. **MULE4_SUPPORT.md** - Detailed Mule4 documentation
   - Architecture decisions
   - API request/response formats
   - OmniGateway vs Mule4 comparison
   - Migration guide

### Reference
1. **CHANGES_SUMMARY.md** - Mule4 implementation changes
2. **COMPLETE_EXAMPLES_INDEX.md** - This file

---

## File Organization

```
examples/comprehensive-e2e/
├── README.md (original)
├── COMPLETE_EXAMPLES_INDEX.md (this file)
│
├── OmniGateway Example (original)
│   ├── main.tf
│   ├── variables.tf
│   └── terraform.tfvars.example
│
├── Mule4 Support (NEW)
│   ├── api_instance_mule4_example.tf
│   ├── MULE4_SUPPORT.md
│   └── CHANGES_SUMMARY.md
│
└── Sub-Organization Setup (NEW)
    ├── suborg_with_privatespace_complete.tf
    ├── suborg_with_privatespace_variables.tf
    ├── suborg_with_privatespace.tfvars.example
    ├── SUBORG_WITH_PRIVATESPACE_GUIDE.md
    ├── SUBORG_EXAMPLE_README.md
    └── setup_suborg.sh
```

---

## Usage Patterns

### Pattern 1: Start with Sub-Org (Greenfield)
```bash
# 1. Create organizational structure
terraform apply -target=anypoint_organization.sub_org
terraform apply -target=anypoint_environment.production
terraform apply -target=anypoint_environment.sandbox

# 2. Assign scopes
terraform apply -target=anypoint_connected_app_scopes.app_scopes

# 3. Create infrastructure
terraform apply -target=anypoint_private_space.production_space
terraform apply -target=anypoint_private_network.production_network

# 4. Deploy API instances
terraform apply
```

### Pattern 2: Add Mule4 APIs (Existing Org)
```bash
# Use existing organization and environment
# Just create API instance
terraform apply -target=anypoint_api_instance.mule4_api
```

### Pattern 3: Full OmniGateway Stack (Existing Org)
```bash
# Create everything at once
terraform apply
```

---

## Best Practices

### 1. Start Small
- ✅ Test with sandbox environment first
- ✅ Use minimal entitlements initially
- ✅ Validate network CIDR planning

### 2. Use Version Control
- ✅ Commit `.tf` and `.md` files
- ❌ Never commit `terraform.tfvars`
- ❌ Never commit `.tfstate` files
- ✅ Use `.gitignore` for sensitive files

### 3. Environment Variables
- ✅ Use env vars for secrets
- ✅ Document required variables
- ✅ Create `.env.example` templates

### 4. Resource Naming
- ✅ Use consistent prefixes
- ✅ Include environment in names
- ✅ Use descriptive labels

### 5. Network Planning
- ✅ Document CIDR allocations
- ✅ Reserve ranges for future use
- ✅ Avoid common network overlaps

---

## Common Use Cases

### Use Case 1: Multi-Tenant SaaS Platform
**Example**: Sub-Organization Setup

Create isolated sub-orgs for each customer:
- Separate billing and entitlements
- Isolated private spaces
- Customer-specific environments
- Scoped access via connected apps

### Use Case 2: Microservices API Gateway
**Example**: OmniGateway Setup

Deploy Omni Gateway with:
- Weighted routing to multiple backends
- Comprehensive policy enforcement
- TLS termination
- Rate limiting and throttling

### Use Case 3: Hybrid Integration
**Example**: Mule4 API Instance

Manage APIs for Mule applications:
- Direct implementation URIs
- Autodiscovery integration
- Policy enforcement
- SLA tier management

---

## Troubleshooting Guide

### Common Issues

| Issue | Example | Solution |
|-------|---------|----------|
| 401 Unauthorized | All | Check auth credentials |
| Scope assignment fails | Sub-Org | Set user auth env vars |
| Private space timeout | Sub-Org | Wait 15 min, normal |
| CIDR overlap | Sub-Org | Choose different range |
| Owner not found | Sub-Org | Use existing user ID |
| Base path vs URI | Mule4 | Check technology field |
| TLS context error | OmniGateway | Verify composite ID format |

### Debug Commands

```bash
# Check current state
terraform show

# Refresh state
terraform refresh

# View specific resource
terraform state show anypoint_organization.sub_org

# Enable debug logging
export TF_LOG=DEBUG
terraform apply

# Validate configuration
terraform validate

# Check formatting
terraform fmt -check
```

---

## Cost Optimization

### Tips for Each Example

**OmniGateway**:
- Start with small gateway size
- Use shared TLS contexts
- Combine policies where possible
- Monitor vCore usage

**Mule4**:
- Use runtime fabric for multiple apps
- Share environments across teams
- Right-size vCore allocation

**Sub-Organization**:
- Start with minimal entitlements
- Grow as needed
- Use sandbox for testing
- Consider global deployment carefully

---

## Next Steps

After deploying these examples:

1. **Security Hardening**
   - Enable MFA for all users
   - Rotate credentials regularly
   - Audit permissions quarterly
   - Implement least privilege

2. **Monitoring Setup**
   - Configure alerts
   - Set up log forwarding
   - Create dashboards
   - Monitor costs

3. **CI/CD Integration**
   - Automate deployments
   - Use workspaces
   - Implement approval gates
   - Add automated testing

4. **Documentation**
   - Document architecture decisions
   - Create runbooks
   - Maintain change logs
   - Update diagrams

---

## Support & Resources

### Documentation
- 📖 [Terraform Provider Docs](https://github.com/mulesoft/terraform-provider-anypoint)
- 📖 [Anypoint Platform Docs](https://docs.mulesoft.com/)
- 📖 [CloudHub 2.0 Docs](https://docs.mulesoft.com/runtime-manager/cloudhub-2)

### Community
- 💬 [MuleSoft Community](https://community.mulesoft.com/)
- 💬 [Stack Overflow](https://stackoverflow.com/questions/tagged/mulesoft)

### Help
- 🆘 [MuleSoft Support](https://help.mulesoft.com/)
- 🆘 [Training](https://training.mulesoft.com/)

---

## Contributing

Found an issue or have an improvement?
1. Fork the repository
2. Create a feature branch
3. Submit a pull request

---

## Changelog

### 2026-03-28
- ✨ Added Mule4 API instance support
- ✨ Added Sub-Organization with Private Space example
- ✨ Created comprehensive documentation
- ✨ Added automated setup script
- 🐛 Fixed ssl_context_id format in OmniGateway example

### Original
- Initial OmniGateway example with 33 policies

---

**Last Updated**: March 28, 2026
**Terraform Version**: >= 1.0
**Provider Version**: 0.1.0
