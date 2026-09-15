---
page_title: "anypoint_api_instance Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages an API instance in Anypoint API Manager. An API instance represents an API specification deployed to a Omni Gateway target with routing rules and upstream backends.
---

# anypoint_api_instance (Resource)

Manages an API instance in Anypoint API Manager. An API instance represents an API specification deployed to a Omni Gateway target with routing rules and upstream backends.

-> **Gateway type:** The provider supports both **MuleSoft Managed Omni Gateway** (CloudHub 2.0, via [`anypoint_managed_omni_gateway`](anypoint_managed_omni_gateway.md)) and **self-managed (connected-mode) Flex/Omni Gateway** that you run on your own infrastructure (via [`anypoint_self_managed_gateway`](anypoint_self_managed_gateway.md)).

-> **Authentication:** This resource calls **Gateway Manager / API Manager control-plane APIs** (the gateway lifecycle and/or the `gateway_id` pre-flight). A `client_credentials` Connected App works — grant it **Manage Servers**, **Read Servers**, and **View Organization** (plus API Manager scopes such as **Manage APIs Configuration**, **Manage Policies**, **Deploy API Proxies**, and **Exchange Viewer** for Omni Gateway operations). A Connected App missing these scopes is rejected with `HTTP 401`/`403` before anything is created; the fix is to add the scopes (or use `auth_type = "user"` with a user that has the equivalent permissions). See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

## Example Usage

~> **Deprecation notice:** the top-level `upstream_uri` field is deprecated and will be removed in the next major version. New configurations should use the `routing` block, which supports TLS, labels, multiple upstreams, and weighted routing. See the migration note below.

### Minimal configuration

```terraform
resource "anypoint_api_instance" "minimal" {
  environment_id = var.environment_id
  gateway_id     = var.gateway_id
  instance_label = "minimal-demo"

  spec = {
    asset_id = "my-api"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    base_path = "minimal"
  }

  routing = [
    {
      upstreams = [
        { uri = "http://backend.internal:8080" }
      ]
    }
  ]
}
```

### Migrating from `upstream_uri`

If you are using the deprecated `upstream_uri` field, replace it with an equivalent single-upstream `routing` block:

```terraform
# Before (deprecated)
resource "anypoint_api_instance" "example" {
  # ...
  upstream_uri = "http://backend.internal:8080"
}

# After
resource "anypoint_api_instance" "example" {
  # ...
  routing = [
    {
      upstreams = [
        { uri = "http://backend.internal:8080" }
      ]
    }
  ]
}
```

The two forms are semantically identical — `upstream_uri` was always shorthand for `routing = [{ upstreams = [{ weight = 100, uri = <value> }] }]`.

### Weighted multi-upstream routing (canary / blue-green)

```terraform
resource "anypoint_api_instance" "weighted_routing" {
  environment_id = var.environment_id
  gateway_id     = var.gateway_id
  instance_label = "weighted-routing-demo"

  spec = {
    asset_id = "my-api"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    base_path = "weightedRouting"
  }

  routing = [
    {
      label = "canary"
      upstreams = [
        {
          weight = 90
          uri    = "http://backend-stable.internal:8080"
          label  = "stable"
        },
        {
          weight = 10
          uri    = "http://backend-canary.internal:8080"
          label  = "canary"
        }
      ]
    }
  ]
}
```

## How routing & upstreams sync to the backend

The API Manager backend models routing and upstreams as **two separate concepts**, but this Terraform resource exposes a single, denormalized `routing` block that nests upstreams inside each route. Understanding the mapping is important when you add, remove, or change routes — especially because the backend's PATCH semantics are stricter than its POST semantics.

### Backend model

For an API instance the API Manager keeps:

- **Upstreams catalog** — a flat list of upstream backends, each with a server-assigned `id`, plus `label`, `uri`, and optional `tlsContext`.
- **Routing** — an ordered list of routes, where each route's `upstreams` array references the catalog **by `id` and `weight`**. The reference does **not** carry `uri`, `label`, or `tls`.

Example backend payload:

```json
{
  "upstreams": [
    { "id": "abc-123", "label": "stable", "uri": "https://stable.internal:8081" },
    { "id": "def-456", "label": "canary", "uri": "https://canary.internal:8081" }
  ],
  "routing": [
    { "label": "main", "upstreams": [{ "id": "abc-123", "weight": 90 }, { "id": "def-456", "weight": 10 }] }
  ]
}
```

