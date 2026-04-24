# Complete Agent Tools Example

This example demonstrates a complete AI agents infrastructure setup with MCP (Model Context Protocol) servers and Agent instances working together on the Anypoint Platform.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Anypoint Flex Gateway                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐ │
│  │  MCP Servers     │  │  Agent Instances │  │  Routing      │ │
│  ├──────────────────┤  ├──────────────────┤  ├───────────────┤ │
│  │ • Atlassian MCP  │  │ • Support Agent  │  │ • Load        │ │
│  │ • Salesforce MCP │  │ • Sales Agent    │  │   Balancing   │ │
│  │ • Database MCP   │  │ • Analytics      │  │ • A/B Testing │ │
│  └──────────────────┘  └──────────────────┘  └───────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
            │                     │                      │
            ▼                     ▼                      ▼
   ┌────────────────┐   ┌──────────────────┐  ┌──────────────┐
   │ Atlassian      │   │ Salesforce CRM   │  │ Database     │
   │ (Jira/Conf)    │   │                  │  │ (Analytics)  │
   └────────────────┘   └──────────────────┘  └──────────────┘
```

## What's Included

### MCP Servers (Tool Providers)
1. **Atlassian MCP** - Provides access to Jira and Confluence
2. **Salesforce MCP** - Exposes CRM data and operations
3. **Database MCP** - Enables analytics and reporting queries

### Agent Instances (AI Agents)
1. **Customer Support Agent** - Handles customer inquiries
2. **Sales Agent** - Provides sales assistance (with A/B testing)
3. **Analytics Agent** - Generates reports and insights

### Features Demonstrated
- MCP server deployment with custom proxy URIs
- Agent instance deployment with routing configuration
- A/B testing with weighted traffic distribution (80/20 split)
- Resource dependencies between agents and MCP servers
- Data sources for querying deployed instances

## Prerequisites

1. Anypoint Platform account with appropriate permissions
2. Connected App credentials (client ID and secret)
3. An existing organization and environment
4. A deployed Flex Gateway

## Usage

1. **Set up credentials**:
   ```bash
   export TF_VAR_anypoint_client_id="your-client-id"
   export TF_VAR_anypoint_client_secret="your-client-secret"
   export TF_VAR_organization_id="your-org-id"
   export TF_VAR_environment_id="your-env-id"
   export TF_VAR_gateway_id="your-gateway-id"
   ```

2. **Initialize Terraform**:
   ```bash
   terraform init
   ```

3. **Preview changes**:
   ```bash
   terraform plan
   ```

4. **Deploy**:
   ```bash
   terraform apply
   ```

5. **View outputs**:
   ```bash
   terraform output
   ```

6. **Clean up**:
   ```bash
   terraform destroy
   ```

## Resource Creation Order

The `depends_on` declarations ensure resources are created in the correct order:

1. **MCP Servers** (parallel creation)
   - Atlassian MCP
   - Salesforce MCP
   - Database MCP

2. **Agent Instances** (after MCP servers)
   - Customer Support Agent (depends on Atlassian + Salesforce)
   - Sales Agent (depends on Salesforce + Database)
   - Analytics Agent (depends on Database + Salesforce)

## Key Concepts

### MCP (Model Context Protocol)
MCP servers expose tools, resources, and prompts that AI agents can use. Each MCP server:
- Has a unique proxy URI (e.g., `/mcp/atlassian`)
- Connects to backend systems
- Provides a standardized interface for agents

### Agent Instances
Agent instances are AI models deployed behind API Manager that:
- Can access MCP servers for tool use
- Support advanced routing (A/B testing, canary deployments)
- Are managed as API instances with policies and SLA tiers

### Weighted Routing
The Sales Agent demonstrates A/B testing:
- 80% of traffic → Stable model
- 20% of traffic → New model
- Useful for gradual rollouts and performance comparison

## Outputs

After deployment, you'll see:
- IDs and statuses of all MCP servers
- IDs and statuses of all agent instances
- Total counts from data sources
- Lists of all deployed IDs

## Next Steps

- Add API policies to agents (rate limiting, JWT validation)
- Configure SLA tiers for consumer self-service
- Set up alerts for monitoring agent health
- Promote agents from sandbox to production
