terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

resource "anypoint_firewall_rules" "example" {
  private_space_id = var.private_space_id
  organization_id = var.organization_id

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
      type       = "outbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 80
      to_port    = 80
      type       = "inbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "inbound"
    },
    {
      cidr_block = "local-private-network"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "inbound"
    },
    {
      cidr_block = "local-private-network"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "outbound"
    }
  ]
}

# resource "anypoint_firewall_rules" "example_custom_org" {
#   private_space_id = "120438e2-f49f-4e2b-a75e-f711a61587fe"
#   organization_id = "080f1918-0096-4cac-85b5-b1cd9cdf9260"

#   rules = [
#     {
#       cidr_block = "0.0.0.0/0"
#       protocol   = "tcp"
#       from_port  = 0
#       to_port    = 65535
#       type       = "outbound"
#     },
#     {
#       cidr_block = "0.0.0.0/0"
#       protocol   = "tcp"
#       from_port  = 443
#       to_port    = 443
#       type       = "outbound"
#     },
#     {
#       cidr_block = "0.0.0.0/0"
#       protocol   = "tcp"
#       from_port  = 80
#       to_port    = 80
#       type       = "inbound"
#     },
#     {
#       cidr_block = "0.0.0.0/0"
#       protocol   = "tcp"
#       from_port  = 443
#       to_port    = 443
#       type       = "inbound"
#     },
#     {
#       cidr_block = "local-private-network"
#       protocol   = "tcp"
#       from_port  = 443
#       to_port    = 443
#       type       = "inbound"
#     },
#     {
#       cidr_block = "local-private-network"
#       protocol   = "tcp"
#       from_port  = 443
#       to_port    = 443
#       type       = "outbound"
#     }
#   ]
# }

output "firewall_rules_id" {
  value = anypoint_firewall_rules.example.id
}

output "firewall_rules_private_space_id" {
  value = anypoint_firewall_rules.example.private_space_id
}

output "firewall_rules_count" {
  value = length(anypoint_firewall_rules.example.rules)
} 