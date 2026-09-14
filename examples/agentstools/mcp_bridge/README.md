# MCP Bridge examples

Turn existing REST APIs into an MCP server without writing MCP code.

| Folder | What it shows |
|--------|----------------|
| [explicit/](./explicit/) | Declare tools by hand (`tools` blocks) |
| [auto_tools/](./auto_tools/) | Auto-create tools from an Exchange REST API asset (`anypoint_mcp_tools`) |
| [import/](./import/) | Commented import template (does not apply anything until uncommented) |

**Managed Flex Gateway:** typically only ports **8081** and **8082**. Port + `base_path` must be unique on the gateway. These two apply folders use different pairs (`8081/petstore` and `8082/petstore-from-spec`) so they can run together. A leftover instance on the same pair will 400.

```bash
cd explicit   # or auto_tools
cp terraform.tfvars.example terraform.tfvars   # fill in values
terraform init && terraform plan && terraform apply
```
