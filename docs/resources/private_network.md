---
page_title: "anypoint_private_network Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Manages an Anypoint Private Network configuration.
---

# anypoint_private_network (Resource)

Manages an Anypoint Private Network configuration.

## Example Usage

```terraform
resource "anypoint_private_network" "example" {
  private_space_id = anypoint_private_space.example.id
  region           = "us-east-1"
  cidr_block       = "10.0.0.0/22"
}

# With a specific organization and reserved CIDRs
resource "anypoint_private_network" "custom_org" {
  private_space_id = "your-private-space-id"
  organization_id  = "your-organization-id"
  region           = "us-east-1"
  cidr_block       = "10.0.0.0/18"
  reserved_cidrs   = ["10.0.0.0/24", "10.0.1.0/24"]
}
```

## Schema

### Required

- `private_space_id` (String) The ID of the private space this network belongs to.
- `region` (String) The region for the private network.
- `cidr_block` (String) The CIDR block for the private network.

### Optional

- `organization_id` (String) The organization ID where the private space is located. If not provided, the organization ID will be inferred from the connected app credentials.
- `reserved_cidrs` (List of String) The reserved CIDR blocks for the private network.

### Read-Only

- `id` (String) The unique identifier for the private network.
- `name` (String) The name of the private network.
- `inbound_static_ips` (List of String) The inbound static IPs for the private network.
- `inbound_internal_static_ips` (List of String) The inbound internal static IPs for the private network.
- `outbound_static_ips` (List of String) The outbound static IPs for the private network.
- `dns_target` (String) The DNS target for the private network.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_private_network.example <private_space_id>
```
