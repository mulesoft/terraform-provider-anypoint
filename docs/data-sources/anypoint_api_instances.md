---
page_title: "anypoint_api_instances Data Source - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Lists all API instances registered in API Manager for the given environment.
---

# anypoint_api_instances (Data Source)

Lists all API instances registered in API Manager for the given environment.

## Example Usage

```terraform
data "anypoint_api_instances" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "api_instance_ids" {
  value = [for inst in data.anypoint_api_instances.all.instances : inst.id]
}
```

### List only the instances deployed to a specific gateway

Use `gateway_id` to list the API instances attached to a given gateway (for
example a self-managed gateway). This is the reverse of the write path, where
an `anypoint_api_instance` references its gateway via `gateway_id`: here the
data source discovers every instance — including ones created outside
Terraform, such as through the Anypoint UI — that targets that gateway.

```terraform
data "anypoint_api_instances" "on_gateway" {
  environment_id = var.environment_id
  gateway_id     = anypoint_self_managed_gateway.gw.gateway_id
}

output "instances_on_gateway" {
  value = [for inst in data.anypoint_api_instances.on_gateway.instances : inst.id]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID to list API instances from.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.
- `gateway_id` (String) Optional filter: only return API instances deployed to the gateway (target) with this ID. Matches the instance deployment's target ID (e.g. a self-managed gateway ID). When omitted, all instances in the environment are returned.

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>`.
- `instances` (List of Object) List of API instances. See [`instances`](#nestedschema--instances) below.

<a id="nestedschema--instances"></a>
### Nested Schema for `instances`

Read-Only:

- `id` (String) The numeric ID of the API instance.
- `asset_id` (String) The Exchange asset ID.
- `asset_version` (String) The Exchange asset version.
- `product_version` (String) The product version.
- `group_id` (String) The Exchange group (organization) ID.
- `technology` (String) The gateway technology (e.g., `omniGateway`).
- `instance_label` (String) The label of the API instance.
- `status` (String) The current status of the API instance.
- `endpoint_uri` (String) The endpoint URI for the API instance.
- `gateway_id` (String) The ID of the gateway (deployment target) this API instance is deployed to, if any. Null for instances without a gateway deployment (e.g. CloudHub/basic-endpoint instances).
- `autodiscovery_instance_name` (String) The autodiscovery instance name.
