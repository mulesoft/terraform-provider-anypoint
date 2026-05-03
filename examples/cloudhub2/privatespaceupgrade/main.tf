terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Example: Private Space Upgrade Resource
# This schedules an upgrade for the specified date
# To cancel the upgrade, simply destroy this resource: terraform destroy -target=anypoint_private_space_upgrade.example
resource "anypoint_private_space_upgrade" "example" {
  private_space_id = var.private_space_id
  organization_id = var.organization_id
  date             = "2025-09-12"
  opt_in           = true
}

# Example: Private Space Upgrade Data Source
# This can be used to check the current upgrade status
# Useful for monitoring the upgrade progress or checking status independently
data "anypoint_private_space_upgrade" "current_status" {
  private_space_id = "f644726a-d0e4-4c60-a1bb-3996543cd56f"
  
  # Optionally depend on the resource to ensure it's created first
  depends_on = [anypoint_private_space_upgrade.example]
}

# Output the upgrade details from the resource
output "upgrade_resource_details" {
  value = {
    id                    = anypoint_private_space_upgrade.example.id
    private_space_id      = anypoint_private_space_upgrade.example.private_space_id
    date                  = anypoint_private_space_upgrade.example.date
    opt_in                = anypoint_private_space_upgrade.example.opt_in
    scheduled_update_time = anypoint_private_space_upgrade.example.scheduled_update_time
    status                = anypoint_private_space_upgrade.example.status
  }
}

# # Output the upgrade status from the data source
output "upgrade_data_source_status" {
  value = {
    id                    = data.anypoint_private_space_upgrade.current_status.id
    private_space_id      = data.anypoint_private_space_upgrade.current_status.private_space_id
    scheduled_update_time = data.anypoint_private_space_upgrade.current_status.scheduled_update_time
    status                = data.anypoint_private_space_upgrade.current_status.status
  }
}

# Example of conditional upgrade cancellation based on external conditions
# Uncomment the lifecycle block below to prevent accidental deletion
/*
resource "anypoint_private_space_upgrade" "conditional_example" {
  private_space_id = "f644726a-d0e4-4c60-a1bb-3996543cd56f"
  date             = "2025-08-12"
  opt_in           = true

  # Prevent accidental deletion/cancellation
  lifecycle {
    prevent_destroy = true
  }
}
*/ 