terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Basic Exchange asset: a metadata-only "custom" asset (no file upload).
# See exchange_asset_example.tf for spec-backed assets (REST/GraphQL/SOAP/AsyncAPI)
# and external instances.
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
