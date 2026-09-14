###############################################################################
# Other Exchange asset types (not REST, not custom, not multi-version).
# Apply this folder to publish GraphQL, SOAP, AsyncAPI, HTTP, gRPC, policy,
# and ruleset. Connector stays commented — no mule-plugin JAR ships here.
###############################################################################

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

# GraphQL API backed by a schema file. `type = "graphql-api"` is normalized to
# `graphql` on read; either is accepted.
resource "anypoint_exchange_asset" "graphql_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-graphql-api"
  version         = "1.0.0"
  name            = "TF Demo GraphQL API"
  type            = "graphql-api"

  classifier  = "graphql"
  api_version = "v1"
  file_path   = "${path.module}/../test-assets/schema.graphql"

  instances = [
    {
      name         = "Production"
      endpoint_uri = "https://graphql.example.com/query"
    },
  ]
}

# SOAP API backed by a WSDL. api_version is required at create.
resource "anypoint_exchange_asset" "soap_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-soap-api"
  version         = "1.0.0"
  name            = "TF Demo SOAP API"
  type            = "soap-api"
  api_version     = "v1"

  classifier = "wsdl"
  file_path  = "${path.module}/../test-assets/weather.wsdl"
  main_file  = "weather.wsdl"

  instances = [
    {
      name         = "Production"
      endpoint_uri = "https://soap.example.com/weather"
    },
  ]
}

# classifier must be evented-api (not asyncapi) or Exchange returns
# 400 COULD_NOT_DETERMINE_ASSET_TYPE. api_version is required at create.
resource "anypoint_exchange_asset" "evented_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-evented-api"
  version         = "1.0.0"
  name            = "TF Demo AsyncAPI"
  type            = "evented-api"

  classifier  = "evented-api"
  api_version = "v1"
  file_path   = "${path.module}/../asyncapi-sample.yaml"
  main_file   = "asyncapi-sample.yaml"
}

# Endpoint-only HTTP API (no spec file). api_version is required at create.
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

# classifier is "protobuf". Upload a bare .proto, or a .zip with at least one
# .proto in its root. api_version is required at create.
resource "anypoint_exchange_asset" "grpc_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "example-grpc-api"
  version         = "1.0.0"
  name            = "Example gRPC API"
  type            = "grpc-api"
  classifier      = "protobuf"
  file_path       = "${path.module}/../test-assets/petstore.proto"
  main_file       = "petstore.proto"
  api_version     = "v1"
}

# Policy publishes TWO files: JSON schema (file_path) + metadata YAML
# (additional_file). Metadata YAML must start with #%Policy Definition 0.1
# and name: must match this resource name.
resource "anypoint_exchange_asset" "policy" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "example-custom-policy"
  version         = "1.0.0"
  name            = "Example Custom Policy" # must match `name:` in policy-metadata.yaml
  type            = "policy"

  classifier = "schema"
  file_path  = "${path.module}/../test-assets/policy-schema.json"

  additional_file = [
    {
      classifier = "metadata"
      path       = "${path.module}/../test-assets/policy-metadata.yaml"
    },
  ]
}

# main_file is required at create for ruleset (400 MISSING_REQUIRED_PROPERTIES otherwise).
resource "anypoint_exchange_asset" "ruleset" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "example-ruleset"
  version         = "1.0.0"
  name            = "Example Governance Ruleset"
  type            = "ruleset"
  classifier      = "ruleset"
  file_path       = "${path.module}/../test-assets/governance.yaml"
  main_file       = "governance.yaml"
}

# Commented so terraform plan succeeds without a JAR. Uncomment after pointing
# file_path at a mule-plugin JAR that contains
# META-INF/mule-artifact/mule-artifact.json. Exchange stores connectors as type
# `extension`; `type = "connector"` is accepted and normalized.
#
# resource "anypoint_exchange_asset" "connector" {
#   organization_id = var.org_id
#   group_id        = var.org_id
#   asset_id        = "example-connector"
#   version         = "1.0.0"
#   name            = "Example Connector"
#   type            = "connector"
#   classifier      = "mule-plugin"
#   file_path       = var.connector_jar_path
# }

output "graphql_api_id" {
  description = "Composite GAV id of the GraphQL API asset"
  value       = anypoint_exchange_asset.graphql_api.id
}

output "graphql_api_version_group" {
  description = "Version group of the GraphQL API asset"
  value       = anypoint_exchange_asset.graphql_api.version_group
}

output "soap_api_id" {
  value = anypoint_exchange_asset.soap_api.id
}

output "evented_api_id" {
  value = anypoint_exchange_asset.evented_api.id
}

output "http_api_id" {
  value = anypoint_exchange_asset.http_api.id
}

output "grpc_api_id" {
  value = anypoint_exchange_asset.grpc_api.id
}

output "policy_id" {
  value = anypoint_exchange_asset.policy.id
}

output "ruleset_id" {
  value = anypoint_exchange_asset.ruleset.id
}
