# Import an existing Exchange asset version into Terraform state.
#
# Import ID format:
#   anypoint_exchange_asset -> <group_id>/<asset_id>/<version>
#   (group_id is usually the organization ID)
#
# On import the provider reads the live asset and seeds the immutable fields
# (classifier, main_file, api_version) and external instances into state, so the
# first plan after import shows zero drift. The local file_path settles on the
# next apply without recreating the asset.
#
# Steps:
#   1. Replace the placeholders with your actual GAV coordinates.
#   2. Uncomment the import block and the resource block below.
#   3. Run: terraform plan -generate-config-out=generated.tf   (Terraform >= 1.5)
#      OR:  terraform import anypoint_exchange_asset.imported <group_id>/<asset_id>/<version>

# import {
#   to = anypoint_exchange_asset.imported
#   id = "<group_id>/<asset_id>/<version>"  # e.g. "6c3c4eb3-.../tf-demo-rest-api/1.0.0"
# }

# resource "anypoint_exchange_asset" "imported" {
#   organization_id = var.org_id
#   group_id        = var.org_id
#   asset_id        = "<asset_id>"
#   version         = "<version>"
#   name            = "<name>"
# }

# output "imported_asset_id" {
#   value = anypoint_exchange_asset.imported.id
# }
