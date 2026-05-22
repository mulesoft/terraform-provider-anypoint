---
page_title: "anypoint_api_policy_a2a_agent_card Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a A2A Agent Card policy on an Anypoint API instance.
---

# anypoint_api_policy_a2a_agent_card (Resource)

Manages a A2A Agent Card policy on an Anypoint API instance.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_api_policy_a2a_agent_card" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    content        = "{\"name\": \"My Agent\", \"description\": \"An example A2A agent\"}"
    consumer_url   = "https://example.com/agent"
    card_path      = "/.well-known/agent-card.json"
    file_name      = "agent-card.json"
    file_mime_type = "application/json"
  }

  order = 1
}
```

## Schema

### Required

- `environment_id` (String) The environment ID.
- `api_instance_id` (String) The API instance ID.
- `configuration` (Block) The policy configuration. See [Configuration](#nestedschema--configuration) below.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `label` (String) A human-readable label for this policy instance.
- `order` (Number) The order of policy execution.
- `asset_version` (String) The policy asset version. Defaults to `2.0.0-20260327083212`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.

### Read-Only

- `id` (String) The policy ID.
- `policy_template_id` (String) The policy template ID assigned by the server.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `content` (String) The agent card content as a JSON string.

Optional:

- `card_path` (String) Path where the agent card is served.
- `consumer_url` (String) URL for the A2A agent consumer.
- `file_mime_type` (String) MIME type of the agent card file.
- `file_name` (String) Filename for the agent card.
- `file_source` (String) Source of the agent card file.

## Import

An existing `anypoint_api_policy_a2a_agent_card` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_a2a_agent_card.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_a2a_agent_card" "imported" {
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
terraform import anypoint_api_policy_a2a_agent_card.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
