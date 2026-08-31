---
page_title: "anypoint_transit_gateway_connection Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Manages a Transit Gateway connection (attachment) in a CloudHub 2.0 Private Space. A Transit Gateway connection links a Private Space to an existing AWS Transit Gateway (shared to MuleSoft via AWS RAM) for private network connectivity.
---

# anypoint_transit_gateway_connection (Resource)

Manages a Transit Gateway connection (attachment) in a CloudHub 2.0 Private Space. A Transit Gateway connection links a Private Space to an existing AWS Transit Gateway — shared to MuleSoft's AWS account via AWS RAM (Resource Access Manager) — for private network connectivity. This resource manages the *connection/attachment* between the Private Space and the AWS Transit Gateway; it does **not** create the AWS Transit Gateway itself (that lives in your AWS account). The connection goes through `Pending` → `Available` states.

Routes are managed **inline** via the `routes` attribute and can be **updated in place** after the connection is created — there is no separate route resource. Updating `routes` replaces the full set of CIDR routes on the connection.

-> **Prerequisites:** A Private Space with its network provisioned, and an AWS Transit Gateway shared to MuleSoft's AWS account via AWS RAM. The `routes` CIDRs must not overlap with the Private Space CIDR. You can discover the `private_space_id` with the [`anypoint_private_spaces`](../data-sources/anypoint_private_spaces.md) data source. The `resource_share_id` and `resource_share_account` come from your AWS RAM share.

-> **Authentication:** This resource calls the **CloudHub 2.0 private-space (Runtime Manager) control-plane API**. A `client_credentials` Connected App works — grant it the **Cloudhub Organization Admin** (`admin:cloudhub`) scope (plus **Manage Runtime Fabrics** for some operations). A Connected App missing these scopes is rejected with `HTTP 401`/`403` before anything is created; the fix is to add the scopes (or use `auth_type = "user"` with a user that has the equivalent permissions). See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

## Example Usage

```terraform
resource "anypoint_transit_gateway_connection" "main" {
  organization_id        = var.organization_id
  private_space_id       = var.private_space_id
  name                   = "tf-test-tgw"
  resource_share_id      = "e8e330a8-4f8c-452b-afd0-7810c41287f1" # AWS RAM resource share UUID
  resource_share_account = "055970264539"                         # AWS account that owns the TGW
  routes                 = ["192.168.1.0/24", "172.16.0.0/12"]    # >=1 CIDR; must not overlap the PS CIDR

  # IMPORTANT: do not set create_before_destroy here — the AWS RAM resource
  # share is exclusive, so a replacement must destroy the old attachment first
  # (the default order) or the new create fails with
  # "resource share ... already exists".
}

output "transit_gateway_status" {
  value = anypoint_transit_gateway_connection.main.status
}

output "aws_transit_gateway_id" {
  value = anypoint_transit_gateway_connection.main.aws_transit_gateway_id
}
```

### Updating routes

`routes` is updatable in place. Change the list and re-apply — the provider replaces the full set of routes on the connection without recreating it:

```terraform
resource "anypoint_transit_gateway_connection" "main" {
  organization_id        = var.organization_id
  private_space_id       = var.private_space_id
  name                   = "tf-test-tgw"
  resource_share_id      = "e8e330a8-4f8c-452b-afd0-7810c41287f1"
  resource_share_account = "055970264539"
  routes                 = ["192.168.1.0/24", "172.16.0.0/12", "10.50.0.0/16"] # added a CIDR
}
```

## Schema

### Required

- `organization_id` (String) The organization ID.
- `private_space_id` (String) The ID of the Private Space where this transit gateway is attached.
- `name` (String) The name of the transit gateway attachment.
- `resource_share_id` (String) The AWS RAM resource share ID in UUID format (e.g. `e8e330a8-4f8c-452b-afd0-7810c41287f1`).
- `resource_share_account` (String) The AWS account ID that owns the Transit Gateway.
- `routes` (List of String) CIDR routes for the transit gateway connection. The attribute is required (it must be present in the configuration) but may be empty: set `routes = []` to clear all routes, which is also the zero-diff shape when importing a detached connection. Routes are managed inline and can be updated in place; updating them replaces the full set of routes on the connection.

### Read-Only

- `id` (String) The unique identifier of the transit gateway attachment.
- `status` (String) The current status of the transit gateway attachment (e.g. `Pending`, `Available`).
- `aws_transit_gateway_id` (String) The AWS Transit Gateway ID discovered by the platform from the resource share, as a bare `tgw-...` identifier suitable for passing to the AWS provider. This is a computed value set after the TGW attachment is created.
- `aws_console_url` (String) Deep link to this transit gateway in the AWS console, matching the Anypoint UI's "View on AWS" link. Empty when the platform does not supply one. Use `aws_transit_gateway_id` for the identifier itself.

-> **Replacement (ForceNew):** Changing `organization_id`, `private_space_id`, `resource_share_id`, or `resource_share_account` forces the connection to be replaced (destroy + create). `routes` is updated in place. Do **not** set `lifecycle { create_before_destroy = true }`: the AWS RAM resource share is exclusive, so creating the replacement before destroying the old attachment fails with `resource share ... already exists`. The default destroy-first order works because the provider's two-step teardown de-registers the share.

## Import

An existing transit gateway connection can be imported using its composite ID: `organization_id/private_space_id/transit_gateway_id`. On import, the current `routes` are read from the platform and seeded into state.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_transit_gateway_connection.imported
  id = "<organization_id>/<private_space_id>/<transit_gateway_id>"
}

resource "anypoint_transit_gateway_connection" "imported" {
  organization_id        = "<organization_id>"
  private_space_id       = "<private_space_id>"
  name                   = "<name>"
  resource_share_id      = "<resource_share_id>"
  resource_share_account = "<resource_share_account>"
  routes                 = ["<cidr>"]
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
terraform import anypoint_transit_gateway_connection.imported <organization_id>/<private_space_id>/<transit_gateway_id>
```
