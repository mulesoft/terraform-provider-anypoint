---
page_title: "anypoint_mcp_bridge Resource - terraform-provider-anypoint"
subcategory: "Agents Tools"
description: |-
  Manages an MCP bridge in Anypoint API Manager. An MCP bridge generates an MCP server from one or more source REST APIs by publishing a generated Exchange asset, creating the gateway instance, and attaching the MCP transcoding policies that map MCP tool calls to REST calls.
---

# anypoint_mcp_bridge (Resource)

Manages an **MCP bridge** in Anypoint API Manager — an MCP server built from REST APIs you already have.

Where [`anypoint_mcp_server`](anypoint_mcp_server.md) fronts an MCP server you supply as an Exchange asset, a bridge *generates* one. You declare one or more source REST APIs and, for each, the operations to expose as MCP tools. From that the provider:

1. publishes a generated Exchange asset describing the tools (`mcp-metadata.json`),
2. creates the gateway instance with one route per source API, matched on the `X-UPSTREAM-NAME` header, and its upstream backend,
3. attaches the MCP policies that translate each incoming tool call into the corresponding REST call.

The result is a single MCP endpoint that an agent can call, backed by any number of existing REST APIs.

-> **Gateway support.** A bridge runs on a Flex or Omni Gateway, managed or self-managed, in a private space or a shared space. The deployment metadata the platform needs is derived from `gateway_id`, so the same configuration works across all of them.

-> **Authentication.** This resource drives both **Exchange** (publishing the generated asset) and **Gateway Manager / API Manager** (the `gateway_id` pre-flight, instance create, policy attach). A `client_credentials` Connected App is sufficient, granted **Manage Servers**, **Read Servers**, **View Organization**, the API Manager scopes (**Manage APIs Configuration**, **Manage Policies**, **Deploy API Proxies**) and **Exchange Viewer** / **Exchange Contributor**. Missing scopes surface as `HTTP 401`/`403` during the gateway pre-flight, before anything is created. See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

-> **The gateway must be running.** The target gateway has to be connected and serving before apply. A disconnected gateway returns `GatewayNotReadyError`; the provider retries briefly, but a gateway that never comes up fails the create.

-> **Status after create.** `status` is read back from the instance and is often `unregistered` until the gateway picks up the deployment. That is expected and does not indicate a failure.

## Example Usage

### A single source REST API

```terraform
resource "anypoint_mcp_bridge" "petstore" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "petstore-mcp-bridge"
  port           = 8081
  base_path      = "petstore"

  source_apis = [
    {
      label        = "petstore-api"
      upstream_uri = "https://sandbox.example.com/petstore/v1"
      asset_id     = "petstore-rest-api"
      version      = "1.0.0"

      tools = [
        {
          method       = "GET"
          path         = "/pets"
          description  = "List all pets."
          query_params = ["limit", "offset"]
        },
        {
          method   = "POST"
          path     = "/pets"
          has_body = true
        },
        {
          method = "GET"
          path   = "/pets/{petId}"
        },
      ]
    },
  ]
}
```

### Several source APIs behind one MCP endpoint

Each source API becomes its own route and upstream. Tool names are unique across the whole bridge, so a client sees one flat tool list.

```terraform
resource "anypoint_mcp_bridge" "commerce" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "commerce-mcp-bridge"
  base_path      = "commerce"

  source_apis = [
    {
      label        = "orders-api"
      upstream_uri = "https://orders.internal:8080"
      asset_id     = "orders-rest-api"
      version      = "1.0.0"

      tools = [
        { method = "GET", path = "/orders/{orderId}" },
        { method = "POST", path = "/orders", has_body = true },
      ]
    },
    {
      label        = "inventory-api"
      upstream_uri = "https://inventory.internal:8080"
      asset_id     = "inventory-rest-api"
      version      = "2.0.0"

      # The gateway must trust this backend's certificate, so point the upstream at a
      # TLS context from Secrets Manager. Format: "<secretGroupId>/<tlsContextId>".
      tls_context_id = "${anypoint_secret_group.internal.id}/${anypoint_secret_group_tls_context.internal.id}"

      tools = [
        {
          method        = "GET"
          path          = "/inventory/{sku}"
          name          = "check_stock"
          description   = "Check available stock for a SKU."
          header_params = ["X-Warehouse-Id"]
        },
      ]
    },
  ]
}
```

