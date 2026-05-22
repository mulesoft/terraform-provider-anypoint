# Import an existing VPN connection into Terraform.
#
# Steps:
#   1. Replace the placeholders with your actual IDs.
#   2. Uncomment the import and resource blocks.
#   3. Run: terraform init && terraform apply
#      OR use CLI: terraform import anypoint_vpn_connection.imported <private_space_id>/<connection_id>
#
# Import ID format:
#   anypoint_vpn_connection -> <private_space_id>/<connection_id>
#   Example: 675c4efb-d44e-44cd-ac6f-d5a1128e6236/a1b2c3d4-e5f6-7890-abcd-ef1234567890

# locals {
#   private_space_id = "<private_space_id>"   # e.g. "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   connection_id    = "<connection_id>"       # e.g. "c9f3a2b1-1234-5678-90ab-cdef01234567"
# }

# import {
#   to = anypoint_vpn_connection.imported
#   id = "${local.private_space_id}/${local.connection_id}"
# }

# resource "anypoint_vpn_connection" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = "<organization_id>"
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
