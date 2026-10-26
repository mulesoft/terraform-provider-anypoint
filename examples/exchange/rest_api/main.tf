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

# REST API backed by an OpenAPI (OAS) spec.
# api_version is required at create for rest-api (400 MISSING_REQUIRED_PROPERTIES otherwise).
resource "anypoint_exchange_asset" "rest_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-rest-api"
  version         = "1.0.0"
  name            = "TF Demo REST API"
  type            = "rest-api"
  api_version     = "v1"

  classifier = "oas"
  file_path  = "${path.module}/../test-assets/petstore.json"
  main_file  = "petstore.json"

  instances = [
    {
      name         = "Production"
      endpoint_uri = "https://api.example.com/petstore"
      is_public    = true
    },
    {
      name         = "Sandbox"
      endpoint_uri = "https://sandbox.example.com/petstore"
    },
  ]

  lifecycle {
    create_before_destroy = true
  }
}

output "rest_api_id" {
  description = "Composite GAV id of the REST API asset"
  value       = anypoint_exchange_asset.rest_api.id
}
