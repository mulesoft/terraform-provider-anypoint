---
page_title: "anypoint_connected_app Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Creates and manages an Anypoint Connected Application.
---

# anypoint_connected_app (Resource)

Creates and manages an Anypoint Connected Application. Connected apps provide a framework for programmatic access to the Anypoint Platform APIs. Two types are supported: 'App acts on behalf of a user' (authorization_code/password/jwt_bearer grants) and 'App acts on its own behalf' (client_credentials grant).

~> **This resource requires `auth_type = "user"`.** A `client_credentials` principal
cannot create a connected app — the create fails with `403: Not authorized to access this
resource`, before scopes are ever applied. **Adding scopes does not fix it** (verified
against an app holding **Access Controls Admin**). Configure the provider with
`auth_type = "user"` as shown below; that still requires a connected app created as "acts
on behalf of a user" with the password grant enabled.
See [Authentication](../index.md).

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

# Connected app that acts on behalf of a user (OAuth flow).
# redirect_uris and client_uri are required for user-behalf apps to be editable in the UI.
resource "anypoint_connected_app" "oauth_app" {
  provider      = anypoint.admin
  name          = "My OAuth Application"
  grant_types   = ["authorization_code"]
  redirect_uris = ["https://example.com/callback"]
  client_uri    = "https://example.com"
  audience      = "internal"
  enabled       = true

  # Scopes on user-behalf apps are stored in the app body (flat field).
  # The provider handles this transparently — usage is identical to client_credentials.
  scopes = [
    {
      scope          = "Exchange Viewer"
      context_params = { org = var.org_id }
    },
  ]
}

# Connected app that acts on its own behalf (client credentials)
resource "anypoint_connected_app" "service_app" {
  provider    = anypoint.admin
  name        = "Backend Service"
  grant_types = ["client_credentials"]
  audience    = "internal"
  enabled     = true
}

# Connected app with JWT Bearer grant for user impersonation.
# redirect_uris and client_uri are required for user-behalf apps to be editable in the UI.
resource "anypoint_connected_app" "jwt_app" {
  provider      = anypoint.admin
  name          = "JWT Service App"
  grant_types   = ["urn:ietf:params:oauth:grant-type:jwt-bearer"]
  redirect_uris = ["https://jwt-service.example.com/callback"]
  client_uri    = "https://jwt-service.example.com"
  public_keys   = [file("${path.module}/public_key.pem")]
  audience      = "internal"
  enabled       = true

  scopes = [
    {
      scope          = "Read Applications"
      context_params = { org = var.org_id, envId = var.env_id }
    },
  ]
}

# Connected app with inline, authoritative scopes (preferred over the
# deprecated anypoint_connected_app_scopes resource).
# Use the display names you see in the Anypoint UI — discover them with
# the anypoint_available_scopes data source.
resource "anypoint_connected_app" "with_scopes" {
  provider    = anypoint.admin
  name        = "Automation App"
  grant_types = ["client_credentials"]

  scopes = [
    # Org-scoped scope (display name as shown in the UI)
    {
      scope          = "Mule Developer Generative AI User"
      context_params = { org = var.org_id }
    },
    # Environment-scoped scope (needs envId)
    {
      scope          = "Read Applications"
      context_params = { org = var.org_id, envId = var.env_id }
    },
    # Another org-scoped scope
    {
      scope          = "Exchange Viewer"
      context_params = { org = var.org_id }
    },
  ]
}
```

## Schema

### Required

- `name` (String) The name of the connected app.
- `grant_types` (List of String) The OAuth grant types. Valid values: 'authorization_code', 'password', 'urn:ietf:params:oauth:grant-type:jwt-bearer' (for apps on behalf of a user), or 'client_credentials' (for apps on their own behalf).

### Optional

- `audience` (String) Who can use this application. 'internal' = members of this organization only, 'everyone' = all Anypoint Platform users. Default: 'internal'.
- `client_uri` (String) Website URL where users can learn more about the app. Required for user-behalf apps (`authorization_code`, `password`, `jwt-bearer`) to be editable in the Anypoint UI.
- `enabled` (Boolean) Whether the connected app is enabled. Default: true.
- `organization_id` (String) The organization ID. Defaults to the provider's org.
- `public_keys` (List of String) Public keys for JWT Bearer grant type.
- `redirect_uris` (List of String) OAuth redirect URIs. Required for user-behalf apps (`authorization_code`, `password`, `jwt-bearer`) to be editable in the Anypoint UI.
- `scopes` (Attributes Set) Scopes assigned to the connected application. **Authoritative when set:** the provider makes the app's scopes match this set exactly — scopes assigned out-of-band (e.g. via the UI) are removed on the next apply. Omit the block to leave scopes unmanaged; set it to an empty list (`[]`) to remove all user-assigned scopes. Scopes work for all grant types — the provider automatically routes to the correct API endpoint: context-aware `/scopes` subresource for `client_credentials`, or the flat body `scopes` field for user-behalf apps (`authorization_code`, `password`, `jwt-bearer`). Prefer this over the separate, deprecated `anypoint_connected_app_scopes` resource. (see [below for nested schema](#nestedatt--scopes))

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

- `scope` (String) The scope display name as shown in the Anypoint UI (e.g. `Read Applications`, `Cloudhub Organization Admin`, `Exchange Viewer`). Discover valid names with the `anypoint_available_scopes` data source. Scope identifiers (e.g. `read:applications`) are also accepted for advanced use.

Optional:

- `context_params` (Map of String) Context parameters for the scope. For `client_credentials` apps, `org` is **required** on every scope (validated at plan time); add `envId` for environment-scoped scopes (e.g. Read Applications). User-behalf apps ignore `context_params`.

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
  name        = "<client_name>"
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
