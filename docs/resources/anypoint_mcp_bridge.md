---
page_title: "anypoint_mcp_bridge Resource - terraform-provider-anypoint"
subcategory: "Agents Tools"
description: |-
  Manages an MCP bridge in Anypoint API Manager. An MCP bridge generates an MCP server from one or more source REST APIs by publishing a generated Exchange asset, creating the Flex/Omni Gateway instance, and attaching the MCP transcoding policies that map MCP tool calls to REST calls.
---

# anypoint_mcp_bridge (Resource)

Manages an **MCP bridge** in Anypoint API Manager. Unlike [`anypoint_mcp_server`](anypoint_mcp_server.md) — where you supply an existing MCP server spec asset — a bridge **generates** its MCP server from one or more source REST APIs. For each source API you declare the tools (REST operations) to expose, and the provider:

1. generates and publishes an Exchange asset (`mcp-metadata.json`) describing the tools,
2. creates the Flex / Self-Managed Gateway instance with **one route per source API** (matched by the `X-UPSTREAM-NAME` header) and its upstream backend,
3. attaches the MCP transcoding policies that map each MCP tool call to the underlying REST call.

-> **Authentication:** This resource orchestrates **Exchange** (asset publish) and **Gateway Manager / API Manager** control-plane APIs (the `gateway_id` pre-flight, instance create, policy attach). A `client_credentials` Connected App works — grant it **Manage Servers**, **Read Servers**, **View Organization**, the API Manager scopes (**Manage APIs Configuration**, **Manage Policies**, **Deploy API Proxies**), and **Exchange Viewer**/**Exchange Contributor**. A Connected App missing these scopes is rejected with `HTTP 401`/`403` during the gateway pre-flight, before anything is created; the fix is to add the scopes (or use `auth_type = "user"` with a user that has the equivalent permissions). See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

-> **Gateway must be running:** The target gateway must be connected and running before apply. If the gateway is not ready the platform returns `GatewayNotReadyError`; the provider retries briefly, but a persistently disconnected gateway will fail the create.

-> **Tools are declared explicitly (v1):** Each tool is a single REST operation. Path parameters (`/pets/{petId}`) automatically become required tool inputs. Query and header parameters are exposed with `query_params` / `header_params`, and request bodies with `has_body = true`.

-> **Updates:** Adding, removing, or editing **tools** (including a tool's `description`) is an in-place update: the generated asset version is bumped (e.g. `1.0.0` → `1.0.1`, reflected in the computed `asset_version`) and the transcoding policies are re-synced. Changing a source API's **structure** — its `label`, `upstream_uri`, `asset_id`, `group_id`, `version`, or adding/removing whole `source_apis` — is rejected with a "requires replacement" error; recreate with `terraform apply -replace` to apply structural changes.

-> **Status:** After create, `status` is read back from the instance and is typically `unregistered` until the connected gateway begins serving the deployment. This is expected and does not indicate a failure.

## Example Usage

### Single source REST API

```terraform
resource "anypoint_mcp_bridge" "petstore" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "petstore-mcp-bridge"
  port        = 8081
  base_path   = "petstore"

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
          description  = "List all pets"
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

### Multiple source APIs

```terraform
resource "anypoint_mcp_bridge" "commerce" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "commerce-mcp-bridge"
  base_path   = "commerce"

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
      tools = [
        {
          method        = "GET"
          path          = "/inventory/{sku}"
          name          = "check_stock"
          description   = "Check available stock for a SKU"
          header_params = ["X-Warehouse-Id"]
        },
      ]
    },
  ]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID where the MCP bridge is created. Changing this forces replacement.
- `gateway_id` (String) The Flex / Self-Managed (Omni) Gateway UUID to deploy the bridge to. The `deployment` block is auto-populated by fetching gateway details. Changing this forces replacement.
- `mcp_asset_name` (String) The MCP asset name (UI: "MCP asset name"). Becomes the generated Exchange asset name and (sanitized) asset ID, and is shown as the instance name on the MCP server summary. Changing this forces replacement.
- `source_apis` (Attributes List) One entry per source REST API. See [`source_apis`](#nestedatt--source_apis) below.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID is inferred from the connected app credentials.
- `port` (Number) The listener port for the bridge on the gateway. Defaults to `8081`. The proxy URI is `http://0.0.0.0:<port>/<base_path>`. Changing this forces replacement.
- `base_path` (String) The base path for the bridge proxy URI (default empty). The proxy URI is `http://0.0.0.0:<port>/<base_path>`. Changing this forces replacement.

### Read-Only

- `id` (String) The numeric identifier of the MCP bridge instance (stored as string).
- `asset_id` (String) The generated Exchange asset ID.
- `asset_version` (String) The generated Exchange asset version (starts at `1.0.0` and bumps on tool updates).
- `product_version` (String) The product version.
- `consumer_endpoint` (String) The consumer-facing MCP endpoint URI (UI: "Consumer Endpoint").
- `status` (String) The current status of the MCP bridge.
- `technology` (String) The gateway technology (always `flexGateway` for a bridge).
- `deployment` (Attributes) Deployment target details, auto-populated from `gateway_id`. See [`deployment`](#nestedatt--deployment) below.

<a id="nestedatt--source_apis"></a>
### Nested Schema for `source_apis`

Required:

- `label` (String) The source API label; used as the route label and the `X-UPSTREAM-NAME` header value. Must be unique within the bridge.
- `upstream_uri` (String) The real backend base URI that tool calls are forwarded to.
- `asset_id` (String) The source REST API's Exchange asset ID (the connection link).
- `version` (String) The source asset version.
- `tools` (Attributes List) The explicit tools exposed for this source API. See [`source_apis.tools`](#nestedatt--source_apis--tools) below.

Optional:

- `group_id` (String) The source asset group (organization) ID. Defaults to `organization_id`.

<a id="nestedatt--source_apis--tools"></a>
### Nested Schema for `source_apis.tools`

Required:

- `method` (String) The HTTP method. Valid values: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`.
- `path` (String) The REST operation path, e.g. `/pets/{petId}`. Path parameters (`{...}`) become required tool inputs automatically. Must start with `/`.

Optional:

- `name` (String) The tool name. Defaults to `<method>_<slug(path)>` (e.g. `GET /pets/{petId}` → `get_pets_petid`).
- `description` (String) The tool description shown to MCP clients. Defaults to the tool name.
- `query_params` (List of String) Query parameter names exposed as tool inputs.
- `header_params` (List of String) Header parameter names exposed as tool inputs.
- `has_body` (Boolean) Whether the operation takes a request body (typically for `POST`/`PUT`/`PATCH`). Adds a required `body` tool input. Defaults to `false`.

<a id="nestedatt--deployment"></a>
### Nested Schema for `deployment` (Read-Only)

- `environment_id` (String) The deployment environment ID.
- `type` (String) The deployment type (`HY`).
- `expected_status` (String) The expected deployment status.
- `overwrite` (Boolean) Whether an existing deployment is overwritten.
- `target_id` (String) The target gateway ID.
- `target_name` (String) The target gateway name.
- `gateway_version` (String) The gateway runtime version.

## Import

An existing MCP bridge can be imported using its composite ID: `organization_id/environment_id/mcp_bridge_id`.

The `mcp_bridge_id` is the numeric ID visible in the Anypoint API Manager URL (e.g. `21058094`).

On import, the provider reconstructs `source_apis` (labels, upstreams, and tools) from the live routing and transcoding policies. Tool `description` values live only in the generated asset metadata and are left null after import — add them back to your config as desired (they will not cause a spurious diff).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_mcp_bridge.imported
  id = "<organization_id>/<environment_id>/<mcp_bridge_id>"
}

resource "anypoint_mcp_bridge" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  gateway_id      = "<gateway_id>"
  mcp_asset_name  = "<mcp_asset_name>"

  source_apis = [
    {
      label        = "<source_api_label>"
      upstream_uri = "https://<backend-host>"
      asset_id     = "<source_rest_api_asset_id>"
      version      = "1.0.0"
      tools = [
        { method = "GET", path = "/example/{id}" },
      ]
    },
  ]
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

### Using the CLI (deprecated, Terraform < 1.5)

```shell
terraform import anypoint_mcp_bridge.imported <organization_id>/<environment_id>/<mcp_bridge_id>
```
