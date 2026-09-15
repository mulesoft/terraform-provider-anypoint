# Transit Gateway Connection Example

Attach an AWS Transit Gateway (shared to MuleSoft via AWS RAM) to an Anypoint
Private Space and manage its CIDR routes inline with the
`anypoint_transit_gateway_connection` resource.

## What this example does

- Creates a Transit Gateway attachment on a Private Space.
- Manages the connection's CIDR routes **inline** via the `routes` attribute
  (there is no separate route resource).
- Exposes the attachment `status` and the platform-discovered
  `aws_transit_gateway_id` as outputs.

## Prerequisites

- A Private Space with its network provisioned.
- An AWS Transit Gateway **shared to MuleSoft's AWS account via AWS RAM**. You
  supply the RAM resource share ID (`resource_share_id`) and the AWS account
  that owns the TGW (`resource_share_account`).
- Route CIDRs that do **not** overlap the Private Space CIDR.
- A Connected App (or user) authorized for the CloudHub 2.0 private-space
  control plane — see [Authentication](../../../docs/index.md).

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars   # then edit values
terraform init
terraform plan
terraform apply
```

## Managing routes

`routes` is **required** but may be **empty** (`routes = []`) — an empty list
clears all routes and is the zero-diff shape when importing a detached
connection. Routes are **updated in place**: change the list and re-apply and
the provider replaces the full set of CIDRs on the connection without
recreating it.

## Replacement (ForceNew) fields

Changing any of these forces the connection to be **replaced** (destroy +
create): `organization_id`, `private_space_id`, `resource_share_id`,
`resource_share_account`. `routes` is updated in place. Do **not** set
`create_before_destroy` on this resource: the AWS RAM resource share is
exclusive, so creating the new attachment before destroying the old one fails
with `resource share ... already exists`. Let the replacement destroy first
(the default order) — the provider's two-step teardown de-registers the share
so the new attachment can claim it.

## Import

See [`import.tf`](./import.tf). Import ID is the composite
`organization_id/private_space_id/transit_gateway_id`.

## Destroy

```bash
terraform destroy
```

Destroy detaches the connection from the Private Space **and** de-registers it
at the organization level, so no org-scoped "ghost" attachment is left behind.
