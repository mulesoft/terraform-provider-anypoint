# Anypoint Agents Tools Resources

This directory contains Terraform examples for the new **Agents Tools** category, which includes resources for managing AI agent instances and MCP (Model Context Protocol) servers on the Anypoint Platform.

## Overview

The Agents Tools category provides two main resources:

1. **Agent Instance** (`anypoint_agent_instance`) - Deploy AI agents that can consume tools and resources from MCP servers
2. **MCP Server** (`anypoint_mcp_server`) - Deploy MCP servers that expose tools, resources, and prompts to AI agents

## Resources

### `anypoint_agent_instance`

Creates and manages an AI agent instance in API Manager. Agent instances can:
- Access MCP servers for tool use
- Be managed with policies and SLA tiers like regular APIs
- Route to a backend agent implementation

**Key Features:**
- Integration with Flex Gateway
- Support for Exchange asset specifications
- `upstream_uri` shorthand or explicit `routing` block (one upstream per route)
- `status` is populated automatically after creation via a POST + GET

> **Note:** Only one upstream per route is supported. Multi-upstream weighted routing is not available for agent instances.

### `anypoint_mcp_server`

Creates and manages an MCP (Model Context Protocol) server instance. MCP servers:
- Expose tools and resources that agents can use
- Follow the Model Context Protocol standard
- Deploy to Flex Gateway with custom proxy URIs

**Key Features:**
- MCP-specific endpoint type (`type = "mcp"`)
- Custom proxy URI via `base_path` (e.g., `base_path = "mcp/atlassian"` → `http://0.0.0.0:8081/mcp/atlassian`)
- `upstream_uri` shorthand or explicit `routing` block (one upstream per route)
- `upstream_id` computed attribute for referencing in outbound policies

> **Note:** Only one upstream per route is supported. Multi-upstream weighted routing is not available for MCP servers.

## Data Sources

### `anypoint_agent_instances`

Lists all agent instances in an environment. Useful for:
- Discovering deployed agents
- Auditing agent deployments
- Building dashboards

### `anypoint_mcp_servers`

Lists all MCP servers in an environment. Useful for:
- Discovering available MCP servers
- Inventory management
- Integration planning

## Examples

### Basic Examples

1. **[agent_instance/](./agent_instance/)** - Simple agent instance deployments
   - Single agent with basic configuration
   - A/B testing with weighted routing
   
2. **[mcp_server/](./mcp_server/)** - MCP server deployments
   - Atlassian MCP server (Jira/Confluence)
   - Salesforce MCP server
   - High-availability MCP cluster

3. **[complete/](./complete/)** - Comprehensive example
   - Multiple MCP servers
   - Multiple agent instances
   - Data source queries
   - Complete infrastructure setup

## Quick Start

### Prerequisites

- Anypoint Platform account
- Connected App credentials
- Existing organization and environment
- Deployed Flex Gateway

### Deploy an Agent Instance

```bash
cd agent_instance
terraform init
terraform plan \
  -var="anypoint_client_id=YOUR_CLIENT_ID" \
  -var="anypoint_client_secret=YOUR_SECRET" \
  -var="organization_id=YOUR_ORG_ID" \
  -var="environment_id=YOUR_ENV_ID" \
  -var="gateway_id=YOUR_GATEWAY_ID"
terraform apply
```

### Deploy an MCP Server

```bash
cd mcp_server
terraform init
terraform plan \
  -var="anypoint_client_id=YOUR_CLIENT_ID" \
  -var="anypoint_client_secret=YOUR_SECRET" \
  -var="organization_id=YOUR_ORG_ID" \
  -var="environment_id=YOUR_ENV_ID" \
  -var="gateway_id=YOUR_GATEWAY_ID"
terraform apply
```

### Deploy Complete Infrastructure

```bash
cd complete
terraform init
terraform apply
```

## Architecture

The Agents Tools resources enable this architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                  Anypoint Flex Gateway                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐         ┌──────────────────────┐      │
│  │  Agent          │         │  MCP Servers         │      │
│  │  Instances      │────────▶│  (Tool Providers)    │      │
│  │                 │         │                       │      │
│  │ • Support Agent │         │ • Atlassian MCP      │      │
│  │ • Sales Agent   │         │ • Salesforce MCP     │      │
│  │ • Analytics     │         │ • Database MCP       │      │
│  └─────────────────┘         └──────────────────────┘      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
         │                                 │
         ▼                                 ▼
  ┌─────────────┐                  ┌──────────────┐
  │  AI Model   │                  │  Enterprise  │
  │  Backends   │                  │  Systems     │
  └─────────────┘                  └──────────────┘
