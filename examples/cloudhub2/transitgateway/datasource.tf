###############################################################################
# Data Sources
# ============
# Read existing Transit Gateway connections without managing them.
###############################################################################

# List every Transit Gateway connection attached to a Private Space.
data "anypoint_transit_gateway_connections" "all" {
  organization_id  = var.organization_id
  private_space_id = var.private_space_id
}

output "all_transit_gateway_connections" {
  description = "All TGW connections (id, name, status, routes) on the Private Space."
  value       = data.anypoint_transit_gateway_connections.all.transit_gateway_connections
}

# Look up a single Transit Gateway connection by its ID.
data "anypoint_transit_gateway_connection" "one" {
  organization_id  = var.organization_id
  private_space_id = var.private_space_id
  id               = anypoint_transit_gateway_connection.main.id
}

output "one_transit_gateway_status" {
  description = "Status of the looked-up connection (e.g. Pending, Available)."
  value       = data.anypoint_transit_gateway_connection.one.status
}

output "one_transit_gateway_aws_id" {
  description = "AWS Transit Gateway ID of the looked-up connection."
  value       = data.anypoint_transit_gateway_connection.one.aws_transit_gateway_id
}


# ---------------------------------------------------------------------------
# Example: discover the private spaces a transit gateway can attach to.
#
# Use this to find the private_space_id required by
# anypoint_transit_gateway_connection and by both transit gateway data sources.
# ---------------------------------------------------------------------------
data "anypoint_private_spaces" "all" {
  organization_id = var.organization_id
}

output "private_space_targets" {
  description = "Private spaces available as transit gateway attachment targets."
  value = [for ps in data.anypoint_private_spaces.all.private_spaces : {
    id     = ps.id
    name   = ps.name
    region = ps.region
  }]
}
