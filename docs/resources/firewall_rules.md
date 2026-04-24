---
page_title: "anypoint_firewall_rules Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Manages firewall rules for an Anypoint Private Space.
---

# anypoint_firewall_rules (Resource)

Manages firewall rules for an Anypoint Private Space.

## Example Usage

```terraform
resource "anypoint_firewall_rules" "example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id

  rules = [
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 0
      to_port    = 65535
      type       = "outbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "inbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 80
      to_port    = 80
      type       = "inbound"
    },
    {
      cidr_block = "local-private-network"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "inbound"
    },
  ]
}
```

## Schema

### Required

- `private_space_id` (String) The ID of the private space for which to manage firewall rules.
- `rules` (Block List) List of firewall rules. See [below for nested schema](#nestedschema--rules).

### Optional

- `organization_id` (String) The organization ID where the private space is located. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) The unique identifier for the firewall rules (same as private_space_id).

<a id="nestedschema--rules"></a>
### Nested Schema for `rules`

Required:

- `cidr_block` (String) The CIDR block for the firewall rule.
- `protocol` (String) The protocol for the firewall rule (tcp, udp, icmp).
- `from_port` (Number) The starting port for the firewall rule.
- `to_port` (Number) The ending port for the firewall rule.
- `type` (String) The type of the firewall rule (inbound, outbound).

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_firewall_rules.example <private_space_id>
```
