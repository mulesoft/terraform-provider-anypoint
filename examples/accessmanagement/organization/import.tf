# Import an existing private space config (space + network + firewall rules) into Terraform.
#
# Steps:
#   1. Copy this file to import.tf (or paste the block into your existing .tf files)
#   2. Replace the placeholder with your actual private space ID
#   3. Add a matching resource block, or run:
#        terraform plan -generate-config-out=generated.tf
#   4. Run: terraform apply
#
# Import ID format:
#   anypoint_private_space_config -> <private_space_id>

# locals {
#   org_id = "<org_id>"   # e.g. "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
# }

# import {
#   provider = anypoint.admin
#   to = anypoint_organization.imported_org
#   id = local.org_id
# }
