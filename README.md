# Anypoint Platform Terraform Provider

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/mulesoft/terraform-provider-anypoint)
[![License](https://img.shields.io/badge/license-MIT-blue)](https://opensource.org/licenses/MIT)
[![Terraform](https://img.shields.io/badge/terraform-1.0%2B-623CE4)](https://www.terraform.io)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8)](https://go.dev)

A comprehensive Terraform provider for managing your Anypoint Platform resources with ease and efficiency. Automate your infrastructure, from access management to CloudHub 2.0 deployments, and embrace Infrastructure as Code (IaC) for a more reliable and scalable integration landscape.

##  Why Use This Provider?

-   ** Automate Everything:** Codify your Anypoint Platform setup to ensure consistency and eliminate manual errors
-   ** Improve Collaboration:** Use version control to manage your infrastructure, making it easier for teams to collaborate and review changes
-   ** Increase Agility:** Spin up or tear down entire environments in minutes, not hours, allowing you to innovate faster
-   ** Enhance Governance:** Enforce standards and policies across all your environments by defining them in code
-   ** Complete Coverage:** 37 resources across 5 modules supporting the full Anypoint Platform lifecycle

## Table of Contents

- [Getting Started](#getting-started)
- [Authentication](#authentication)
- [Resources Overview](#resources-overview)
- [Provider Configuration](#provider-configuration)
- [Quick Start Examples](#quick-start-examples)
- [Complete Resource List](#complete-resource-list)
- [Examples](#examples)
- [Documentation](#documentation)
- [Contributing](#contributing)

## Getting Started

### Prerequisites

-   [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli) 1.0 or later
-   An Anypoint Platform account
-   Anypoint Platform Connected App credentials (Client ID and Secret) or user credentials

### Installation

Add the following to your Terraform configuration:

```hcl
terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = "https://anypoint.mulesoft.com"  # or your region-specific URL
}
```

## Authentication

The provider supports two authentication methods:

### 1. Connected App Authentication (Recommended)

Best for automation and CI/CD pipelines:

```hcl
provider "anypoint" {
  client_id     = "your-connected-app-client-id"
  client_secret = "your-connected-app-client-secret"
  base_url      = "https://anypoint.mulesoft.com"
  auth_type     = "connected_app"  # default
}
```

### 2. User Authentication

Required for operations that need user context (e.g., connected app scope management):

```hcl
provider "anypoint" {
  client_id     = "your-connected-app-client-id"
  client_secret = "your-connected-app-client-secret"
  username      = "your-username"
  password      = "your-password"
  base_url      = "https://anypoint.mulesoft.com"
  auth_type     = "user"
}
```

### Regional Base URLs

- **US Control Plane:** `https://anypoint.mulesoft.com`
- **EU Control Plane:** `https://eu1.anypoint.mulesoft.com`
- **Government Cloud:** `https://gov.anypoint.mulesoft.com`
- **Staging (Testing):** `https://stgx.anypoint.mulesoft.com`

## Resources Overview

The provider supports **37 resources** across **5 main categories**:

| Category | Resources | Description |
|----------|-----------|-------------|
| **Access Management** | 11 | Organizations, environments, users, teams, roles, and connected apps |
| **API Management** | 6 | API instances, policies, SLA tiers, alerts, and Flex Gateways |
| **CloudHub 2.0** | 11 | Private spaces, networks, VPNs, firewalls, and TLS contexts |
| **Governance** | 1 | API governance profiles and conformance validation |
| **Secrets Management** | 8 | Certificates, keystores, truststores, and TLS contexts |

## Provider Configuration

### Full Configuration Options

```hcl
provider "anypoint" {
  # Authentication - Connected App
  client_id     = var.anypoint_client_id     # Required
  client_secret = var.anypoint_client_secret # Required

  # Authentication - User (optional, for user-context operations)
  username      = var.anypoint_username      # Optional
  password      = var.anypoint_password      # Optional

  # Platform Configuration
  base_url      = var.anypoint_base_url      # Optional, defaults to US control plane
  auth_type     = "connected_app"            # Optional: "connected_app" or "user"
}
```

### Environment Variables

You can also configure the provider using environment variables:

```bash
export ANYPOINT_CLIENT_ID="your-client-id"
export ANYPOINT_CLIENT_SECRET="your-client-secret"
export ANYPOINT_USERNAME="your-username"         # Optional
export ANYPOINT_PASSWORD="your-password"         # Optional
export ANYPOINT_BASE_URL="https://anypoint.mulesoft.com"
```

##  Quick Start Examples

### Create a Sub-Organization

```hcl
resource "anypoint_organization" "dev_org" {
  name              = "Development Organization"
  parent_id         = var.parent_organization_id
  owner_id          = var.owner_user_id
  is_federated      = false
  session_timeout   = 240
  default_vCores    = 1.0
}
```

### Create an Environment

```hcl
resource "anypoint_environment" "dev" {
  organization_id = anypoint_organization.dev_org.id
  name            = "Development"
  type            = "sandbox"
  is_production   = false
}
```

### Deploy a Private Space

```hcl
resource "anypoint_private_space" "production" {
  organization_id = var.organization_id
  name            = "production-space"
  region          = "us-east-2"

  features {
    enhanced_security = true
    persistent_object_store = true
  }
}
```

### Configure a VPN Connection

```hcl
resource "anypoint_vpn_connection" "on_prem" {
  organization_id  = var.organization_id
  private_space_id = anypoint_private_space.production.id

  name                 = "OnPrem-VPN"
  remote_ip_address    = "203.0.113.5"
  remote_asn           = 65000
  local_asn            = 64512
  static_routes        = ["10.0.0.0/16"]

  tunnels {
    psk                 = var.vpn_psk_1
    ptp_cidr            = "169.254.1.0/30"
  }
}
```

##  Complete Resource List

###  Access Management Resources

| Resource | Description |
|----------|-------------|
| `anypoint_organization` | Manage Anypoint organizations and sub-organizations |
| `anypoint_environment` | Create and manage environments (sandbox/production) |
| `anypoint_user` | Manage platform users and their properties |
| `anypoint_team` | Create teams for organizing users |
| `anypoint_team_members` | Assign users to teams |
| `anypoint_team_roles` | Assign roles to teams |
| `anypoint_rolegroup` | Create custom role groups |
| `anypoint_rolegroup_roles` | Assign roles to role groups |
| `anypoint_rolegroup_users` | Assign users to role groups |
| `anypoint_connectedapp` | Create connected applications for API authentication |
| `anypoint_connectedapp_scopes` | Manage scopes for connected apps |

**Example:** [Access Management Examples](./examples/accessmanagement)

###  API Management Resources

| Resource | Description |
|----------|-------------|
| `anypoint_api_instance` | Deploy and manage API instances |
| `anypoint_api_policy` | Apply policies to API instances (rate limiting, JWT validation, etc.) |
| `anypoint_api_instance_sla_tier` | Configure SLA tiers for API access control |
| `anypoint_managed_flexgateway` | Deploy managed Flex Gateway instances |

**Example:** [API Management Examples](./examples/apimanagement)

###  CloudHub 2.0 Resources

| Resource | Description |
|----------|-------------|
| `anypoint_private_space` | Create isolated runtime environments in CloudHub 2.0 |
| `anypoint_private_network` | Configure private networking for spaces |
| `anypoint_vpn_connection` | Establish site-to-site VPN connections |
| `anypoint_firewall_rules` | Define ingress firewall rules for private spaces |
| `anypoint_tls_context` | Configure TLS/SSL for private space ingress |
| `anypoint_transit_gateway` | Connect to AWS Transit Gateways |
| `anypoint_private_space_connection` | Manage private space connections |
| `anypoint_private_space_association` | Associate environments with private spaces |
| `anypoint_private_space_advanced_config` | Configure advanced private space settings |
| `anypoint_private_space_upgrade` | Schedule private space upgrades |
| `anypoint_privatespace_advanced_config` | Advanced configuration for private spaces |

**Example:** [CloudHub 2.0 Examples](./examples/cloudhub2)

###  Governance Resources

| Resource | Description |
|----------|-------------|
| `anypoint_api_governance_profile` | Define API governance rules and conformance validation |

**Example:** [Governance Examples](./examples/governance)

###  Secrets Management Resources

| Resource | Description |
|----------|-------------|
| `anypoint_secret_group` | Create groups for organizing secrets |
| `anypoint_secret_group_certificate` | Store and manage TLS certificates |
| `anypoint_secret_group_certificate_pinset` | Configure certificate pinning |
| `anypoint_secret_group_keystore` | Manage keystores (PEM, JKS, PKCS12, JCEKS) |
| `anypoint_secret_group_truststore` | Manage truststores for certificate validation |
| `anypoint_secret_group_shared_secret` | Store shared secrets and credentials |
| `anypoint_flex_tls_context` | Configure TLS contexts for Flex Gateway |
| `anypoint_secret_group_tls_context` | Configure TLS contexts in secret groups |

**Example:** [Secrets Management Examples](./examples/secretsmanagement)

##  Data Sources

The provider also includes data sources for reading existing resources:

- `anypoint_user_data` - Read user information
- `anypoint_environment_data` - Read environment details
- `anypoint_organization_data` - Read organization information
- And more...

##  Examples

Comprehensive examples are available in the [`examples/`](./examples) directory:

### Basic Examples by Category

- **[Access Management](./examples/accessmanagement)** - Users, teams, roles, and organizations
- **[API Management](./examples/apimanagement)** - API instances, policies, and monitoring
- **[CloudHub 2.0](./examples/cloudhub2)** - Private spaces, networking, and security
- **[Governance](./examples/governance)** - API governance profiles
- **[Secrets Management](./examples/secretsmanagement)** - Certificates and secure storage
- **[Connected App Scopes](./examples/connected_app_scopes)** - Managing connected app permissions

### Complete End-to-End Examples

- **[Sub-Org with Private Space](./examples/e2e/suborg_with_privatespace_complete.tf)** - Complete workflow creating a sub-organization with private space, networking, and security
- **[Comprehensive E2E](./examples/comprehensive-e2e)** - Full platform setup including all major resources

### Authentication Examples

- **[Connected App - Own Behalf](./examples/auth_types/connected_app_on_own_behalf)** - Service-to-service authentication
- **[Connected App - User Behalf](./examples/auth_types/connected_app_on_user_behalf)** - Delegated user authentication

## Documentation

### Quick References

- **[Provider Resources CRUD APIs](./examples/e2e/provider_resources_crud_apis.csv)** - Client method reference
- **[Provider REST APIs](./examples/e2e/provider_resources_rest_apis.csv)** - Complete HTTP endpoint reference
- **[API Endpoints Reference](./examples/e2e/provider_api_endpoints_reference.md)** - Detailed API documentation
- **[Resources Summary](./examples/e2e/provider_resources_summary.md)** - Overview and statistics
- **[Examples Update Summary](./examples/EXAMPLES_UPDATE_SUMMARY.md)** - Default configuration guide

### Testing Documentation

- **[Testing Framework](./TESTING_FRAMEWORK.md)** - Guide to acceptance and integration tests
- **[Test Organization](./TEST_ORGANIZATION.md)** - Test structure and patterns
- **[Pre-Flight Checklist](./PRE_FLIGHT_CHECKLIST.md)** - Testing checklist

### Development Guides

- **[Client Quick Start](./CLIENT_QUICK_START.md)** - Getting started with client development
- **[Client SOP](./CLIENT_SOP.md)** - Standard operating procedures
- **[Client Distribution Guide](./CLIENT_DISTRIBUTION_GUIDE.md)** - How to distribute the provider

##  Common Use Cases

### 1. Multi-Region Deployment

```hcl
module "us_deployment" {
  source = "./modules/cloudhub2-space"

  region           = "us-east-2"
  organization_id  = var.organization_id
  space_name       = "us-production"
}

module "eu_deployment" {
  source = "./modules/cloudhub2-space"

  region           = "eu-central-1"
  organization_id  = var.organization_id
  space_name       = "eu-production"
}
```

### 2. Multi-Environment Setup

```hcl
# Development Environment
resource "anypoint_environment" "dev" {
  name = "Development"
  type = "sandbox"
}

# Staging Environment
resource "anypoint_environment" "staging" {
  name = "Staging"
  type = "sandbox"
}

# Production Environment
resource "anypoint_environment" "prod" {
  name = "Production"
  type = "production"
}
```

### 3. Secure Hybrid Connectivity

```hcl
# Private Space
resource "anypoint_private_space" "secure_space" {
  name   = "secure-production"
  region = "us-east-1"
}

# Private Network
resource "anypoint_private_network" "internal" {
  private_space_id = anypoint_private_space.secure_space.id
  cidr_block       = "10.0.0.0/16"
}

# VPN to On-Premises
resource "anypoint_vpn_connection" "datacenter" {
  private_space_id  = anypoint_private_space.secure_space.id
  name              = "Corporate-Datacenter"
  remote_ip_address = var.datacenter_public_ip
  static_routes     = ["192.168.0.0/16"]

  tunnels {
    psk      = var.vpn_psk
    ptp_cidr = "169.254.1.0/30"
  }
}

# Firewall Rules
resource "anypoint_firewall_rules" "ingress" {
  private_space_id = anypoint_private_space.secure_space.id

  rules {
    cidr_block = "10.0.0.0/8"
    protocol   = "tcp"
    from_port  = 443
    to_port    = 443
  }
}
```

##  Advanced Configuration

### Multiple Provider Configurations

Use provider aliases for managing multiple organizations or environments:

```hcl
# Admin provider for privileged operations
provider "anypoint" {
  alias         = "admin"
  client_id     = var.admin_client_id
  client_secret = var.admin_client_secret
  username      = var.admin_username
  password      = var.admin_password
  auth_type     = "user"
}

# Standard provider for regular operations
provider "anypoint" {
  alias         = "standard"
  client_id     = var.standard_client_id
  client_secret = var.standard_client_secret
}

# Use admin provider for organization creation
resource "anypoint_organization" "sub_org" {
  provider = anypoint.admin
  name     = "New Sub-Organization"
}
```

### Dynamic Configuration with Workspaces

```hcl
locals {
  environment_config = {
    dev = {
      base_url = "https://stgx.anypoint.mulesoft.com"
      org_id   = "dev-org-id"
    }
    prod = {
      base_url = "https://anypoint.mulesoft.com"
      org_id   = "prod-org-id"
    }
  }

  current_env = local.environment_config[terraform.workspace]
}

provider "anypoint" {
  base_url = local.current_env.base_url
}
```

## 🛠️ Troubleshooting

### Common Issues

**Authentication Failures:**
```bash
# Verify credentials
terraform plan -var="anypoint_client_id=YOUR_ID" -var="anypoint_client_secret=YOUR_SECRET"

# Enable debug logging
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform-debug.log
terraform plan
```

**Resource Not Found:**
- Ensure the resource exists in the specified organization/environment
- Check that your connected app has the necessary scopes
- Verify the organization_id and environment_id parameters

**Rate Limiting:**
- Implement delays between resource operations
- Use `depends_on` to control resource creation order
- Consider batching operations when possible

##  Contributing

We welcome contributions! Here's how you can help:

1. **Report Bugs:** Open an issue describing the bug and how to reproduce it
2. **Suggest Features:** Open an issue describing the feature and its use case
3. **Submit PRs:** Fork the repo, make your changes, and submit a pull request

Please see our [Contributing Guide](./CONTRIBUTING.md) for detailed guidelines.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/mulesoft/terraform-provider-anypoint.git
cd terraform-provider-anypoint

# Install dependencies
go mod download

# Build the provider
make build

# Run tests
make test

# Run acceptance tests (requires valid credentials)
make testacc
```

##  Version History

See [CHANGELOG.md](./CHANGELOG.md) for a detailed version history.

##  License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

##  Acknowledgments

- MuleSoft and Salesforce teams for the Anypoint Platform APIs
- HashiCorp for the Terraform Plugin Framework
- All contributors who have helped improve this provider

##  Support

- **Documentation:** [Anypoint Platform Documentation](https://docs.mulesoft.com/)
- **Issues:** [GitHub Issues](https://github.com/mulesoft/terraform-provider-anypoint/issues)
- **Community:** [MuleSoft Community Forums](https://help.mulesoft.com/)

##  Related Projects

- [Anypoint CLI](https://docs.mulesoft.com/anypoint-cli/) - Command-line interface for Anypoint Platform
- [Anypoint Platform APIs](https://anypoint.mulesoft.com/exchange/) - Official API documentation
- [Terraform Registry](https://registry.terraform.io/) - Terraform provider registry

---

**Built for the MuleSoft Community**
