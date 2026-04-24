---
page_title: "anypoint_api_governance_profile Resource - terraform-provider-anypoint"
subcategory: "Governance"
description: |-
  Manages an API Governance profile in Anypoint Platform. A governance profile applies Exchange rulesets to APIs matching a filter criteria.
---

# anypoint_api_governance_profile (Resource)

Manages an API Governance profile in Anypoint Platform. A governance profile applies Exchange rulesets to APIs matching a filter criteria.

## Example Usage

```terraform
resource "anypoint_api_governance_profile" "example" {
  organization_id = var.organization_id
  name            = "API Best Practices"
  description     = "Enforce best practices across all HTTP APIs"
  filter          = "scope:http-api"

  rulesets = [
    {
      group_id = var.mulesoft_ruleset_group_id
      asset_id = "api-catalog-information-best-practices"
      version  = "latest"
    },
    {
      group_id = var.mulesoft_ruleset_group_id
      asset_id = "api-documentation-best-practices"
      version  = "latest"
    }
  ]

  notification_config = {
    enabled = true
    notifications = [
      {
        enabled   = true
        condition = "OnFailure"
        recipients = [
          {
            contact_type      = "Publisher"
            notification_type = "Email"
            value             = ""
            label             = ""
          }
        ]
      }
    ]
  }
}
```

## Schema

### Required

- `name` (String) Name of the governance profile.
- `rulesets` (Block List) List of Exchange rulesets to apply. Each ruleset is identified by group_id, asset_id, and version. See [`rulesets`](#nestedschema--rulesets) below.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `description` (String) Description of the governance profile.
- `filter` (String) Filter expression to select APIs (e.g. `scope:http-api`). Defaults to `scope:http-api`.
- `allowing` (List of String) List of API identifiers to explicitly allow (exceptions).
- `denying` (List of String) List of API identifiers to explicitly deny.
- `notification_config` (Block) Notification configuration for governance violations. See [`notification_config`](#nestedschema--notification_config) below.

### Read-Only

- `id` (String) Unique identifier of the governance profile.

<a id="nestedschema--rulesets"></a>
### Nested Schema for `rulesets`

Required:

- `group_id` (String) Exchange group ID of the ruleset.
- `asset_id` (String) Exchange asset ID of the ruleset (e.g. `anypoint-best-practices`).

Optional:

- `version` (String) Version of the ruleset (e.g. `1.0.0` or `latest`). Defaults to `latest`.

<a id="nestedschema--notification_config"></a>
### Nested Schema for `notification_config`

Optional:

- `enabled` (Boolean) Whether notifications are enabled. Defaults to `true`.
- `notifications` (Block List) List of notification rules. See [`notification_config.notifications`](#nestedschema--notification_config--notifications) below.

<a id="nestedschema--notification_config--notifications"></a>
### Nested Schema for `notification_config.notifications`

Optional:

- `enabled` (Boolean) Whether this notification rule is enabled. Defaults to `true`.
- `condition` (String) When to send notification (e.g. `OnFailure`). Defaults to `OnFailure`.
- `recipients` (Block List) Notification recipients. See [`notification_config.notifications.recipients`](#nestedschema--notification_config--notifications--recipients) below.

<a id="nestedschema--notification_config--notifications--recipients"></a>
### Nested Schema for `notification_config.notifications.recipients`

Optional:

- `contact_type` (String) Contact type (e.g. `Publisher`). Defaults to `Publisher`.
- `notification_type` (String) Notification channel type (e.g. `Email`). Defaults to `Email`.
- `value` (String) Recipient value (e.g. email address). Empty for publisher-type contacts.
- `label` (String) Recipient label.
