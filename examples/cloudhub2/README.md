# CloudHub 2.0 Examples

This directory contains examples for managing CloudHub 2.0 resources using Terraform.

## Available Examples

### [Private Space](./privatespace/)
- **Resource**: `anypoint_private_space`
- **Data Source**: `anypoint_privatespace`
- **Description**: Manage CloudHub 2.0 private spaces with regional deployment
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces`

### [Private Network](./privatenetwork/)
- **Resource**: `anypoint_private_network`
- **Data Source**: `anypoint_privatenetwork`
- **Description**: Manage network configurations for private spaces
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}`

### [TLS Context](./tlscontext/)
- **Resource**: `anypoint_tls_context`
- **Data Source**: `anypoint_tlscontext`
- **Description**: Manage TLS/SSL configurations for secure communications
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/tlscontexts`

### [Private Space Connection](./privatespaceconnection/)
- **Resource**: `anypoint_private_space_connection`
- **Data Source**: `anypoint_privatespaceconnection`
- **Description**: Manage VPN and direct connections to private spaces
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections`

### [Firewall Rules](./firewallrules/)
- **Resource**: `anypoint_firewall_rules`
- **Data Source**: `anypoint_firewallrules`
- **Description**: Manage inbound and outbound firewall rules for private spaces
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` (PATCH)

### [Private Space Advanced Configuration](./privatespaceadvancedconfig/)
- **Resource**: `anypoint_privatespace_advanced_config`
- **Description**: Manage advanced configuration settings for private spaces including ingress configuration and IAM role settings
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` (PATCH)

### [Private Space Association](./privatespaceassociation/)
- **Resource**: `anypoint_privatespace_association`
- **Data Source**: `anypoint_privatespaceassociation`
- **Description**: Manage associations between private spaces and other resources
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/associations`

### [Private Space Upgrade](./privatespaceupgrade/)
- **Resource**: `anypoint_privatespace_upgrade`
- **Data Source**: `anypoint_privatespaceupgrade`
- **Description**: Manage private space upgrades and version management
- **API**: `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/upgrade`

### [VPN Connection](./vpnconnection/)
- **Resource**: `anypoint_vpn_connection`
- **Data Source**: `anypoint_vpnconnection`
- **Description**: Manage VPN connections for secure network access
- **API**: `/runtimefabric/api/organizations/{orgId}/vpnconnections`

## Common Setup

All examples in this category require:

1. **Provider Configuration**:
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

2. **Authentication**: Anypoint Platform credentials
3. **Base URL**: `https://anypoint.mulesoft.com`
4. **Private Space**: Most resources require an existing private space

## Quick Start

1. **Start with Private Space**: Create a private space first as most other resources depend on it
2. Navigate to any example directory
3. Copy `terraform.tfvars.example` to `terraform.tfvars`
4. Fill in your Anypoint Platform credentials and required IDs
5. Run:
   ```bash
   terraform plan
   terraform apply
   ```

## Resource Dependencies

```
Private Space (Foundation)
├── Private Network Configuration
├── Private Space Advanced Configuration
├── Private Space Associations
├── TLS Context
├── Firewall Rules
├── VPN Connections
└── Private Space Connections
```

## Common Use Cases

### Complete Private Space Setup
```hcl
# 1. Create private space
resource "anypoint_private_space" "main" {
  name            = "production-space"
  region          = "us-east-1"
  enable_iam_role = true
  enable_egress   = true
}

# 2. Configure firewall rules
resource "anypoint_firewall_rules" "main" {
  private_space_id = anypoint_private_space.main.id
  rules = [
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "inbound"
    }
  ]
}

# 3. Set up TLS context
resource "anypoint_tls_context" "main" {
  private_space_id = anypoint_private_space.main.id
  name             = "Main TLS Context"
  target           = "inbound"
}
```

## API Documentation

For detailed API documentation, visit:
- [CloudHub 2.0 API Documentation](https://docs.mulesoft.com/cloudhub-2/)
- [Runtime Fabric API Reference](https://docs.mulesoft.com/runtime-fabric/) 