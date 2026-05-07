# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "675c4efb-d44e-44cd-ac6f-d5a1128e6236"
resource "anypoint_private_space_config" "imported" {
  enable_egress   = false
  enable_iam_role = false
  firewall_rules = [
    {
      cidr_block = "0.0.0.0/0"
      from_port  = 80
      protocol   = "tcp"
      to_port    = 80
      type       = "inbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      from_port  = 443
      protocol   = "tcp"
      to_port    = 443
      type       = "inbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      from_port  = 0
      protocol   = "tcp"
      to_port    = 65535
      type       = "outbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      from_port  = 443
      protocol   = "tcp"
      to_port    = 443
      type       = "outbound"
    },
  ]
  name            = "demo-private-space"
  organization_id = "<org_id>"
  network {
    cidr_block     = "10.0.0.0/18"
    region         = "us-east-2"
    reserved_cidrs = null
  }
}