### Provider model

In Terraform the upstream definition is written **inline** inside `routing[].upstreams[]`:

```hcl
routing = [
  {
    label = "main"
    upstreams = [
      { weight = 90, uri = "https://stable.internal:8081", label = "stable" },
      { weight = 10, uri = "https://canary.internal:8081", label = "canary" },
    ]
  }
]
```

The provider takes care of translating between the two shapes. You never manage upstream `id`s yourself — they live in state.

### Identity matching

When the provider needs to map a plan upstream to a backend upstream (during Update), it matches in this order:

1. **`label`** — if the plan upstream has a label, the provider looks for a backend upstream with the same label.
2. **`uri`** — falls back to URI match if there's no label match.
3. If neither matches, the upstream is treated as **new**.

> **Recommendation:** always set `label` on every upstream. Labels are stable across URI changes and across reordering. Without a label, changing a URI looks like "delete + add" instead of "edit".

### What happens on `terraform apply`

#### Create (`POST /apis`)

A single API call. Inline upstreams in `routing[].upstreams[]` are accepted by the create endpoint, so the provider sends the routing block as-is. The backend creates the catalog entries and the routing references in one shot.

#### Update (`PATCH /apis/{id}`) — stricter

The PATCH endpoint **rejects** inline upstreams in `routing[].upstreams[]` — it requires every routing reference to be a strict `{ id, weight }` against an upstream that already exists in the catalog. To honor that constraint, the provider runs Update as up to two PATCHes:

