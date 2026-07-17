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
  Exchange stores it bundled as `fat-oas`; the provider strips the prefix on read
  so there is no perpetual plan diff.
- **External instances are authoritative.** Instances removed from the `instances`
  list are deleted from Anypoint's api-metadata-service on the next apply, and are
  removed on `terraform destroy` before the asset version is hard-deleted — so no
  orphaned instances remain to block a later recreate at the same version group.
- **Hard delete.** Deleting an asset frees its GAV coordinates (hard delete), so
  the same `asset_id`/`version` can be published again without a `409` tombstone.
