---
page_title: "anypoint_organization Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Creates and manages an Anypoint Platform organization.
---

# anypoint_organization (Resource)

Creates and manages an Anypoint Platform organization.

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

resource "anypoint_organization" "example" {
  provider = anypoint.admin
  name                   = "my-sub-org"
  parent_organization_id = "parent-org-id"
  owner_id               = "owner-user-id"

  entitlements = {
    create_sub_orgs     = false
    create_environments = true
    global_deployment   = false

    vcores_production = {
      assigned = 0
    }

    vcores_sandbox = {
      assigned = 0
    }

    vcores_design = {
      assigned = 0
    }

    vpcs = {
      assigned = 0
    }

    network_connections = {
      assigned = 0
    }

    managed_gateway_small = {
      assigned = 0
    }

    managed_gateway_large = {
      assigned = 0
    }
  }
}
```

## Schema

### Required

- `name` (String) The name of the organization.
- `owner_id` (String) The ID of the organization owner.
- `parent_organization_id` (String) The ID of the parent organization.

### Optional

- `entitlements` (Block) Entitlements for the organization. Every sub-attribute defaults to its zero value (false for booleans, 0 for quotas). Omitting the whole block is equivalent to declaring `entitlements = {}`. See [below for nested schema](#nestedschema--entitlements).

### Read-Only

- `client_id` (String) The client ID associated with the organization.
- `created_at` (String) The creation timestamp of the organization.
- `deleted_at` (String) The deletion timestamp of the organization.
- `domain` (String) The domain of the organization.
- `environments` (Block List) The environments within the organization. See [below for nested schema](#nestedschema--environments).
- `gdot_id` (String) The GDOT ID of the organization.
- `id` (String) The unique identifier for the organization.
- `idprovider_id` (String) The ID provider ID for the organization.
- `is_automatic_admin_promotion_exempt` (Boolean) Whether the organization is exempt from automatic admin promotion.
- `is_federated` (Boolean) Whether the organization is federated.
- `is_master` (Boolean) Whether the organization is a master organization.
- `is_root` (Boolean) Whether the organization is a root organization.
- `mfa_required` (String) Whether MFA is required for the organization.
- `org_type` (String) The type of the organization.
- `parent_organization_ids` (List of String) List of parent organization IDs.
- `session_timeout` (Number) The session timeout for the organization.
- `sub_organization_ids` (List of String) List of sub-organization IDs.
- `subscription` (Block) The subscription details for the organization. See [below for nested schema](#nestedschema--subscription).
- `tenant_organization_ids` (List of String) List of tenant organization IDs.
- `updated_at` (String) The last update timestamp of the organization.

<a id="nestedschema--entitlements"></a>
### Nested Schema for `entitlements`

Optional:

- `create_environments` (Boolean) Whether environments can be created. Defaults to `false`.
- `create_sub_orgs` (Boolean) Whether sub-organizations can be created. Defaults to `false`.
- `global_deployment` (Boolean) Whether global deployment is enabled. Defaults to `false`.
- `design_center` (Block) Design Center entitlement. See [below for nested schema](#nestedschema--entitlements--design_center).
- `flex_gateway` (Block) Flex Gateway entitlement. See [below for nested schema](#nestedschema--entitlements--enabled_entitlement).
- `gateways` (Block) Gateways entitlement. See [below for nested schema](#nestedschema--entitlements--assigned_entitlement).
- `hybrid` (Block) Hybrid entitlement. See [below for nested schema](#nestedschema--entitlements--enabled_entitlement).
- `load_balancer` (Block) Load balancer entitlement. See [below for nested schema](#nestedschema--entitlements--assigned_entitlement).
- `managed_gateway_large` (Block) Managed Gateway (large) entitlement. See [below for nested schema](#nestedschema--entitlements--assigned_entitlement).
- `managed_gateway_small` (Block) Managed Gateway (small) entitlement. See [below for nested schema](#nestedschema--entitlements--assigned_entitlement).
- `mq_messages` (Block) MQ messages entitlement. See [below for nested schema](#nestedschema--entitlements--mq_entitlement).
- `mq_requests` (Block) MQ requests entitlement. See [below for nested schema](#nestedschema--entitlements--mq_entitlement).
- `network_connections` (Block) Network connections entitlement. See [below for nested schema](#nestedschema--entitlements--vcore_entitlement).
- `runtime_fabric` (Boolean) Whether Runtime Fabric is enabled.
- `service_mesh` (Block) Service Mesh entitlement. See [below for nested schema](#nestedschema--entitlements--enabled_entitlement).
- `vcores_design` (Block) Design vCore entitlement. See [below for nested schema](#nestedschema--entitlements--vcore_entitlement).
- `vcores_production` (Block) Production vCore entitlement. See [below for nested schema](#nestedschema--entitlements--vcore_entitlement).
- `vcores_sandbox` (Block) Sandbox vCore entitlement. See [below for nested schema](#nestedschema--entitlements--vcore_entitlement).
- `vpcs` (Block) VPC entitlement. See [below for nested schema](#nestedschema--entitlements--vcore_entitlement).
- `worker_logging_override` (Block) Worker logging override entitlement. See [below for nested schema](#nestedschema--entitlements--enabled_entitlement).

> **Note:** `static_ips` and `vpns` entitlements are managed server-side by
> Anypoint and are not settable via Terraform, so they are intentionally
> omitted from this schema. Configure them through the Anypoint UI or API.

<a id="nestedschema--entitlements--vcore_entitlement"></a>
### Nested Schema for `vcores_production` / `vcores_sandbox` / `vcores_design` / `vpcs` / `network_connections`

Optional:

- `assigned` (Number) The number of assigned units. Defaults to `0`.
- `reassigned` (Number) The number of reassigned units. Defaults to `0`.

<a id="nestedschema--entitlements--enabled_entitlement"></a>
### Nested Schema for `hybrid` / `flex_gateway` / `worker_logging_override` / `service_mesh`

Optional:

- `enabled` (Boolean) Whether this feature is enabled.

<a id="nestedschema--entitlements--assigned_entitlement"></a>
### Nested Schema for `gateways` / `load_balancer` / `managed_gateway_small` / `managed_gateway_large`

Optional:

- `assigned` (Number) The number of assigned units.

<a id="nestedschema--entitlements--mq_entitlement"></a>
### Nested Schema for `mq_messages` / `mq_requests`

Optional:

- `add_on` (Number) The add-on number of MQ units. Defaults to `0`.
- `base` (Number) The base number of MQ units. Defaults to `0`.

<a id="nestedschema--entitlements--design_center"></a>
### Nested Schema for `design_center`

Optional:

- `api` (Boolean) Whether API Designer is enabled.
- `mozart` (Boolean) Whether Flow Designer (Mozart) is enabled.

<a id="nestedschema--subscription"></a>
### Nested Schema for `subscription`

Read-Only:

- `category` (String) The subscription category.
- `expiration` (String) The subscription expiration date.
- `type` (String) The subscription type.

Optional:

- `justification` (String) The subscription justification.

<a id="nestedschema--environments"></a>
### Nested Schema for `environments`

Read-Only:

- `client_id` (String) The environment client ID.
- `id` (String) The environment ID.
- `is_production` (Boolean) Whether the environment is a production environment.
- `name` (String) The environment name.
- `organization_id` (String) The organization ID.
- `type` (String) The environment type.

Optional:

- `arc_namespace` (String) The ARC namespace of the environment.

## Import

Existing Anypoint organizations can be imported using their organization ID:

```shell
terraform import anypoint_organization.example_org 00000000-0000-0000-0000-000000000000
```

Your HCL must declare `name`, `parent_organization_id`, and `owner_id` before
you import — those are Required attributes on the resource. The first
`terraform plan` after import refreshes all Read-Only / Optional attributes
(including entitlements) from the Anypoint API.

`parent_organization_id` is derived from the server-returned ancestor chain
(`parent_organization_ids`) on the first refresh. If the derivation doesn't
match what you wrote in HCL, update the HCL to match — changing
`parent_organization_id` would otherwise trigger a destroy+recreate because it
has the `RequiresReplace` plan modifier.
