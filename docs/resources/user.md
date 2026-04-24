---
page_title: "anypoint_user Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages an Anypoint Platform user.
---

# anypoint_user (Resource)

Manages an Anypoint Platform user.

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

resource "anypoint_user" "example" {
  provider = anypoint.admin
  username                  = "jdoe"
  first_name                = "John"
  last_name                 = "Doe"
  email                     = "jdoe@example.com"
  phone_number              = "555-0100"
  password                  = "SecureP@ssw0rd!"
  mfa_verification_excluded = false
}
```

## Schema

### Required

- `email` (String) The email address of the user.
- `first_name` (String) The first name of the user.
- `last_name` (String) The last name of the user.
- `password` (String, Sensitive) The password for the user. This is only used during creation and updates.
- `username` (String) The username of the user.

### Optional

- `mfa_verification_excluded` (Boolean) Indicates whether the user is excluded from MFA verification. Defaults to `false`.
- `organization_id` (String) The organization ID where the user will be created. If not provided, the organization ID will be inferred from the connected app credentials.
- `phone_number` (String) The phone number of the user.

### Read-Only

- `id` (String) The unique identifier for the user.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_user.example <user_id>
```
