terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = "185150e267f64f37b75fef95a3d0a329"
  client_secret = "42b0021E775D4Af38C108A121004DA8e"
}

# Import test: http-api asset with api_version=v2
resource "anypoint_exchange_asset" "imported" {
  organization_id = "6c3c4eb3-f16b-47f0-b9b7-ee98353c9e04"
  group_id        = "6c3c4eb3-f16b-47f0-b9b7-ee98353c9e04"
  asset_id        = "import-httpapi-test"
  version         = "1.0.0"
  name            = "Import HTTP API Test"
  type            = "http-api"
  api_version     = "v2"
}
