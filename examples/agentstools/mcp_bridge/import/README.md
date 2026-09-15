# Import an existing MCP bridge

The provider and variables are live so `terraform init` resolves `mulesoft/anypoint` (not `hashicorp/anypoint`). The `import` block and resource stub stay commented until you have a real instance id.

```bash
cp terraform.tfvars.example terraform.tfvars   # fill creds, org, env, mcp_bridge_id
# uncomment the import {} block in import.tf
terraform init
terraform plan -generate-config-out=generated.tf
terraform apply
terraform plan
```

- Import ID: `organization_id/environment_id/mcp_bridge_id` (numeric API Manager id)
- After import, `source_apis` (including `http_mapping`) is reconstructed from the platform
- `description` and `input_schema` come back null — those live only on the generated Exchange MCP asset

Do not run this at the same time as `explicit/` or `auto_tools/` against the same instance (two states). Do not `terraform destroy` an imported e2e/production bridge unless you intend to delete it.