### Publishing the endpoint and controlling access

`consumer_endpoint` is the URL clients use to reach the bridge, and is what API Manager shows as the instance's consumer endpoint. It is not derived automatically, so set it from the gateway's ingress URL.

```terraform
resource "anypoint_mcp_bridge" "internal_tools" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = anypoint_managed_omni_gateway.shared.id

  mcp_asset_name = "internal-tools"

  instance_label    = "internal-tools (staging)"
  approval_method   = "manual"
  consumer_endpoint = "${anypoint_managed_omni_gateway.shared.ingress.public_url}/mcp"

  source_apis = [
    # ...
  ]
}
```

`approval_method = "manual"` holds client access requests for review instead of granting them automatically. It only restricts access when a policy enforcing client credentials is also applied to the bridge — see [`anypoint_api_policy_client_id_enforcement`](anypoint_api_policy_client_id_enforcement.md).

### Shaping a tool's contract and its upstream request

By default a tool's input schema is derived from its parameters, with every property typed `string`, and each parameter is passed straight through to the backend under its own name. Both can be replaced when you need something more precise.

```terraform
tools = [
  {
    method        = "GET"
    path          = "/search"
    name          = "search_pets"
    description   = "Search the catalogue."
    query_params  = ["q", "limit"]
    header_params = ["apiKey"]

    # Describe the inputs the model sees, with real types and guidance.
    input_schema = jsonencode({
      type = "object"
      properties = {
        q      = { type = "string", description = "Free-text search query." }
        limit  = { type = "integer", description = "Maximum results to return." }
        apiKey = { type = "string", description = "API key for the upstream." }
      }
      required = ["q", "apiKey"]
    })

    # Describe how those inputs become the upstream HTTP request.
    http_mapping = {
      query_params = [
        # the backend calls it "query"; the tool exposes "q"
        { key = "query", value = "#[vars.params['q']]" },
        { key = "size", value = "#[vars.params['limit']]" },
      ]
      headers = [
        { key = "X-Api-Key", value = "#[vars.params['apiKey']]" },
      ]
    }
  },
]
```

The two are independent: `input_schema` describes what the model sends *to the tool*, `http_mapping` describes how that becomes the *request to your backend*. API Manager splits them across the "Tool definition" and "HTTP mapping" tabs for the same reason.

## Applying changes

Tool edits — adding, removing or changing a tool, including its `description`, `input_schema` or `http_mapping` — are applied in place. The generated asset is republished and the transcoding policies re-synced as needed.

The instance-level fields (`instance_label`, `approval_method`, `consumer_endpoint`, `provider_id`) are also applied in place, and do not touch the generated asset.

Changing a source API's **structure** requires replacement: its `label`, `upstream_uri`, `tls_context_id`, `asset_id`, `group_id` or `version`, or adding and removing whole `source_apis` entries. These live on the bridge's routing and upstreams, which the platform only accepts at create time. Terraform reports this as a "requires replacement" error; use `terraform apply -replace` to recreate the bridge. `port` and `base_path` are structural for the same reason.

Destroying a bridge removes its instance and policies but deliberately leaves the generated Exchange asset published, so any client still referencing that asset version keeps working. Manage the asset with [`anypoint_exchange_asset`](anypoint_exchange_asset.md) if you want it removed too.

## Schema

### Required

