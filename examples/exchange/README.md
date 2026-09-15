# Anypoint Exchange asset examples

Each subdirectory is its own Terraform root. Apply **one folder at a time**.

| Folder | What it shows |
|--------|----------------|
| [custom/](./custom/) | Metadata-only `custom` asset (no file) |
| [rest_api/](./rest_api/) | OAS REST API (`api_version` required at create) |
| [types/](./types/) | GraphQL, SOAP, AsyncAPI, HTTP, gRPC, policy, ruleset |
| [multi_version/](./multi_version/) | Several versions of one REST API via `for_each` |
| [datasource/](./datasource/) | List REST APIs; singular read is commented until `rest_api/` exists |
| [import/](./import/) | Commented import template |

`custom/` is separate from `types/` so a first apply can publish one metadata-only asset without also creating every spec-backed type.

Shared specs live in [test-assets/](./test-assets/). The AsyncAPI spec is [asyncapi-sample.yaml](./asyncapi-sample.yaml).

```bash
cd rest_api   # or custom, types, multi_version, datasource, import
cp terraform.tfvars.example terraform.tfvars   # fill in values
terraform init && terraform plan && terraform apply
```

Connected-app **client credentials** is the right grant for Exchange (`edit:exchange` / `view:exchange`).

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
  | `ruleset` | `ruleset` | `.yaml` | no; **`main_file` yes** |
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
  `example`). Use the artifact your Mule build produces. The connector block in
  [types/](./types/) is commented for that reason.
- **`policy` is the multi-file case.** Publish the JSON schema as `file_path`
  (`classifier = "schema"`) and the metadata YAML via `additional_file`
  (`classifier = "metadata"`). The YAML must start with `#%Policy Definition 0.1`,
  its `name:` must exactly equal the resource's `name`, and its `type:` must be an
  allowed value such as `custom`.
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
