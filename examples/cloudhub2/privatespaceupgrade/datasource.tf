# Example: Using the Private Space Upgrade Data Source independently
# This can be used to check the upgrade status of an existing private space

# data "anypoint_private_space_upgrade" "existing_upgrade" {
#   private_space_id = "ed75f94a-9a73-430b-9c8a-d838681c4ecf"
# }

# # Output the current upgrade status
# output "existing_upgrade_status" {
#   value = {
#     id                    = data.anypoint_private_space_upgrade.existing_upgrade.id
#     private_space_id      = data.anypoint_private_space_upgrade.existing_upgrade.private_space_id
#     scheduled_update_time = data.anypoint_private_space_upgrade.existing_upgrade.scheduled_update_time
#     status                = data.anypoint_private_space_upgrade.existing_upgrade.status
#   }
#   description = "Current upgrade status for the private space"
# }