---
page_title: "anypoint_api_instance_alert Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages an alert for an API instance in Anypoint Monitoring.
---

# anypoint_api_instance_alert (Resource)

Manages an alert for an API instance in Anypoint Monitoring.

## Example Usage

```terraform
resource "anypoint_api_instance_alert" "high_request_count" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name            = "High Request Count"
  severity        = "warning"
  deployment_type = "HY"
  metric_type     = "api_request_count"

  condition = {
    operator  = "above"
    threshold = 1000
    interval  = 5
  }

  notifications = [
    {
      type       = "email"
      recipients = ["ops-team@example.com"]
      subject    = "Alert: ${severity} - ${api}"
      message    = "API ${api} has exceeded ${condition}. Current value: ${value} at ${timestamp}."
    }
  ]
}
```

## Schema

### Required

- `environment_id` (String) Environment ID.
- `api_instance_id` (String) Numeric ID of the API instance this alert is for.
- `name` (String) Name of the alert.
- `severity` (String) Alert severity. Valid values: `info`, `warning`, `critical`.
- `deployment_type` (String) Deployment type. Accepts shortcodes (`CH`, `CH2`, `HY`, `RF`, `SM`) which are mapped to the API enum values (cloudHub, cloudHub2, hybrid, runtimeFabric, serviceMesh).
- `metric_type` (String) Metric to alert on (e.g. 'api_request_count', 'api_response_time').
- `condition` (Block) Alert trigger condition. See [below for nested schema](#nestedschema--condition).
- `notifications` (Block List) Notification channels for the alert. See [below for nested schema](#nestedschema--notifications).

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `type` (String) Alert type (e.g. 'basic'). Defaults to `basic`.
- `resource_type` (String) Resource type (e.g. 'api'). Defaults to `api`.
- `wildcard_alert` (Boolean) Whether this is a wildcard alert that applies to all APIs. Defaults to `false`.

### Read-Only

- `id` (String) Unique identifier of the alert.

<a id="nestedschema--condition"></a>
### Nested Schema for `condition`

Required:

- `operator` (String) Comparison operator (e.g. 'above', 'below').
- `threshold` (Number) Threshold value that triggers the alert.
- `interval` (Number) Time interval in minutes over which the metric is evaluated.

<a id="nestedschema--notifications"></a>
### Nested Schema for `notifications`

Required:

- `type` (String) Notification type (e.g. 'email').
- `recipients` (List of String) List of recipient email addresses.

Optional:

- `subject` (String) Email subject template. Supports variables like `${severity}`, `${api}`, `${condition}`.
- `message` (String) Email body template. Supports variables like `${api}`, `${condition}`, `${value}`, `${timestamp}`.

## Import

Import is supported using the following format:

```shell
terraform import anypoint_api_instance_alert.example organization_id/environment_id/alert_id
```
