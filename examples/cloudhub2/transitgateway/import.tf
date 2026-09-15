# ---------------------------------------------------------------------------
# Import an existing Transit Gateway connection into Terraform state.
#
# Import ID format (composite):
#   organization_id/private_space_id/transit_gateway_id
#
# On import, the current routes are read from the platform and seeded into
# state. For a DETACHED-but-registered connection there are no routes, so the
# zero-diff shape is routes = [].
#
# Option A — import block (Terraform >= 1.5, recommended):
#   1. Uncomment the block below and fill in the placeholders.
#   2. terraform plan   # review the generated diff
#   3. terraform apply
# ---------------------------------------------------------------------------

# import {
#   to = anypoint_transit_gateway_connection.imported
#   id = "<organization_id>/<private_space_id>/<transit_gateway_id>"
# }
#
# resource "anypoint_transit_gateway_connection" "imported" {
#   organization_id        = "<organization_id>"
#   private_space_id       = "<private_space_id>"
#   name                   = "<name>"
#   resource_share_id      = "<resource_share_id>"
#   resource_share_account = "<resource_share_account>"
#   routes                 = ["<cidr>"] # or [] for a detached connection
# }

# ---------------------------------------------------------------------------
# Option B — CLI import (Terraform < 1.5):
#   terraform import anypoint_transit_gateway_connection.imported \
#     <organization_id>/<private_space_id>/<transit_gateway_id>
# ---------------------------------------------------------------------------
