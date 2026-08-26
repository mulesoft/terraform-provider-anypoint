# Anypoint Exchange Asset Examples

This directory contains examples for publishing and managing **Exchange assets**
with the Terraform provider.

## Resources & Data Sources Demonstrated

- `anypoint_exchange_asset` (resource) — publish/update/delete an Exchange asset
  version. Supports metadata-only assets, spec-backed assets (RAML/OAS/WSDL/
  GraphQL/AsyncAPI), documentation pages, categories, custom fields, and external
  (non-managed) API instances.
- `anypoint_exchange_asset` (data source) — read a single asset version by GAV.
- `anypoint_exchange_assets` (data source) — list assets in an org, filtered by
  type and free-text search.

## Files

- `main.tf` — provider config + a basic metadata-only `custom` asset.
- `exchange_asset_example.tf` — spec-backed REST / GraphQL / SOAP assets and an
  endpoint-only HTTP API, each with external instances.
- `datasource.tf` — single-asset and list data source usage with outputs.
- `variables.tf` — variable definitions (placeholder defaults; NO real creds).
- `import.tf` — how to import an existing asset (commented out).
- `test-assets/` — sample spec files (`petstore.json`, `schema.graphql`,
  `weather.wsdl`, `events.json`) referenced by the examples.

## Usage

```shell
cp terraform.tfvars.example terraform.tfvars   # then fill in real values
terraform init
terraform plan
terraform apply
```

## Behavior worth knowing

- **Immutable versions.** `organization_id`, `group_id`, `asset_id`, and
  `version` form the GAV identity; changing any of them (or `type`) replaces the
  asset. Publish a new `version` instead of mutating a published one.
- **Classifier normalization.** Set the user-facing `classifier` (e.g. `oas`).
  Exchange generates several derived copies of every upload and prefixes them —
  `fat-` (bundled), `light-` (trimmed) and `original-` (verbatim). The provider
  strips all three on read, so there is no perpetual plan diff and no spurious
  forced replacement after `terraform import`.
- **Classifier is the FILE kind, not the type**, and the FILE EXTENSION matters too
  (the upload is sent as `files.{classifier}.{ext}` and Exchange validates `{ext}`).

  | `type` | `classifier` | file ext | `api_version` at create? |
  |---|---|---|---|
  | `rest-api` | `oas` or `raml` | `.json` / `.raml` | **yes** |
  | `soap-api` | `wsdl` | `.wsdl` | **yes** |
  | `graphql-api` | `graphql` | `.graphql` | no |
  | `evented-api` | `evented-api` | **`.yaml` or `.zip` only** | **yes** |
  | `ruleset` | `ruleset` | `.yaml` | no |
  | `custom` | `custom` | any (file optional) | no |
  | `http-api` / `mcp` / `llm` | — (no file) | — | `http-api` **yes**, others no |
  | `app` | `mule-application` | `.jar` | no |
  | `template` | `mule-application-template` | `.jar` | no |
  | `example` | `mule-application-example` | `.jar` | no |
  | `connector` | `mule-plugin` | `.jar` | no |
  | `policy` | `schema` + `metadata` (**two files**) | `.json` + `.yaml` | no |
  | `raml-fragment` | `raml-fragment` | `.raml` | no |
  | `agent` | `a2a-card` | `.json` | no |
  | `grpc-api` | `protobuf` | `.proto` or `.zip` | **yes** |

  Two that trip people up: **AsyncAPI** uses `classifier = "evented-api"` (the same
  string as the type) — `asyncapi` is rejected with
  `400 COULD_NOT_DETERMINE_ASSET_TYPE` — and it only accepts `.yaml` or `.zip`, so a
  `.json` AsyncAPI spec fails with `400 MISSING_FILES_ERROR`.

- **JAR-backed types need a real Mule descriptor inside the jar.** `app`, `template`,
  `example` and `connector` fail with `400 INVALID_ASSET_METADATA: "Could not find
  mule-artifact file inside jar file"` unless the jar contains
  `META-INF/mule-artifact/mule-artifact.json` (plus `classloader-model.json` for
  `example`). Use the artifact your Mule build produces.
- **`policy` is the multi-file case.** Publish the JSON schema as `file_path`
  (`classifier = "schema"`) and the metadata YAML via `additional_file`
  (`classifier = "metadata"`). The YAML must start with `#%Policy Definition 0.1`,
  its `name:` must exactly equal the resource's `name`, and its `type:` must be an
  allowed value such as `custom`. See `exchange_asset_example.tf`.
- **mule-plugin family uses `type = "extension"`.** Exchange stores the whole
  mule-plugin family (policies and connectors, `classifier = "mule-plugin"`)
  under the generic `extension` super-type. Declare `type = "extension"` — it is
  the canonical, round-trip-stable value (an imported asset reads back as
  `extension`). The provider also accepts `policy` and `connector` as aliases and
  normalizes them so there is no post-apply `type` drift, but `extension` is
  recommended.
- **External instances are authoritative.** Instances removed from the `instances`
  list are deleted from Anypoint's api-metadata-service on the next apply, and are
  removed on `terraform destroy` before the asset version is hard-deleted — so no
  orphaned instances remain to block a later recreate at the same version group.
- **Hard delete, not a soft tombstone.** Delete uses a hard delete
  (`x-delete-type: hard-delete`) rather than a soft-delete tombstone. To publish
  new content, **bump `version`** — republishing onto a `group/asset/version`
  that still exists is rejected with `409 ASSET_PRE_CONDITIONS_FAILED`, which the
  provider surfaces at plan time (before any destroy) and again at apply.
