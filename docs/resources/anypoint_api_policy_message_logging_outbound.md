---
page_title: "anypoint_api_policy_message_logging_outbound Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a Message Logging (Outbound) policy on an Anypoint API instance.
---

# anypoint_api_policy_message_logging_outbound (Resource)

Manages a Message Logging (Outbound) policy on an Anypoint API instance.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_api_policy_message_logging_outbound" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    logging_configuration = [
      {
        item_name = "response"
        item_data = {
          message        = "#[payload]"
          conditional    = "#[true]"
          level          = "INFO"
          first_section  = false
          second_section = true
        }
      }
    ]
  }

  upstream_ids = [anypoint_api_upstream.example.id]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID.
- `api_instance_id` (String) The API instance ID.
- `configuration` (Block) The policy configuration. See [Configuration](#nestedschema--configuration) below.
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `label` (String) A human-readable label for this policy instance.
- `asset_version` (String) The policy asset version. Defaults to `2.0.2`.

### Read-Only

- `id` (String) The policy ID.
- `policy_template_id` (String) The policy template ID assigned by the server.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `logging_configuration` (Dynamic) Array of logging rule objects. Each element **must** use the `item_name` + `item_data` wrapper — the Platform rejects any flat field structure with HTTP 400.

**Required structure per element:**

```hcl
logging_configuration = [
  {
    item_name = "<string>"   # unique label for this logging rule
    item_data = {
      message        = "<string>"  # DataWeave expression or literal, e.g. "#[payload]"
      level          = "<string>"  # Log level: DEBUG, INFO, WARN, ERROR (default: INFO)
      conditional    = "<string>"  # Optional DataWeave boolean expression, e.g. "#[true]"
      category       = "<string>"  # Optional logger category name
      first_section  = <bool>      # Log on request phase (default: true)
      second_section = <bool>      # Log on response phase (default: false)
    }
  }
]
```

> **Note:** Do **not** use flat fields (`message`, `level`, etc.) directly inside `configuration` — those are not valid for this policy and will cause an HTTP 400 at apply time.

## Import

An existing `anypoint_api_policy_message_logging_outbound` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_message_logging_outbound.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_message_logging_outbound" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  api_instance_id = "<api_instance_id>"
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
terraform import anypoint_api_policy_message_logging_outbound.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
