# Anypoint Private Network Resource

This resource creates and configures a private network for an Anypoint Private Space using the simplified API.

## API Reference

**Method:** PATCH  
**URL:** `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}`

### Request Payload

```json
{
  "network": {
    "region": "us-east-1",
    "cidrBlock": "10.0.0.0/18",
    "reservedCidrs": ["192.168.0.0/16"]
  }
}
```

### Response

The response includes the complete private space configuration with network details:

```json
{
  "id": "93b81521-9fcc-429a-9f29-976d65d1e929",
  "name": "example-private-space-2",
  "network": {
    "region": "us-east-1",
    "cidrBlock": "10.0.0.0/18",
    "reservedCidrs": ["192.168.0.0/16"],
    "inboundStaticIps": [],
    "inboundInternalStaticIps": [],
    "outboundStaticIps": [],
    "dnsTarget": "kzthnx.usa-e1.qax.cloudhub.io"
  },
  "managedFirewallRules": [...],
  "firewallRules": [...],
  "logForwarding": {...},
  "ingressConfiguration": {...}
}
```

## Usage

```hcl
resource "anypoint_private_network" "example" {
  private_space_id = anypoint_private_space.example.id
  organization_id  = var.organization_id  # Optional: specify target organization
  region           = "us-east-1"
  cidr_block       = "10.0.0.0/18"
  reserved_cidrs   = ["192.168.0.0/16"]
}
```

## Configuration Arguments

- `private_space_id` - (Required) The ID of the private space to configure the network for
- `organization_id` - (Optional) The ID of the target organization (defaults to provider's organization)
- `region` - (Required) The AWS region where the network will be created
- `cidr_block` - (Required) The CIDR block for the private network (e.g., "10.0.0.0/18")
- `reserved_cidrs` - (Optional) List of CIDR blocks to reserve (e.g., ["192.168.0.0/16"])

## Computed Attributes

- `id` - The ID of the private space
- `name` - The name of the private space
- `dns_target` - The DNS target for the private network
- `inbound_static_ips` - List of inbound static IP addresses
- `inbound_internal_static_ips` - List of inbound internal static IP addresses
- `outbound_static_ips` - List of outbound static IP addresses

## Example

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

# First create a private space
resource "anypoint_private_space" "example" {
  name   = "example-private-space-with-network"
  region = "us-east-1"
}

# Then configure its network
resource "anypoint_private_network" "example" {
  private_space_id = anypoint_private_space.example.id
  organization_id  = var.organization_id  # Optional: specify target organization
  region           = "us-east-1"
  cidr_block       = "10.0.0.0/18"
  reserved_cidrs   = ["192.168.0.0/16"]
}

# Outputs
output "private_network_id" {
  description = "The ID of the private network"
  value       = anypoint_private_network.example.id
}

output "private_network_dns_target" {
  description = "The DNS target for the private network"
  value       = anypoint_private_network.example.dns_target
}

output "private_network_inbound_static_ips" {
  description = "The inbound static IPs for the private network"
  value       = anypoint_private_network.example.inbound_static_ips
}
```

## Import

Private networks can be imported using the private space ID:

```sh
terraform import anypoint_private_network.example private-space-id
```

## Notes

- The private network is configured as part of the private space through the PATCH API
- The network configuration includes region, CIDR block, and reserved CIDRs
- Additional features like firewall rules, log forwarding, and ingress configuration are automatically configured by the API
- The resource depends on an existing private space and will configure its network settings 