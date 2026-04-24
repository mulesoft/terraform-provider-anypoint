# Access Management Examples

This directory contains examples for managing Anypoint Platform Access Management resources using Terraform.

## Available Examples

### [Environment](./environment/)
- **Resource**: `anypoint_environment`
- **Data Source**: `anypoint_environment`
- **Description**: Manage Anypoint Platform environments (development, production, etc.)
- **API**: `/accounts/api/organizations/{orgId}/environments`

### [Organization](./organization/)
- **Resource**: `anypoint_organization`
- **Data Source**: `anypoint_organization`
- **Description**: Manage Anypoint Platform organizations
- **API**: `/accounts/api/organizations`

### [Team](./team/)
- **Resource**: `anypoint_team`
- **Data Source**: `anypoint_team`
- **Description**: Manage Anypoint Platform teams with hierarchical support
- **API**: `/accounts/api/organizations/{orgId}/teams`

### [User](./user/)
- **Resource**: `anypoint_user`
- **Data Source**: `anypoint_user`
- **Description**: Manage Anypoint Platform users and their attributes
- **API**: `/accounts/api/organizations/{orgId}/users`

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

## Quick Start

1. Navigate to any example directory
2. Copy `terraform.tfvars.example` to `terraform.tfvars`
3. Fill in your Anypoint Platform credentials and required IDs
4. Run:
   ```bash   
   terraform plan
   terraform apply
   ```

## API Documentation

For detailed API documentation, visit:
- [Anypoint Platform Access Management API](https://docs.mulesoft.com/access-management/) 