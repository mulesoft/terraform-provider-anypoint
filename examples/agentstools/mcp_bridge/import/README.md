# Import an existing MCP bridge

`import.tf` is **fully commented**. Terraform in this folder does nothing until you uncomment it.

It shows how to bring an already-created bridge into state:

- Import ID: `organization_id/environment_id/mcp_bridge_id` (numeric API Manager id)
- After import, `source_apis` (including `http_mapping`) is reconstructed from the platform
- `description` and `input_schema` come back null — those live only on the generated Exchange MCP asset

Do not run this at the same time as `explicit/` or `auto_tools/` unless you have uncommented it and pointed it at a real instance.
