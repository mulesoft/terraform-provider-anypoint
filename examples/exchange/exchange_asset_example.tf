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

# AsyncAPI (evented-api) backed by an AsyncAPI spec file.
# IMPORTANT: the classifier is `evented-api` (same as the type), NOT `asyncapi`.
# Using classifier = "asyncapi" fails with 400 COULD_NOT_DETERMINE_ASSET_TYPE.
# Like rest-api/grpc-api, evented-api REQUIRES api_version at create.
resource "anypoint_exchange_asset" "evented_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-evented-api"
  version         = "1.0.0"
  name            = "TF Demo AsyncAPI"
  type            = "evented-api"

  classifier  = "evented-api"
  api_version = "v1"
  file_path   = "${path.module}/asyncapi-sample.yaml"
  main_file   = "asyncapi-sample.yaml"
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

# ---------------------------------------------------------------------------
# policy — the MULTI-FILE case. A policy publishes TWO files in ONE request:
# the JSON schema as `file_path` (classifier "schema") and the metadata YAML via
# `additional_file` (classifier "metadata").
#
# The metadata YAML has three hard requirements, all enforced by the platform:
#   1. it must start with the header  #%Policy Definition 0.1
#   2. its `name:` must EXACTLY equal this resource's `name`
#   3. its `type:` must be one of the allowed values (e.g. "custom")
# Getting any of them wrong returns 400 INVALID_ASSET_METADATA naming the problem.
# ---------------------------------------------------------------------------
resource "anypoint_exchange_asset" "policy" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "example-custom-policy"
  version         = "1.0.0"
  name            = "Example Custom Policy" # must match `name:` in metadata.yaml
  type            = "policy"

  classifier = "schema"
  file_path  = "${path.module}/test-assets/policy-schema.json"

  additional_file = [
    {
      classifier = "metadata"
      path       = "${path.module}/test-assets/policy-metadata.yaml"
    },
  ]
}

# ---------------------------------------------------------------------------
# connector / app / template / example — the JAR-backed mule types.
# The uploaded .jar MUST contain META-INF/mule-artifact/mule-artifact.json
# (and classloader-model.json for type = "example"), otherwise the publish fails
# with 400 INVALID_ASSET_METADATA "Could not find mule-artifact file inside jar file".
#
#   type = "connector"  -> classifier = "mule-plugin"               (stored as "extension")
#   type = "app"        -> classifier = "mule-application"
#   type = "template"   -> classifier = "mule-application-template"
#   type = "example"    -> classifier = "mule-application-example"
# ---------------------------------------------------------------------------
# No .jar is shipped with these examples — point this at the artifact your Mule
# build produces (target/<your-connector>-mule-plugin.jar).
resource "anypoint_exchange_asset" "connector" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "example-connector"
  version         = "1.0.0"
  name            = "Example Connector"
  type            = "connector"
  classifier      = "mule-plugin"
  file_path       = var.connector_jar_path
}

# ---------------------------------------------------------------------------
# ruleset — API Governance ruleset. classifier == type, uploaded as .yaml.
# ---------------------------------------------------------------------------
resource "anypoint_exchange_asset" "ruleset" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "example-ruleset"
  version         = "1.0.0"
  name            = "Example Governance Ruleset"
  type            = "ruleset"
  classifier      = "ruleset"
  file_path       = "${path.module}/test-assets/governance.yaml"
}

# ---------------------------------------------------------------------------
# grpc-api — the classifier is "protobuf". Upload a bare .proto, or a .zip with
# at least one .proto in its root directory. api_version is required at create.
# ---------------------------------------------------------------------------
resource "anypoint_exchange_asset" "grpc_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "example-grpc-api"
  version         = "1.0.0"
  name            = "Example gRPC API"
  type            = "grpc-api"
  classifier      = "protobuf"
  file_path       = "${path.module}/test-assets/petstore.proto"
  main_file       = "petstore.proto"
  api_version     = "v1"
}
