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

### Read-Only

- `client_secret` (String, Sensitive) The client secret. Only returned on creation. Sensitive.
- `created_at` (String) When the connected app was created.
- `id` (String) The client_id of the connected app (unique identifier).
- `owner_user_id` (String) The user ID of the app owner.
- `updated_at` (String) When the connected app was last updated.

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
