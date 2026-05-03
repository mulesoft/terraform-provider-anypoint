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

locals {
  private_space_id = "675c4efb-d44e-44cd-ac6f-d5a1128e6236"   # e.g. "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
}

import {
  to = anypoint_private_space_config.imported
  id = local.private_space_id
}
