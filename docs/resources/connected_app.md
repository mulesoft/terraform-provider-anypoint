---
page_title: "anypoint_connected_app Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manage Connected Applications in Anypoint Platform.
---

# anypoint_connected_app (Resource)

Manage Connected Applications in Anypoint Platform.

~> **Note:** This is an Access Management resource and requires the **admin provider** (`anypoint.admin`), which uses admin user credentials along with the `client_id` and `client_secret` of a connected app to authenticate on behalf of the user (`auth_type = "user"`). You must set `provider = anypoint.admin` on this resource. The default provider (connected app credentials only) does not have sufficient privileges for Access Management operations.

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

resource "anypoint_connected_app" "example" {
  provider = anypoint.admin
  client_id     = "my-connected-app-client-id"
  owner_org_id  = "your-org-id"
  client_name   = "My Connected App"
  client_secret = "my-client-secret"
  audience      = "internal"

  grant_types = ["client_credentials"]

  scopes       = ["admin:cloudhub"]
  public_keys  = []
  redirect_uris = []

  enabled                          = true
  generate_iss_claim_without_token = false
}
```

## Schema

### Required

- `audience` (String) The audience for the connected application.
- `client_id` (String) The client ID of the connected application.
- `client_name` (String) The name of the connected application.
- `client_secret` (String, Sensitive) The client secret of the connected application.
- `grant_types` (List of String) List of grant types for the connected application.
- `owner_org_id` (String) The organization ID that owns the connected application.

### Optional

- `enabled` (Boolean) Whether the connected application is enabled.
- `generate_iss_claim_without_token` (Boolean) Whether to generate iss claim without token.
- `public_keys` (List of String) List of public keys for the connected application.
- `redirect_uris` (List of String) List of redirect URIs for the connected application.
- `scopes` (List of String) List of scopes for the connected application.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_connected_app.example <client_id>
```
