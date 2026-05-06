# Anypoint Terraform Provider - Client Standard Operating Procedure (SOP)

## Document Information
- **Version**: 1.0
- **Purpose**: Complete setup and usage guide for clients using the Anypoint Terraform Provider
- **Audience**: External clients and implementation teams

---

## 🚀 Quick Start

**New to this provider?** Start with `CLIENT_QUICK_START.md` for a 5-minute setup guide.

**This document covers advanced topics** like production deployment, troubleshooting, and comprehensive configuration options.

## Table of Contents
1. [Advanced Authentication](#1-advanced-authentication)
2. [Advanced Configuration](#2-advanced-configuration)
3. [Running Multiple Examples](#3-running-multiple-examples)
4. [Troubleshooting](#4-troubleshooting)
5. [Production Deployment](#5-production-deployment)
6. [Support and Contact](#6-support-and-contact)

---

> **📋 Prerequisites, System Requirements, and Basic Installation**
> 
> These are covered in `CLIENT_QUICK_START.md`. This document focuses on advanced topics.

## 1. Advanced Authentication

### 1.1 Understanding Authentication Types

The provider supports two authentication methods:

| Method | Use Case | Requirements |
|---|---|---|
| **Connected App** | Most resources, automated workflows | Client ID + Client Secret |
| **User Authentication** | Organization management, connected app scopes | Username + Password + Client ID + Client Secret |

### 1.2 Setting Up Connected App Authentication

#### 1.2.1 Create a Connected App (if not existing)
1. Log in to Anypoint Platform
2. Navigate to **Access Management** → **Connected Apps**
3. Click **Create App**
4. Configure the app with required scopes:

**Minimum Required Scopes:**
```
- read:organizations
- write:organizations (for creating/modifying)
- admin:cloudhub (for CloudHub 2.0 resources)
- read:environments
- write:environments
- admin:runtimefabric (if using Runtime Fabric)
```

**Advanced Scopes (for full functionality):**
```
- admin:users
- admin:teams
- admin:rolegroups
- admin:applications
```

#### 1.2.2 Note Your Credentials
After creating the connected app, save:
- **Client ID**: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
- **Client Secret**: `xxxxxxxxxxxxxxxxxxxxxxxxxxxx`

### 1.3 User Authentication Setup (For Advanced Features)

Some operations require user authentication:
- Organization management (`anypoint_organization`)
- Connected app scope management (`anypoint_connected_app_scopes`)

You'll need:
- Your Anypoint Platform **username**
- Your Anypoint Platform **password**
- A connected app on behalf of user with appropriate scopes

---

## 2. Advanced Configuration

> **💡 Basic credential setup is covered in `CLIENT_QUICK_START.md`**

### 2.1 Multi-Environment Configuration

Configure different environments with workspace-specific variables:

```bash
# Production
export TF_VAR_anypoint_base_url="https://anypoint.mulesoft.com"
export TF_VAR_anypoint_client_id="prod-client-id"

# Staging  
export TF_VAR_anypoint_base_url="https://stgx.anypoint.mulesoft.com"
export TF_VAR_anypoint_client_id="staging-client-id"
```

### 2.2 Advanced Provider Features

```hcl
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
  
  # Advanced options
  timeout       = "30s"        # API timeout
  retry_count   = 3            # Number of retries for failed requests
  debug         = true         # Enable debug logging
}
```

### 2.3 Secure Credential Management

For production environments:

```bash
# Using AWS Secrets Manager
export TF_VAR_anypoint_client_id=$(aws secretsmanager get-secret-value --secret-id anypoint/client-id --query SecretString --output text)

# Using Azure Key Vault
export TF_VAR_anypoint_client_id=$(az keyvault secret show --name anypoint-client-id --vault-name my-vault --query value -o tsv)

# Using HashiCorp Vault
export TF_VAR_anypoint_client_id=$(vault kv get -field=client_id secret/anypoint)
```

## 3. Running Multiple Examples

> **💡 Single example usage is covered in `CLIENT_QUICK_START.md`**

### 3.1 Sequential Deployment Strategy

For learning and testing, deploy examples in order of dependency:

```bash
# 1. Foundation: Environments and Teams
cd examples/accessmanagement/environment && terraform apply
cd ../team && terraform apply

# 2. Infrastructure: Private Spaces
cd ../../cloudhub2/privatespace && terraform apply

# 3. Networking: Networks and Security
cd ../privatenetwork && terraform apply
cd ../firewallrules && terraform apply

# 4. Advanced: TLS and VPN
cd ../tlscontext && terraform apply
cd ../vpnconnection && terraform apply
```

### 3.2 Integrated Multi-Resource Configuration

Create a master configuration combining multiple resources:

```hcl
# main.tf
terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

# Environment setup
module "environments" {
  source = "./modules/environments"
  
  environments = {
    development = { type = "sandbox", is_production = false }
    staging     = { type = "sandbox", is_production = false }
    production  = { type = "production", is_production = true }
  }
}

# Team structure
module "teams" {
  source = "./modules/teams"
  
  teams = {
    platform_team = { members = ["user1", "user2"] }
    dev_team      = { members = ["user3", "user4"] }
  }
  
  depends_on = [module.environments]
}

# Infrastructure
module "cloudhub2" {
  source = "./modules/cloudhub2"
  
  private_spaces = {
    production = { region = "us-east-1", enable_egress = true }
    staging    = { region = "us-west-2", enable_egress = false }
  }
  
  depends_on = [module.environments]
}
```

### 3.3 Environment-Specific Deployments

Use Terraform workspaces for environment separation:

```bash
# Create workspaces
terraform workspace new development
terraform workspace new staging  
terraform workspace new production

# Deploy to each environment
terraform workspace select development
terraform apply -var-file="development.tfvars"

terraform workspace select production
terraform apply -var-file="production.tfvars"
```

### 3.4 Dependency Management

Handle resource dependencies properly:

```hcl
# Private space first
resource "anypoint_private_space" "main" {
  name   = "production-space"
  region = "us-east-1"
}

# Network depends on space
resource "anypoint_private_network" "main" {
  name              = "production-network"
  private_space_id  = anypoint_private_space.main.id
  cidr_block        = "10.0.0.0/16"
}

# Firewall rules depend on network
resource "anypoint_firewall_rules" "main" {
  private_space_id = anypoint_private_space.main.id
  
  rules = [
    {
      protocol    = "TCP"
      from_port   = 443
      to_port     = 443
      destination = "0.0.0.0/0"
    }
  ]
  
  depends_on = [anypoint_private_network.main]
}
```

---

## 4. Troubleshooting

### 4.1 Common Issues and Solutions

#### 4.1.1 Authentication Issues

**Problem**: `Error: 401 Unauthorized`
```
Error: failed to authenticate: HTTP 401: {"message":"Unauthorized"}
```

**Solutions**:
1. Verify client ID and secret are correct
2. Check connected app scopes
3. Ensure base URL is correct for your environment
4. For user auth operations, verify username/password

**Problem**: `Error: 403 Forbidden`
```
Error: insufficient permissions for this operation
```

**Solutions**:
1. Check connected app has required scopes
2. Verify user has appropriate permissions
3. Ensure target organization access

#### 4.1.2 Provider Installation Issues

**Problem**: `Provider not found`
```
Error: Failed to query available provider packages
```

**Solutions**:
1. Rebuild and reinstall the provider:
   ```bash
   make clean && make install
   ```
2. Check the installation path matches your architecture
3. Verify Terraform can find the provider:
   ```bash
   terraform providers
   ```

#### 4.1.3 Resource Configuration Issues

**Problem**: `Invalid configuration`
```
Error: Unsupported argument
```

**Solutions**:
1. Check the example configurations for correct syntax
2. Verify variable names and types
3. Review the resource documentation

#### 4.1.4 Network and Connectivity Issues

**Problem**: `Connection timeout`
```
Error: context deadline exceeded
```

**Solutions**:
1. Check internet connectivity
2. Verify firewall settings allow HTTPS traffic
3. Test API access directly:
   ```bash
   curl -H "Authorization: Bearer YOUR_TOKEN" https://anypoint.mulesoft.com/accounts/api/me
   ```

### 4.2 Debugging Tools

#### 4.2.1 Enable Debug Logging
```bash
export TF_LOG=DEBUG
terraform plan
```

#### 4.2.2 Terraform State Inspection
```bash
# View current state
terraform show

# List resources in state
terraform state list

# Get detailed resource info
terraform state show anypoint_environment.example
```

#### 4.2.3 API Testing
Test your credentials and connectivity:

```bash
# Get access token
curl -X POST https://anypoint.mulesoft.com/accounts/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your-username",
    "password": "your-password"
  }'

# Test with connected app
curl -X POST https://anypoint.mulesoft.com/accounts/api/v2/oauth2/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "your-client-id",
    "client_secret": "your-client-secret",
    "grant_type": "client_credentials"
  }'
```

### 4.3 Getting Support

When requesting support, provide:

1. **Environment Information**:
   ```bash
   terraform version
   go version  # if building from source
   uname -a    # OS information
   ```

2. **Error Details**:
   - Complete error message
   - Debug logs (with sensitive data removed)
   - Steps to reproduce

3. **Configuration** (anonymized):
   - Terraform configuration files
   - Variable values (remove credentials)

---

## 5. Production Deployment

### 5.1 Best Practices for Production

#### 5.1.1 State Management
- Use remote state backends (S3, Azure Storage, etc.)
- Enable state locking
- Use separate state files for different environments

```hcl
terraform {
  backend "s3" {
    bucket = "your-terraform-state"
    key    = "anypoint/prod/terraform.tfstate"
    region = "us-east-1"
  }
}
```

#### 5.1.2 Security
- Use environment variables or secure secret managers for credentials
- Rotate API credentials regularly
- Implement least-privilege access
- Never commit sensitive data to version control

#### 5.1.3 Environment Management
```bash
# Use workspaces for environment separation
terraform workspace new production
terraform workspace new staging
terraform workspace new development
```

### 5.2 CI/CD Integration

#### 5.2.1 Example GitHub Actions Workflow
```yaml
name: Terraform Apply
on:
  push:
    branches: [main]

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v2
        with:
          terraform_version: 1.5.0
          
      - name: Terraform Init
        run: terraform init
        env:
          TF_VAR_anypoint_client_id: ${{ secrets.ANYPOINT_CLIENT_ID }}
          TF_VAR_anypoint_client_secret: ${{ secrets.ANYPOINT_CLIENT_SECRET }}
          
      - name: Terraform Plan
        run: terraform plan
        
      - name: Terraform Apply
        if: github.ref == 'refs/heads/main'
        run: terraform apply -auto-approve
```

### 5.3 Monitoring and Maintenance

#### 5.3.1 Regular Tasks
- Monitor Terraform state drift
- Update provider versions
- Review and rotate credentials
- Backup state files

#### 5.3.2 Health Checks
```bash
# Verify provider functionality
terraform plan -refresh-only

# Check resource states
terraform state list | xargs -I {} terraform state show {}
```

---

## 6. Support and Contact

### 6.1 Documentation Resources
- **Quick Start**: `CLIENT_QUICK_START.md` - 5-minute setup guide
- **Main README**: `README.md` - Project overview
- **Example Documentation**: `examples/README.md` - Example overview
- **Individual Example READMEs**: Located in each example directory

### 6.2 Self-Help Resources

#### 6.2.1 Quick Reference Commands
```bash
# Provider build and install
make install

# Example execution
cd examples/accessmanagement/environment
terraform plan && terraform apply

# Cleanup
terraform destroy

# Debug mode
export TF_LOG=DEBUG && terraform apply
```

#### 6.2.2 Useful URLs
- **Anypoint Platform**: https://anypoint.mulesoft.com
- **Terraform Documentation**: https://www.terraform.io/docs
- **Go Documentation**: https://golang.org/doc

### 6.3 Technical Support

For technical issues or questions:

1. **Check this SOP** and existing documentation first
2. **Search known issues** in the repository
3. **Gather required information**:
   - Terraform version
   - Go version (if building from source)
   - Operating system details
   - Complete error messages
   - Anonymized configuration files
   - Steps to reproduce the issue

4. **Contact your technical support representative** with the gathered information

### 6.4 Escalation Process

| Issue Severity | Response Time | Escalation |
|---|---|---|
| 🔴 **Critical** - Production down | 4 hours | Immediate escalation |
| 🟡 **High** - Major functionality impacted | 1 business day | Standard escalation |
| 🟢 **Medium** - Minor issues, workarounds available | 3 business days | Standard process |
| 🔵 **Low** - Questions, documentation | 5 business days | Standard process |

---

## Document Version History

| Version | Date | Changes | Author |
|---|---|---|---|
| 1.0 | 2024 | Initial comprehensive SOP creation | Implementation Team |

---

**End of Document**

*This SOP is a living document and will be updated as the provider evolves and based on client feedback.*