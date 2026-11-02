---
page_title: "anypoint_environment Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Fetches information about an Anypoint Platform environment.
---

# anypoint_environment (Data Source)

Fetches information about an Anypoint Platform environment.

-> **Authentication:** Reading an environment works with a standard connected app (client credentials — the default `auth_type = "connected_app"`) that has the appropriate environment read scopes. User authentication (`auth_type = "user"`) is only required to create or update environments (see the [`anypoint_environment` resource](../resources/anypoint_environment.md)).

## Example Usage

```terraform
data "anypoint_environment" "sandbox" {
  id              = "abc123ef-0000-0000-0000-000000000000"
  organization_id = var.organization_id
}

output "env_name" {
  value = data.anypoint_environment.sandbox.name
}
```

## Schema

### Required

- `id` (String) The unique identifier for the environment.

### Optional

- `organization_id` (String) The organization ID where the environment is located. If not specified, uses the organization from provider credentials.

### Read-Only

- `name` (String) The name of the environment.
- `type` (String) The type of the environment (e.g., `design`, `sandbox`, `production`).
- `is_production` (Boolean) Whether this is a production environment.
- `client_id` (String) The client ID associated with the environment.
- `arc_namespace` (String) The ARC namespace for the environment.
- `created_at` (String) The timestamp when the environment was created.
- `updated_at` (String) The timestamp when the environment was last updated.
