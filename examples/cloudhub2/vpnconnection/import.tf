# Import an existing VPN connection into Terraform.
#
# Steps:
#   1. Copy this file to import.tf (or paste the block into your existing .tf files)
#   2. Replace the placeholders with your actual IDs
#   3. Add a matching resource block, or run:
#        terraform plan -generate-config-out=generated.tf
#   4. Run: terraform apply
#
# Import ID format:
#   Root org:  anypoint_vpn_connection -> <private_space_id>/<connection_id>
#   Sub-org:   anypoint_vpn_connection -> <org_id>/<private_space_id>/<connection_id>
#
# Use the <org_id>/<private_space_id>/<connection_id> format when the private space was
# created in a Business Group (sub-org) rather than the root organization.

# --- Root org (2-part ID) ---
# locals {
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   connection_id    = "c9f3a2b1-1234-5678-90ab-cdef01234567"
# }
#
# import {
#   to = anypoint_vpn_connection.imported
#   id = "${local.private_space_id}/${local.connection_id}"
# }

# --- Sub-org (3-part ID) ---
# locals {
#   org_id           = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   connection_id    = "c9f3a2b1-1234-5678-90ab-cdef01234567"
# }
#
# import {
#   to = anypoint_vpn_connection.imported
#   id = "${local.org_id}/${local.private_space_id}/${local.connection_id}"
# }

# resource "anypoint_vpn_connection" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = local.org_id
#   name             = "<connection_name>"
#
#   vpns = [
#     {
#       local_asn         = <local_asn>
#       remote_asn        = <remote_asn>
#       remote_ip_address = "<remote_ip_address>"
#       static_routes     = []
#
#       vpn_tunnels = [
#         {
#           psk            = "<pre_shared_key_1>"
#           ptp_cidr       = "<point_to_point_cidr_1>"
#           startup_action = "start"
#         },
#         {
#           psk            = "<pre_shared_key_2>"
#           ptp_cidr       = "<point_to_point_cidr_2>"
#           startup_action = "start"
#         }
#       ]
#     }
#   ]
# }

# output "imported_id" {
#   value = anypoint_vpn_connection.imported.id
# }