```

## Key Concepts

### Agent Instance vs MCP Server

- **Agent Instance**: The AI agent itself that makes decisions and executes tasks
- **MCP Server**: The tool provider that agents can call to access enterprise systems

Think of it like:
- Agent Instance = The brain (AI model)
- MCP Server = The hands (tools to interact with systems)

### Routing

Both resource types support the `routing` block with a list of routes, each containing a single upstream. Multi-upstream weighted routing is **not** supported — each route must have exactly one upstream (weight defaults to 100).

Example with a single upstream route:
```hcl
routing = [
  {
    upstreams = [
      { weight = 100, uri = "http://agent-backend.internal:8080" }
    ]
  }
]
```

For multiple routes (e.g. read vs write separation), use multiple route entries:
```hcl
routing = [
  {
    label = "reads"
    rules = { methods = "GET" }
    upstreams = [{ uri = "http://read-backend.internal:8080" }]
  },
  {
    label = "writes"
    rules = { methods = "POST|PUT|DELETE" }
    upstreams = [{ uri = "http://write-backend.internal:8080" }]
  }
]
```

### MCP Proxy URI

MCP servers use a special proxy URI format:
```
http://0.0.0.0:8081/mcp/<server-name>
```

This allows:
- Multiple MCP servers on the same gateway
- Unique paths for each tool provider
- Easy routing and discovery

## Best Practices

1. **Resource Naming**: Use descriptive labels that indicate the agent's purpose
2. **Weighted Routing**: Start with conservative weights (90/10) for A/B tests
3. **Dependencies**: Use `depends_on` to ensure MCP servers deploy before agents
4. **Environment Promotion**: Test in sandbox before promoting to production
5. **Monitoring**: Add alerts and policies to track agent performance

## Common Use Cases

### Customer Support Agent

```hcl
resource "anypoint_agent_instance" "support" {
  instance_label = "customer-support-agent"
  spec = {
    asset_id = "support-agent"
    version  = "1.0.0"
  }
  upstream_uri = "http://support-agent.internal:8080"
}
```

### Multi-Tool MCP Server

```hcl
resource "anypoint_mcp_server" "enterprise_tools" {
  instance_label = "enterprise-tools-mcp"
  endpoint = {
    base_path = "mcp/tools"
  }
  upstream_uri = "http://mcp-tools.internal:8080"
}
```

### Multiple Route Setup (Read/Write Separation)

```hcl
resource "anypoint_agent_instance" "separated" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  instance_label  = "sales-agent"
  spec = {
    asset_id = "sales-agent-spec"
    group_id = var.organization_id
    version  = "1.0.0"
  }
  endpoint = {
    base_path = "agent/sales"
  }
  gateway_id = var.gateway_id
  routing = [
    {
      label     = "reads"
      rules     = { methods = "GET" }
      upstreams = [{ uri = "http://sales-agent-read.internal:8080" }]
    },
    {
      label     = "writes"
      rules     = { methods = "POST|PUT|DELETE" }
      upstreams = [{ uri = "http://sales-agent-write.internal:8080" }]
    }
  ]
}
```

## Troubleshooting

### Agent Not Responding
1. Check agent instance status: `terraform output agent_status`
2. Verify gateway is running
3. Check routing configuration
4. Review backend logs

### MCP Server Connection Issues
1. Verify proxy URI is unique
2. Check MCP server status
3. Test backend connectivity
4. Review gateway logs

### Build Errors
If you encounter build errors when working with these resources:
1. Ensure Go 1.21+ is installed
2. Run `go mod tidy`
3. Rebuild: `make build`
4. Reinstall: `make install`

## Next Steps

- **Add Policies**: Secure agents with JWT validation, rate limiting
- **Configure SLA Tiers**: Enable consumer self-service
- **Set Up Monitoring**: Add alerts for agent performance

## Related Documentation

- [Terraform Provider README](../../README.md)
- [API Management Examples](../apimanagement/)
- [Secrets Management](../secretsmanagement/)
- [MCP Specification](https://modelcontextprotocol.io)

## Support

For issues or questions:
- GitHub Issues: [terraform-provider-anypoint/issues](https://github.com/mulesoft/terraform-provider-anypoint/issues)
- MuleSoft Docs: [docs.mulesoft.com](https://docs.mulesoft.com/)
