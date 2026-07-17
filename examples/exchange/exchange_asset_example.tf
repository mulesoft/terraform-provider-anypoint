# Spec-backed Exchange assets across the common API types, plus external
# (non-managed) instances. Each references a sample spec file shipped in this
# example directory.
#
# Notes demonstrated here:
#   * `classifier` is the user-facing value (e.g. `oas`). Exchange stores it
#     bundled as `fat-oas`; the provider normalizes it back on read so there is
#     no perpetual diff.
#   * `type = "graphql-api"` is normalized to `graphql` on read; either accepted.
#   * `instances` are managed authoritatively — instances removed from the list
#     are deleted from the api-metadata-service on the next apply.
#   * A `version` change forces replacement, which hard-deletes the old version
#     BEFORE publishing the new one. `create_before_destroy` (below) flips that
#     order so the new version is published first — safe because a version bump
#     is a distinct GAV. Do NOT rely on it for a same-version file/type change
#     (that re-publishes to the same version URL); bump `version` instead.

# REST API backed by an OpenAPI (OAS) spec, with two external instances.
resource "anypoint_exchange_asset" "rest_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-rest-api"
  version         = "1.0.0"
  name            = "TF Demo REST API"
  type            = "rest-api"

  classifier = "oas"
  file_path  = "${path.module}/test-assets/petstore.json"
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

  # Recommended backstop for version bumps: publish the new version BEFORE
  # hard-deleting the old one, so a failed publish can't leave you with neither.
  # Safe here because a `version` change is a distinct GAV that can coexist.
  lifecycle {
    create_before_destroy = true
  }
}

# GraphQL API backed by a schema file.
resource "anypoint_exchange_asset" "graphql_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-graphql-api"
  version         = "1.0.1"
  name            = "TF Demo GraphQL API"
  type            = "graphql-api"

  classifier  = "graphql"
  api_version = "v1"
  file_path   = "${path.module}/test-assets/schema.graphql"

  instances = [
    {
      name         = "Production"
      endpoint_uri = "https://graphql.example.com/query"
    },
  ]
}

# SOAP API backed by a WSDL.
resource "anypoint_exchange_asset" "soap_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-soap-api"
  version         = "1.0.0"
  name            = "TF Demo SOAP API"
  type            = "soap-api"

  classifier = "wsdl"
  file_path  = "${path.module}/test-assets/weather.wsdl"
  main_file  = "weather.wsdl"

  instances = [
    {
      name         = "Production"
      endpoint_uri = "https://soap.example.com/weather"
    },
  ]
}

# HTTP API (endpoint-only, no spec file) with multiple external instances.
resource "anypoint_exchange_asset" "http_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-http-api"
  version         = "1.0.0"
  name            = "TF Demo HTTP API"
  type            = "http-api"
  api_version     = "v1"

  instances = [
    {
      name         = "US-East Production"
      endpoint_uri = "https://us-east.example.com/api"
      is_public    = true
    },
    {
      name         = "EU-West Production"
      endpoint_uri = "https://eu-west.example.com/api"
    },
    {
      name         = "Staging"
      endpoint_uri = "https://staging.example.com/api"
    },
  ]
}

output "rest_api_id" {
  description = "Composite GAV id of the REST API asset"
  value       = anypoint_exchange_asset.rest_api.id
}

output "graphql_api_version_group" {
  description = "Version group of the GraphQL API asset"
  value       = anypoint_exchange_asset.graphql_api.version_group
}