1. **Pre-PATCH (only if the plan introduces brand-new upstreams)**
   - Body: `routing` = unchanged (the server's current routing), `upstreams` = existing catalog (id-only) + new entries with `{ label, uri, tls }` and no `id`.
   - The backend assigns ids to the new entries. Existing routing keeps validating because nothing it references has changed.
2. **Re-list** (`GET /upstreams`) to capture the newly assigned ids.
3. **Main PATCH**
   - Body: `routing` = the new routing built from the plan with `{ id, weight }` refs only; `upstreams` = full catalog with all ids resolved, so any `uri`/`label`/`tls` change on an existing upstream propagates in this same call.

When the plan only changes weights, rules, or fields on existing upstreams (no genuinely new upstream), the pre-PATCH is skipped and Update is a single PATCH.

### Worked example: adding a new route to an existing instance

Starting state — one route, one upstream:

```hcl
resource "anypoint_api_instance" "demo" {
  # ...
  routing = [
    {
      label = "stable"
      rules = { methods = "GET", path = "/stable/.*" }
      upstreams = [
        { weight = 100, uri = "https://stable.internal:8081", label = "stable" }
      ]
    }
  ]
}
```

Backend after Create:

```json
{
  "upstreams": [{ "id": "abc-123", "label": "stable", "uri": "https://stable.internal:8081" }],
  "routing": [{ "label": "stable", "upstreams": [{ "id": "abc-123", "weight": 100 }] }]
}
```

Now add a canary route with a brand-new upstream:

```hcl
resource "anypoint_api_instance" "demo" {
  # ...
  routing = [
    {
      label = "stable"
      rules = { methods = "GET", path = "/stable/.*" }
      upstreams = [
        { weight = 100, uri = "https://stable.internal:8081", label = "stable" }
      ]
    },
    {
      label = "canary"
      rules = { methods = "GET", path = "/canary/.*" }
      upstreams = [
        { weight = 100, uri = "https://canary.internal:8081", label = "canary" }
      ]
    },
  ]
}
```

What the provider sends:

**Pre-PATCH (registers the new `canary` upstream):**

```json
{
  "routing": [{ "label": "stable", "upstreams": [{ "id": "abc-123", "weight": 100 }] }],
  "upstreams": [
    { "id": "abc-123" },
    { "label": "canary", "uri": "https://canary.internal:8081" }
  ]
}
```

The backend assigns `def-456` to the canary entry.

**Main PATCH (uses the new id in routing):**

```json
{
  "routing": [
    { "label": "stable", "rules": {"methods":"GET","path":"/stable/.*"}, "upstreams": [{ "id": "abc-123", "weight": 100 }] },
    { "label": "canary", "rules": {"methods":"GET","path":"/canary/.*"}, "upstreams": [{ "id": "def-456", "weight": 100 }] }
  ],
  "upstreams": [
    { "id": "abc-123", "label": "stable", "uri": "https://stable.internal:8081" },
    { "id": "def-456", "label": "canary", "uri": "https://canary.internal:8081" }
  ]
}
```

### Other Update scenarios

| HCL change | Backend calls |
|---|---|
| Tweak a `weight` on an existing upstream | 1 PATCH (no new upstreams) |
| Change `uri`/`label`/`tls_context_id` on an existing upstream | 1 PATCH (catalog entry updated by id) |
| Insert a new route in the middle, referencing an existing upstream | 1 PATCH (no new upstream needed) |
| Reorder routes | 1 PATCH (routing array reordered) |
| Add a route with a brand-new upstream | 2 PATCHes (pre-register + main) |
| Add multiple new upstreams across multiple routes | 2 PATCHes (all new upstreams registered together in the pre-PATCH) |
| Remove a route | 1 PATCH (routing array shortened; orphaned upstreams remain in the catalog and are harmless) |

### Caveats

- **Orphan upstreams**: removing a route does not remove its upstreams from the catalog. The backend keeps them; they are inert because no route references them. Cleanup (if desired) is a manual API call today.
- **Reused upstreams**: if two routes reference the same backend (same `label` or `uri`), write the upstream definition in both routes — the provider deduplicates them into a single catalog entry. The dedup key is the resolved `id` (or `label` / `uri` for new upstreams).
- **Drop the `label` field at your own risk**: identity matching falls back to URI, so changing a URI without a label looks like "delete one upstream, create another" — fine for stateless backends, surprising for stateful ones (cookies, sticky sessions, etc.).

## Schema

### Required

- `environment_id` (String) The environment ID where the API instance will be created.
- `spec` (Block) The Exchange asset specification backing this API instance. Required when creating an instance; on import it is populated from the platform, so an imported instance may omit it. See [below for nested schema](#nestedschema--spec).

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `technology` (String) The gateway technology. Only `omniGateway` is currently supported. Defaults to `omniGateway`.
- `provider_id` (String) The identity provider ID for the API.
- `instance_label` (String) A human-readable label for this API instance.
- `approval_method` (String) Client approval method. Valid values: `manual`. Defaults to null (no approval required). **Note:** `automatic` approval is no longer supported.
- `consumer_endpoint` (String) Consumer-facing endpoint URI (the public URL clients use to reach the API). Maps to top-level endpointUri in the API.
- `upstream_uri` (String, **Deprecated**) Shorthand for a single-upstream routing configuration. When set, the provider constructs routing as `[{upstreams: [{weight: 100, uri: <value>}]}]`. Mutually exclusive with the `routing` block. **This field will be removed in the next major version.** Use the `routing` block instead — it supports TLS via `tls_context_id`, labels, multiple upstreams, and weighted routing.
- `gateway_id` (String) The Omni Gateway UUID. When provided, the deployment block is auto-populated by fetching gateway details (target_id, target_name, gateway_version) from the Gateway Manager API. Mutually exclusive with specifying a full deployment block.
- `endpoint` (Block) Endpoint / proxy configuration for the API instance. See [below for nested schema](#nestedschema--endpoint).
- `deployment` (Block) Deployment target configuration. Auto-populated when gateway_id is set. See [below for nested schema](#nestedschema--deployment).
- `routing` (Block List) Routing rules with weighted upstream backends. See [below for nested schema](#nestedschema--routing).

### Read-Only

- `id` (String) The numeric identifier of the API instance (stored as string for Terraform compatibility).
- `status` (String) The current status of the API instance.
- `asset_id` (String) The Exchange asset ID (computed from API response).
- `asset_version` (String) The Exchange asset version (computed from API response).
- `product_version` (String) The product version (computed from API response).

<a id="nestedschema--spec"></a>
### Nested Schema for `spec`

Required:

- `asset_id` (String) The Exchange asset ID.
- `group_id` (String) The Exchange group (organization) ID.
- `version` (String) The asset version.

<a id="nestedschema--endpoint"></a>
### Nested Schema for `endpoint`

Optional:

- `deployment_type` (String) Deployment type. Valid values: `HY` (hybrid), `CH` (CloudHub), `CH2`, `RF` (Runtime Fabric). Defaults to `HY`.
- `type` (String) Endpoint protocol type. Valid values: `http`, `rest`, `raml`, `wsdl`, `graphql`, `grpc`, `websocket`. The value must be compatible with the backing Exchange asset's type — e.g. a `graphql` Exchange asset requires endpoint type `graphql`, and a `grpc-api` asset requires `grpc`. Defaults to `http`.
- `base_path` (String) API base path for the Omni Gateway proxy listener (e.g. `my-api`). The provider constructs the full proxy URI as `http://0.0.0.0:<port>/<base_path>` (see `port` for the listener port). A single gateway can host multiple API instances either on distinct ports or on the same port under distinct, non-root base paths; a base path of `/` is a catch-all that monopolizes the whole port.
- `port` (Number) Listener port for the Omni/Flex Gateway proxy (the port in the constructed proxy URI `http://0.0.0.0:<port>/<base_path>`). Defaults to `8081` for a newly created instance. Set a distinct port to host multiple API instances on the same gateway without sharing a base path. Omitting it on an existing or imported instance keeps whatever port that instance is already using, so upgrading the provider never moves a live listener.
- `response_timeout` (Number) Response timeout in milliseconds.

<a id="nestedschema--deployment"></a>
### Nested Schema for `deployment`

Optional:

- `environment_id` (String) The environment ID for deployment (usually matches the top-level environment_id).
- `type` (String) Deployment type. Valid values: `HY`, `CH`, `RF`. Defaults to `HY`.
- `expected_status` (String) Expected deployment status. Valid values: `deployed`, `undeployed`. Defaults to `deployed`.
- `overwrite` (Boolean) Whether to overwrite an existing deployment.
- `target_id` (String) The target gateway ID to deploy to.
- `target_name` (String) The target gateway name.
- `gateway_version` (String) The Omni Gateway runtime version.

<a id="nestedschema--routing"></a>
### Nested Schema for `routing`

Optional:

- `label` (String) A label for this route.
- `rules` (Block) Match conditions for this route (methods, path, headers). See [below for nested schema](#nestedschema--routing--rules).

Required:

- `upstreams` (Block List) Weighted upstream backends for this route. See [below for nested schema](#nestedschema--routing--upstreams).

<a id="nestedschema--routing--rules"></a>
### Nested Schema for `routing.rules`

Optional:

- `methods` (String) Pipe-separated HTTP methods (e.g. 'GET', 'POST|PUT').
- `path` (String) URL path pattern to match (e.g. '/api/*').
- `host` (String) Host header value to match.
- `headers` (Map) Header key-value pairs to match.

<a id="nestedschema--routing--upstreams"></a>
### Nested Schema for `routing.upstreams`

Required:

- `uri` (String) The upstream backend URI.

Optional:

- `weight` (Number) Traffic weight percentage (0-100). Weights across upstreams should sum to 100. Defaults to `100`.
- `label` (String) A label for this upstream.
- `tls_context_id` (String) TLS context for upstream connections. Format: 'secretGroupId/tlsContextId'.

## Import

An existing API instance can be imported using a composite ID. Use the 2-part form for root-org instances and the 3-part form when the instance belongs to a Business Group (sub-org).

The `api_instance_id` is the numeric ID visible in the Anypoint API Manager URL (e.g. `16563478`).

### Using an import block (Terraform ≥ 1.5 — recommended)

**Root org (2-part ID):**

```terraform
import {
  to = anypoint_api_instance.imported
  id = "<environment_id>/<api_instance_id>"
}

resource "anypoint_api_instance" "imported" {
  environment_id = "<environment_id>"
  spec = {
    asset_id = "<api_asset_id>"
    group_id = "<organization_id>"
    version  = "1.0.0"
  }
}
```

**Sub-org (3-part ID):**

```terraform
import {
  to = anypoint_api_instance.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>"
}

resource "anypoint_api_instance" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  spec = {
    asset_id = "<api_asset_id>"
    group_id = "<organization_id>"
    version  = "1.0.0"
  }
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
# Root org:
terraform import anypoint_api_instance.imported <environment_id>/<api_instance_id>

# Sub-org:
terraform import anypoint_api_instance.imported <organization_id>/<environment_id>/<api_instance_id>
```
