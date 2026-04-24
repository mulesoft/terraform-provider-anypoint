# Anypoint Terraform Provider Examples

This directory contains comprehensive examples for using the Anypoint Terraform provider to manage Anypoint Platform resources.

## Directory Structure

The examples are organized by service category to match the provider's internal structure:

```
examples/
├── accessmanagement/     # Access Management resources
│   ├── environment/      # Environment management
│   ├── organization/     # Organization management
│   ├── rolegroup/        # Role group management
│   ├── team/            # Team management
│   └── user/            # User management
├── apimanagement/       # API Management resources
│   ├── api_instance/    # API instance management
│   ├── managed_flexgateway/  # Flex Gateway management
│   ├── policies/        # API policy examples
│   └── slatier/         # SLA tier configuration
├── auth_types/          # Authentication method examples
│   ├── connected_app_on_own_behalf/    # Connected app authentication
│   └── connected_app_on_user_behalf/   # User-based authentication
├── connected_app_scopes/ # Connected app scope examples
├── cloudhub2/           # CloudHub 2.0 resources
│   ├── firewallrules/   # Firewall rules management
│   ├── privatenetwork/  # Network configuration
│   ├── privatespace/    # Private space management
│   ├── privatespaceadvancedconfig/  # Advanced private space configuration
│   ├── privatespaceassociation/     # Private space associations
│   ├── privatespaceconnection/      # VPN and connections
│   ├── privatespaceupgrade/         # Private space upgrades
│   ├── tlscontext/      # TLS/SSL configuration
│   ├── transitgateway/  # Transit gateway management
│   └── vpnconnection/   # VPN connection management
├── governance/          # API Governance resources
└── secretsmanagement/   # Secrets Management resources
```

## Categories

### 🏢 [Access Management](./accessmanagement/)
Manage users, teams, organizations, and environments within Anypoint Platform.

**Resources:**
- **Environment** - Development, staging, production environments
- **Organization** - Business group and organization hierarchy
- **Role Group** - Role group management and user/role assignments
- **Team** - Team management with parent-child relationships
- **User** - User accounts and profile management

**Use Cases:**
- Setting up organizational structure
- Managing user access and permissions
- Creating environment hierarchies
- Team-based access control

### 🌐 [API Management](./apimanagement/)
Manage API instances, policies, SLA tiers, and Flex Gateways.

**Resources:**
- **API Instance** - Deploy and manage API instances
- **Managed Flex Gateway** - Deploy managed Flex Gateway instances
- **Policies** - Apply security and traffic management policies
- **SLA Tier** - Configure rate limits and access tiers for API consumers

**Use Cases:**
- API lifecycle management
- Implementing API security policies (JWT, OAuth, rate limiting)
- Creating tiered API access levels
- Managing Flex Gateway deployments

### ☁️ [CloudHub 2.0](./cloudhub2/)
Manage CloudHub 2.0 infrastructure including private spaces, networking, and security.

**Resources:**
- **Private Space** - Dedicated runtime environments
- **Private Network** - Network configuration and connectivity
- **Private Space Advanced Config** - Advanced configuration settings
- **Private Space Association** - Resource associations and linking
- **Private Space Connection** - VPN and direct connections
- **Private Space Upgrade** - Upgrade management and versioning
- **TLS Context** - SSL/TLS certificate management
- **Transit Gateway** - Network transit gateway management
- **VPN Connection** - VPN connection configuration
- **Firewall Rules** - Inbound and outbound traffic control

**Use Cases:**
- Setting up secure private clouds
- Configuring network connectivity
- Managing SSL certificates
- Implementing security policies

## Getting Started

### Prerequisites
- Terraform >= 1.0
- Anypoint Platform account with appropriate permissions
- Valid Anypoint Platform credentials

### Quick Start
1. **Choose a category** that matches your needs
2. **Navigate to the specific example** directory
3. **Copy the example configuration**:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```
4. **Configure your credentials** in `terraform.tfvars`
5. **Initialize and apply**:
   ```bash   
   terraform plan
   terraform apply
   ```

### Common Configuration

All examples use this provider configuration pattern:

```hcl
terraform {
  required_providers {
    anypoint = {
      source = "example.com/ankitsarda/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}
```

## Example Workflows

### 🏗️ **Complete Infrastructure Setup**
1. Start with [organization](./accessmanagement/organization/) setup
2. Create [environments](./accessmanagement/environment/) for different stages
3. Set up [teams](./accessmanagement/team/) and [users](./accessmanagement/user/)
4. Deploy [private spaces](./cloudhub2/privatespace/) for CloudHub 2.0
5. Configure [networking](./cloudhub2/privatenetwork/) and [security](./cloudhub2/firewallrules/)

### 🔧 **Development Environment**
1. Create a development [environment](./accessmanagement/environment/)
2. Set up [role groups](./accessmanagement/rolegroup/) for access control
3. Set up a small [private space](./cloudhub2/privatespace/)
4. Configure basic [firewall rules](./cloudhub2/firewallrules/)

### 🏭 **Production Environment**
1. Create production [organization](./accessmanagement/organization/) structure
2. Configure [role groups](./accessmanagement/rolegroup/) and permissions
3. Deploy production [private space](./cloudhub2/privatespace/)
4. Configure [advanced settings](./cloudhub2/privatespaceadvancedconfig/)
5. Set up [TLS contexts](./cloudhub2/tlscontext/) for security
6. Configure [VPN connections](./cloudhub2/vpnconnection/) and [transit gateways](./cloudhub2/transitgateway/)
7. Implement comprehensive [firewall rules](./cloudhub2/firewallrules/)

## Best Practices

### Security
- Use environment variables or secure variable files for credentials
- Implement least-privilege access policies
- Regularly rotate API credentials
- Use TLS contexts for encrypted communications

### Organization
- Use consistent naming conventions across resources
- Implement proper resource tagging where supported
- Document your infrastructure with clear variable descriptions
- Use modules for repeated patterns

### Development
- Start with development environments before production
- Use `terraform plan` to review changes before applying
- Implement proper state management for team collaboration
- Use version control for your Terraform configurations

## Support

- **Documentation**: Each example includes detailed README files
- **API Reference**: See the main [provider documentation](../README.md)
- **Issues**: Report issues in the main repository
- **Community**: Join discussions in the Anypoint Platform community

## Contributing

We welcome contributions to improve these examples:
1. Fork the repository
2. Create examples following the established patterns
3. Include proper documentation and variable files
4. Test your examples thoroughly
5. Submit a pull request

---

For detailed information about each resource type, refer to the category-specific README files in the subdirectories. 