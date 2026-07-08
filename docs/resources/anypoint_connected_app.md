---
page_title: "anypoint_connected_app Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Creates and manages an Anypoint Connected Application.
---

# anypoint_connected_app (Resource)

Creates and manages an Anypoint Connected Application. Connected apps provide a framework for programmatic access to the Anypoint Platform APIs. Two types are supported: 'App acts on behalf of a user' (authorization_code/password/jwt_bearer grants) and 'App acts on its own behalf' (client_credentials grant).

~> **Note:** This is an Access Management resource and requires the **admin provider** (`anypoint.admin`), which uses admin user credentials along with the `client_id` and `client_secret` of a connected app to authenticate on behalf of the user (`auth_type = "user"`). You must set `provider = anypoint.admin` on this resource. The default provider (connected app credentials only) does not have sufficient privileges for Access Management operations.

-> **Connected App:** This resource requires an **admin connected app** configured with `auth_type = "user"` (user credentials + connected app client credentials). Use the `anypoint.admin` provider alias. A standard connected app (client credentials only) does not have sufficient privileges for Access Management operations.

## Example Usage

```terraform
# Admin provider – authenticates on behalf of a user using connected app credentials
provider "anypoint" {
  alias         = "admin"
  auth_type     = "user"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  base_url      = var.anypoint_base_url
}

# Connected app that acts on behalf of a user (OAuth flow)
resource "anypoint_connected_app" "oauth_app" {
  provider     = anypoint.admin
  client_name  = "My OAuth Application"
  grant_types  = ["authorization_code"]
  redirect_uris = ["https://example.com/callback"]
  audience     = "internal"
  enabled      = true
}

# Connected app that acts on its own behalf (client credentials)
resource "anypoint_connected_app" "service_app" {
  provider    = anypoint.admin
  client_name = "Backend Service"
  grant_types = ["client_credentials"]
  audience    = "internal"
  enabled     = true
}

# Connected app with JWT Bearer grant for user impersonation
resource "anypoint_connected_app" "jwt_app" {
  provider    = anypoint.admin
  client_name = "JWT Service App"
  grant_types = ["urn:ietf:params:oauth:grant-type:jwt-bearer"]
  public_keys = [
    file("${path.module}/public_key.pem")
  ]
  audience = "internal"
  enabled  = true
}

# Connected app with inline, authoritative scopes (preferred over the
# deprecated anypoint_connected_app_scopes resource)
resource "anypoint_connected_app" "with_scopes" {
  provider    = anypoint.admin
  client_name = "Automation App"
  grant_types = ["client_credentials"]

  scopes = [
    # Org-scoped scope (identifier form)
    {
      scope          = "create:generations"
      context_params = { org = var.org_id }
    },
    # Environment-scoped scope (needs envId)
    {
      scope          = "read:applications"
      context_params = { org = var.org_id, envId = var.env_id }
    },
    # Display-name form — resolved to its identifier automatically,
    # and preserved as typed in state (no perpetual diff)
    {
      scope          = "Exchange Viewer"
      context_params = { org = var.org_id }
    },
  ]
}
```

## Schema

### Required

- `client_name` (String) The name of the connected app.
- `grant_types` (List of String) The OAuth grant types. Valid values: 'authorization_code', 'password', 'urn:ietf:params:oauth:grant-type:jwt-bearer' (for apps on behalf of a user), or 'client_credentials' (for apps on their own behalf).

### Optional

- `audience` (String) Who can use this application. 'internal' = members of this organization only, 'everyone' = all Anypoint Platform users. Default: 'internal'.
- `client_uri` (String) Website URL where users can learn more about the app.
- `enabled` (Boolean) Whether the connected app is enabled. Default: true.
- `organization_id` (String) The organization ID. Defaults to the provider's org.
- `public_keys` (List of String) Public keys for JWT Bearer grant type.
- `redirect_uris` (List of String) OAuth redirect URIs. Required for 'authorization_code' grant type.
- `scopes` (Attributes Set) Context-aware scopes assigned to the connected application. **Authoritative when set:** the provider makes the app's scopes match this set exactly — scopes assigned out-of-band are removed on the next apply. Omit the block to leave scopes unmanaged; set it to an empty list (`[]`) to remove all user-assigned scopes. Scopes are orthogonal to grant type (they apply to both `client_credentials` and user-behalf apps). Prefer this over the separate, deprecated `anypoint_connected_app_scopes` resource. (see [below for nested schema](#nestedatt--scopes))

~> **Note:** The platform auto-assigns an undeletable `profile` scope to every connected app. It is managed by the platform, never appears in this set, and must not be listed here — the provider ignores it so that a `plan` immediately after `apply` reports no changes.

### Read-Only

- `client_secret` (String, Sensitive) The client secret. Only returned on creation. Sensitive.
- `created_at` (String) When the connected app was created.
- `id` (String) The client_id of the connected app (unique identifier).
- `owner_user_id` (String) The user ID of the app owner.
- `updated_at` (String) When the connected app was last updated.

<a id="nestedatt--scopes"></a>
### Nested Schema for `scopes`

Required:

- `scope` (String) The scope identifier (e.g. `read:applications`, `admin:cloudhub`) or display name (e.g. `CloudHub Admin`). Display names are resolved to identifiers automatically. Use the `anypoint_scopes_catalog` data source to discover available scopes.

Optional:

- `context_params` (Map of String) Context parameters for the scope. Always include `org`; add `envId` for environment-scoped scopes.

## Import

An existing connected app can be imported using its client_id or a composite format `{org_id}:{client_id}`.

~> **Note:** The `client_secret` is not available after import — it's only shown at creation time.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_connected_app.imported
  id       = "<client_id>"
  # Or with organization: id = "<org_id>:<client_id>"
}

resource "anypoint_connected_app" "imported" {
  provider    = anypoint.admin
  client_name = "<client_name>"
  grant_types = ["client_credentials"]
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
terraform import anypoint_connected_app.imported <client_id>
# Or with organization:
terraform import anypoint_connected_app.imported <org_id>:<client_id>
```
