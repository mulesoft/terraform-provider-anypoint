# Anypoint Transit Gateway Resource

This resource creates and manages a Transit Gateway in a CloudHub 2.0 private space.

## API Reference

**Method:** POST  
**URL:** `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/transitgateways`

### Request Payload

```json
{
  "name": "asdsadsa",
  "resourceShareId": "e8e330a8-4f8c-452b-afd0-7810c41287f1",
  "resourceShareAccount": "055970264539",
  "routes": ["10.0.0.0/8"]
}
```

### Response

The API returns a Transit Gateway with detailed specification and status:

```json
{
  "id": "83d77850-04ee-4368-8122-192f760913de",
  "name": "rtf-kamboocha",
  "spec": {
    "resourceShare": {
      "id": "5e409a9d-49a7-456c-82d7-a6254738a18d",
      "account": "25102306"
    },
    "region": "us-east-1",
    "networkIds": [
      "network-deployment-id1"
    ],
    "spaceName": "space"
  },
  "status": {
    "gateway": "unknown",
    "attachment": "unattached",
    "tgwResource": "http://aws.tgw.link.com",
    "routes": [
      "10.0.0.0/21",
      "10.0.0.0/22"
    ]
  }
}
```

## Usage

```hcl
resource "anypoint_transit_gateway" "example" {
  private_space_id         = var.private_space_id
  name                     = "my-transit-gateway"
  resource_share_id        = "e8e330a8-4f8c-452b-afd0-7810c41287f1"
  resource_share_account   = "055970264539"
  routes                   = ["10.0.0.0/8", "172.16.0.0/12"]
}
```

## Configuration Arguments

### Required Arguments

- `private_space_id` - (Required) The ID of the private space where the Transit Gateway will be created
- `name` - (Required) The name of the Transit Gateway
- `resource_share_id` - (Required) The resource share ID for the Transit Gateway
- `resource_share_account` - (Required) The resource share account for the Transit Gateway
- `routes` - (Required) List of route CIDR blocks for the Transit Gateway

## Computed Attributes

### Top-level Computed Attributes

- `id` - The unique identifier for the Transit Gateway

### Spec Computed Attributes

- `spec` - The specification of the Transit Gateway containing:
  - `resource_share` - Resource share information with `id` and `account`
  - `region` - The AWS region of the Transit Gateway
  - `network_ids` - List of associated network IDs
  - `space_name` - The name of the private space

### Status Computed Attributes

- `status` - The status of the Transit Gateway containing:
  - `gateway` - Gateway status (e.g., "unknown", "available")
  - `attachment` - Attachment status (e.g., "unattached", "attached")
  - `tgw_resource` - AWS Transit Gateway resource link
  - `routes` - List of active routes

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

resource "anypoint_transit_gateway" "example" {
  private_space_id         = var.private_space_id
  name                     = "example-transit-gateway"
  resource_share_id        = "e8e330a8-4f8c-452b-afd0-7810c41287f1"
  resource_share_account   = "055970264539"
  routes                   = ["10.0.0.0/8", "172.16.0.0/12"]
}

# Access computed attributes
output "transit_gateway_region" {
  value = anypoint_transit_gateway.example.spec.region
}

output "gateway_status" {
  value = {
    gateway    = anypoint_transit_gateway.example.status.gateway
    attachment = anypoint_transit_gateway.example.status.attachment
    routes     = anypoint_transit_gateway.example.status.routes
  }
}
```

## Import

Transit Gateways can be imported using the format `private_space_id:transit_gateway_id`:

```sh
terraform import anypoint_transit_gateway.example private-space-id:transit-gateway-id
```

## Notes

- The Transit Gateway enables connectivity between your CloudHub 2.0 private space and AWS resources
- Resource sharing must be set up in AWS before creating the Transit Gateway
- The `resource_share_id` and `resource_share_account` must correspond to valid AWS Resource Access Manager (RAM) shares
- Routes define which traffic will be routed through the Transit Gateway
- Status monitoring allows you to track the provisioning and connectivity state
- The resource supports full CRUD operations (Create, Read, Update, Delete)

## Common Use Cases

1. **Hybrid Cloud Connectivity**: Connect CloudHub 2.0 private spaces to on-premises networks via AWS Transit Gateway
2. **Multi-VPC Access**: Enable communication between private spaces and multiple AWS VPCs
3. **Centralized Routing**: Use Transit Gateway as a central hub for network routing in hybrid architectures
4. **Cross-Account Connectivity**: Connect to AWS resources in different accounts using resource sharing

## Prerequisites

1. **AWS Resource Share**: Set up AWS Resource Access Manager (RAM) to share the Transit Gateway
2. **Account Permissions**: Ensure the resource share account has proper permissions
3. **Network Planning**: Plan your CIDR blocks to avoid conflicts
4. **Private Space**: Have an existing CloudHub 2.0 private space ready

## Monitoring and Troubleshooting

Monitor the Transit Gateway status through computed attributes:

```hcl
# Check gateway status
output "is_gateway_ready" {
  value = anypoint_transit_gateway.example.status.gateway == "available"
}

# Check attachment status
output "is_attached" {
  value = anypoint_transit_gateway.example.status.attachment == "attached"
}

# Monitor active routes
output "active_routes" {
  value = anypoint_transit_gateway.example.status.routes
}
```

Common status values:
- Gateway: `unknown`, `pending`, `available`, `failed`
- Attachment: `unattached`, `attaching`, `attached`, `failed` 