- `environment_id` (String) The environment ID where the MCP bridge is created. Changing this forces replacement.
- `gateway_id` (String) The UUID of the gateway to deploy the bridge to, managed or self-managed. The `deployment` block is derived from it. Changing this forces replacement.
- `mcp_asset_name` (String) The MCP asset name (UI: "MCP asset name"). Becomes the generated Exchange asset name and, sanitized, its asset ID, and is the instance name shown on the MCP server summary. Changing this forces replacement.
- `source_apis` (Attributes List) One entry per source REST API. See [`source_apis`](#nestedatt--source_apis) below.

### Optional

- `organization_id` (String) The organization ID. Inferred from the provider credentials when omitted.
- `port` (Number) The listener port for the bridge on the gateway, `8081` by default. The proxy URI is `http://0.0.0.0:<port>/<base_path>`. Changing this forces replacement; omitting it on an existing bridge keeps the port that bridge already uses, so importing a bridge on a non-default port does not drag it back to `8081`.
- `base_path` (String) The base path of the bridge proxy URI, empty by default. A leading slash is not significant — `"/mcp"` and `"mcp"` describe the same bridge, and an absent base path and an empty one are equivalent. Changing this forces replacement.
- `instance_label` (String) A human-readable label for this instance (UI: "Instance label"). Worth setting when several instances share one asset, so they can be told apart in API Manager.
- `approval_method` (String) Set to `manual` to hold client access requests for review (UI: "Manual approval"). Omit for automatic approval.
- `consumer_endpoint` (String) The consumer-facing MCP endpoint URI (UI: "Consumer Endpoint") — the public URL clients use to reach the bridge, normally the gateway's ingress URL plus the bridge's base path. When omitted, whatever the platform holds is read back, so a bridge that was never given one reports `null`.
- `provider_id` (String) The client provider used to authenticate applications requesting access (UI: "Client provider"). Omit to use Anypoint's built-in provider, which is what API Manager shows as "Anypoint"; set it to the ID of an external client provider configured on the organization.

### Read-Only

- `id` (String) The numeric identifier of the MCP bridge instance, stored as a string.
- `asset_id` (String) The generated Exchange asset ID.
- `asset_version` (String) The generated Exchange asset version, starting at `1.0.0`. It advances whenever a tool edit republishes the asset, and instance-level edits leave it alone. It moves to the next **unused** patch version rather than always `+1`: destroying a bridge leaves its published asset versions behind, so a bridge recreated under the same name starts at `1.0.0` and skips past whatever its predecessor published.
- `product_version` (String) The product version.
- `status` (String) The current status of the MCP bridge.
- `technology` (String) The gateway technology backing the bridge.
- `deployment` (Attributes) Deployment target details, derived from `gateway_id`. See [`deployment`](#nestedatt--deployment) below.

<a id="nestedatt--source_apis"></a>

### Nested Schema for `source_apis`

Required:

- `label` (String) The source API label, used as the route label and the `X-UPSTREAM-NAME` header value. Must be unique within the bridge.
- `upstream_uri` (String) The backend base URI that tool calls are forwarded to.
- `asset_id` (String) The source REST API's Exchange asset ID.
- `version` (String) The source asset version.
- `tools` (Attributes List) The tools exposed for this source API. See [`source_apis.tools`](#nestedatt--source_apis--tools) below.

Optional:

- `group_id` (String) The source asset's group (organization) ID. Defaults to `organization_id`.
- `tls_context_id` (String) A TLS context for the outbound connection to this backend, as `secretGroupId/tlsContextId` — the same format [`anypoint_mcp_server`](anypoint_mcp_server.md) uses. Set it when `upstream_uri` is `https` and the gateway must trust the backend's certificate, or when the backend requires mutual TLS. Omit it for plain `http` backends.

<a id="nestedatt--source_apis--tools"></a>

### Nested Schema for `source_apis.tools`

Required:

- `method` (String) The HTTP method: `GET`, `POST`, `PUT`, `PATCH` or `DELETE`.
- `path` (String) The REST operation path, for example `/pets/{petId}`. Path parameters become required tool inputs automatically. Must start with `/`.

Optional:

- `name` (String) The tool name. Defaults to `<method>_<slug(path)>`, so `GET /pets/{petId}` becomes `get_pets_petid`.
- `description` (String) The description shown to MCP clients. Defaults to the tool name.
- `query_params` (List of String) Query parameter names exposed as tool inputs.
- `header_params` (List of String) Header parameter names exposed as tool inputs.
- `has_body` (Boolean) Whether the operation takes a request body, typically for `POST`, `PUT` and `PATCH`. Adds a required `body` input. Defaults to `false`.
- `input_schema` (String) A JSON Schema object describing the tool's inputs (UI: "Input schema"), used instead of the schema derived from the parameters above. Build it with `jsonencode({...})`.
- `http_mapping` (Attributes) How the tool's inputs become the upstream HTTP request (UI: "HTTP mapping"). See [`source_apis.tools.http_mapping`](#nestedatt--source_apis--tools--http_mapping) below.

Without `input_schema`, the schema is derived from the tool's parameters: path parameters are required, query and header parameters optional, and every property is typed `string`. Supply `input_schema` when a property needs a different type, a description, an enum, or nested structure. The value is published exactly as given rather than merged with the derived schema, so it must describe every input the tool accepts.

~> `input_schema` and `description` live only in the generated asset metadata, which is not read back from the gateway, so both are `null` after `terraform import`. Adding them back to your configuration is a **real change**, not a cosmetic one: the next plan shows the tools updating in place, and applying it republishes the generated asset at a new `asset_version`. That is expected — it is how the descriptions reach the asset — and the plan is clean from then on.

<a id="nestedatt--source_apis--tools--http_mapping"></a>

### Nested Schema for `source_apis.tools.http_mapping`

By default every declared parameter is passed upstream under its own name, as `#[vars.params['<name>']]`, and path placeholders are taken from `path`. Set `http_mapping` when the backend expects a different name, a computed value, or a particular body.

Each field is independent: omit one and that part stays derived, so changing only the body does not mean restating every parameter. An empty list is meaningful — it sends none of that kind of parameter, which is different from omitting the field.

Optional:

- `query_params` (Attributes List) Mapping for query string parameters.
- `uri_params` (Attributes List) Mapping for path placeholders such as `{petId}`.
- `headers` (Attributes List) Mapping for request headers.
- `body` (String) DataWeave expression producing the request body. Defaults to `#[vars.params.body]` when `has_body` is `true`. Override only when the body should come from a differently named input (for example `#[vars.params.payload]`). Set it to `""` to send no body.

Each entry of `query_params`, `uri_params` and `headers` takes:

- `key` (String, Required) The name sent upstream.
- `value` (String, Required) DataWeave expression producing the value.

<a id="nestedatt--deployment"></a>

### Nested Schema for `deployment` (Read-Only)

- `environment_id` (String) The deployment environment ID.
- `type` (String) The deployment type.
- `expected_status` (String) The expected deployment status.
- `overwrite` (Boolean) Whether an existing deployment is overwritten.
- `target_id` (String) The target gateway ID.
- `target_name` (String) The target gateway name.
- `gateway_version` (String) The gateway runtime version.

## Import

An MCP bridge is imported with its composite ID, `organization_id/environment_id/mcp_bridge_id`, where `mcp_bridge_id` is the numeric ID in the API Manager URL.

On import the provider rebuilds `source_apis` — labels, upstreams and tools — from the bridge's live routing and transcoding policies. A tool whose request mapping is the default one imports in the concise `query_params` / `header_params` form; one with a customised mapping imports with an explicit `http_mapping` block. Tool `description` and `input_schema` are not recoverable, as noted above.

```terraform
import {
  to = anypoint_mcp_bridge.imported
  id = "<organization_id>/<environment_id>/<mcp_bridge_id>"
}
```

Then generate the configuration:

```shell
terraform plan -generate-config-out=generated.tf
```

To adopt the Exchange asset the bridge generated, import it alongside with [`anypoint_exchange_asset`](anypoint_exchange_asset.md), using `group_id/asset_id/version`. Use the computed `asset_id` (the sanitized form of `mcp_asset_name`), not the raw display name:

```terraform
import {
  to = anypoint_exchange_asset.generated
  id = "<organization_id>/<asset_id>/<asset_version>"
}
```
