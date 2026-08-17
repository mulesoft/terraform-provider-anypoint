###############################################################################
# Transit Gateway Connection Example
# ==================================
# Attaches an AWS Transit Gateway (shared to MuleSoft via AWS RAM) to an
# Anypoint Private Space, and manages its CIDR routes inline.
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply
#
# Prerequisites:
#   - A Private Space with its network provisioned.
#   - An AWS Transit Gateway shared to MuleSoft's AWS account via AWS RAM
#     (you supply the RAM resource share ID and the owning AWS account ID).
#   - The route CIDRs must NOT overlap the Private Space CIDR.
#
# Authentication: this resource calls the CloudHub 2.0 private-space control
# plane. A client_credentials Connected App works if it has the
# "Cloudhub Organization Admin" (admin:cloudhub) scope; otherwise use
# auth_type = "user" with an equivalently-permissioned user.
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

resource "anypoint_transit_gateway_connection" "main" {
  organization_id        = var.organization_id
  private_space_id       = var.private_space_id
  name                   = var.name
  resource_share_id      = var.resource_share_id      # AWS RAM resource share UUID
  resource_share_account = var.resource_share_account # AWS account that owns the TGW

  # routes is required but may be empty (routes = []). Provide >=1 CIDR that
  # does NOT overlap the Private Space CIDR. Routes are updatable in place —
  # change this list and re-apply to replace the full set.
  routes = var.routes

  # Recommended backstop: if a re-create is ever triggered, stand up the new
  # attachment BEFORE tearing down the old one (avoids a connectivity gap).
  lifecycle {
    create_before_destroy = true
  }
}

output "transit_gateway_status" {
  description = "Lifecycle status of the attachment (e.g. Pending, Available)."
  value       = anypoint_transit_gateway_connection.main.status
}

output "aws_transit_gateway_id" {
  description = "AWS Transit Gateway ID discovered by the platform from the RAM share."
  value       = anypoint_transit_gateway_connection.main.aws_transit_gateway_id
}
