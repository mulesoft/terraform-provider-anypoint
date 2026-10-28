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

# Metadata-only custom asset (no file upload).
resource "anypoint_exchange_asset" "custom" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-custom"
  version         = "1.0.0"
  name            = "TF Demo Custom Asset"
  type            = "custom"
  description     = "A custom Exchange asset published by Terraform."
  keywords        = "terraform,demo,custom"
}

output "custom_id" {
  description = "Composite GAV id of the custom asset"
  value       = anypoint_exchange_asset.custom.id
}
