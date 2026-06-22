---
page_title: "anypoint_privatespace_advanced_config Data Source - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Fetches the advanced configuration for a CloudHub 2.0 private space.
---

# anypoint_privatespace_advanced_config (Data Source)

Fetches the advanced configuration for a CloudHub 2.0 private space.

## Example Usage

```terraform
data "anypoint_privatespace_advanced_config" "example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id
}

output "ingress_protocol" {
  value = data.anypoint_privatespace_advanced_config.example.ingress_configuration.protocol
}

output "enable_iam_role" {
  value = data.anypoint_privatespace_advanced_config.example.enable_iam_role
}
```

## Schema

### Required

- `private_space_id` (String) The ID of the private space to fetch advanced configuration for.

### Optional

- `organization_id` (String) The organization ID. If not provided, the provider's default organization will be used.

### Read-Only

- `enable_iam_role` (Boolean) Whether IAM role is enabled for the private space.
- `id` (String) Identifier for the data source.
- `ingress_configuration` (Object) Ingress configuration for the private space. (see [below for nested schema](#nestedatt--ingress_configuration))

<a id="nestedatt--ingress_configuration"></a>
### Nested Schema for `ingress_configuration`

Read-Only:

- `deployment` (Object) Deployment status information. (see [below for nested schema](#nestedobjatt--ingress_configuration--deployment))
- `logs` (Object) Logs configuration for ingress. (see [below for nested schema](#nestedobjatt--ingress_configuration--logs))
- `protocol` (String) Protocol used for ingress.
- `read_response_timeout` (String) Read response timeout in milliseconds.

<a id="nestedobjatt--ingress_configuration--deployment"></a>
### Nested Schema for `ingress_configuration.deployment`

Read-Only:

- `last_seen_timestamp` (Number) Last seen timestamp for the deployment.
- `status` (String) Deployment status.

<a id="nestedobjatt--ingress_configuration--logs"></a>
### Nested Schema for `ingress_configuration.logs`

Read-Only:

- `filters` (List of Object) List of log filters. (see [below for nested schema](#nestedobjatt--ingress_configuration--logs--filters))
- `port_log_level` (String) Port log level.

<a id="nestedobjatt--ingress_configuration--logs--filters"></a>
### Nested Schema for `ingress_configuration.logs.filters`

Read-Only:

- `ip` (String) IP address for the filter.
- `level` (String) Log level for the filter.
