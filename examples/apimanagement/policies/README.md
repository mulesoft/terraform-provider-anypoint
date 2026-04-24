# API Policies Example

This example demonstrates how to apply and configure API policies to protect and manage your APIs in Anypoint Platform using Terraform.

## Overview

API policies provide security, rate limiting, transformation, and monitoring capabilities for your APIs. This example showcases 16 commonly used policies organized by category:

- **Security** (5 policies)
- **Rate Limiting** (3 policies)
- **Traffic Management** (3 policies)
- **Threat Protection** (2 policies)
- **Monitoring** (2 policies)
- **Caching** (1 policy)

## Prerequisites

Before running this example, you need:

1. **Anypoint Platform Account** with API Manager permissions
2. **Connected App Credentials** (Client ID and Secret)
3. **Existing API Instance** - Get the numeric API instance ID
4. **Environment ID** - Where the API is deployed

### Finding Your API Instance ID

```bash
# Using Anypoint CLI
anypoint-cli api-manager api list --environment-id=<env-id>

# Or from UI URL:
# .../apis/12345678 <- This is your API Instance ID
```

## Policies Included

### Security Policies

#### 1. Client ID Enforcement
Requires API consumers to provide valid client credentials:
```hcl
configuration = {
  credentials_origin_has_http_basic_authentication_header = "customExpression"
  client_id_expression     = "#[attributes.headers['client_id']]"
  client_secret_expression = "#[attributes.headers['client_secret']]"
}
```

#### 2. JWT Validation
Validates JSON Web Tokens for OAuth 2.0/OpenID Connect:
```hcl
configuration = {
  jwt_origin                      = "httpBearerAuthenticationHeader"
  signing_method                  = "rsa"
  signing_key_length              = 256
  validate_aud_claim              = true
  mandatory_exp_claim             = true
  # ... additional JWT settings
}
```

#### 3. IP Allowlist
Restricts access to specific IP addresses or ranges:
```hcl
configuration = {
  ip_expression = "#[attributes.headers['x-forwarded-for']]"
  ips           = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
}
```

#### 4. IP Blocklist
Blocks specific IP addresses or ranges:
```hcl
configuration = {
  ip_expression = "#[attributes.headers['x-forwarded-for']]"
  ips           = ["203.0.113.0/24", "198.51.100.0/24"]
}
```

#### 5. Basic Authentication
Simple username/password authentication:
```hcl
configuration = {
  username = "admin"
  password = "changeme"
}
```

### Rate Limiting Policies

#### 6. Rate Limiting
General-purpose rate limiting based on key selector:
```hcl
configuration = {
  key_selector = "#[attributes.queryParams['identifier']]"
  rate_limits = [
    {
      maximum_requests            = 100
      time_period_in_milliseconds = 60000
    }
  ]
  expose_headers = true
  clusterizable  = true
}
```

#### 7. Spike Control
Prevents traffic bursts with queuing:
```hcl
configuration = {
  maximum_requests            = 1
  time_period_in_milliseconds = 1000
  delay_time_in_millis        = 1000
  delay_attempts              = 1
  queuing_limit               = 5
  expose_headers              = true
}
```

#### 8. SLA-Based Rate Limiting
Rate limits based on API consumer tier:
```hcl
configuration = {
  client_id_expression     = "#[attributes.headers['client_id']]"
  client_secret_expression = "#[attributes.headers['client_secret']]"
  expose_headers           = true
  clusterizable            = true
}
```

### Traffic Management Policies

#### 9. CORS
Enable Cross-Origin Resource Sharing:
```hcl
configuration = {
  public_resource     = true
  support_credentials = false
  origin_groups       = []
}
```

#### 10. Header Injection
Add headers to requests and responses:
```hcl
configuration = {
  inbound_headers = [
    { key = "X-Request-ID", value = "#[uuid()]" },
    { key = "X-Request-Time", value = "#[now()]" }
  ]
  outbound_headers = [
    { key = "X-API-Version", value = "v1.0" },
    { key = "X-Response-Time", value = "#[now()]" }
  ]
}
```

#### 11. Header Removal
Remove sensitive headers:
```hcl
configuration = {
  inbound_headers  = ["X-Internal-Debug", "X-Temp-Token"]
  outbound_headers = ["Server", "X-Powered-By"]
}
```

### Threat Protection Policies

#### 12. JSON Threat Protection
Protects against malicious JSON payloads:
```hcl
configuration = {
  max_container_depth          = 5
  max_string_value_length      = 1000
  max_object_entry_name_length = 100
  max_object_entry_count       = 100
  max_array_element_count      = 100
}
```

#### 13. XML Threat Protection
Protects against malicious XML payloads:
```hcl
configuration = {
  max_node_depth                  = 10
  max_attribute_count_per_element = 10
  max_child_count                 = 100
  max_text_length                 = 1000
  max_attribute_length            = 100
  max_comment_length              = 500
}
```

### Monitoring Policies

#### 14. Message Logging
Log request/response details:
```hcl
configuration = {
  logging_configuration = [
    {
      item_name = "Request Logging"
      item_data = {
        message        = "#[attributes.headers['request-id']]"
        conditional    = "#[attributes.method == 'POST']"
        category       = "api-requests"
        level          = "INFO"
        first_section  = true
        second_section = true
      }
    }
  ]
}
```

#### 15. Response Timeout
Set maximum response time:
```hcl
configuration = {
  timeout = 30  # seconds
}
```

### Caching Policies

#### 16. HTTP Caching
Cache responses to improve performance:
```hcl
configuration = {
  http_caching_key       = "#[attributes.requestPath ++ '?' ++ attributes.queryString]"
  max_cache_entries      = 10000
  ttl                    = 600  # seconds
  distributed            = true
  persist_cache          = true
  use_http_cache_headers = true
  invalidation_header    = "X-Cache-Invalidate"
}
```

## Usage

### Step 1: Set Required Variables

Create a `terraform.tfvars` file:

```hcl
anypoint_client_id     = "your-client-id"
anypoint_client_secret = "your-client-secret"
anypoint_base_url      = "https://anypoint.mulesoft.com"

organization_id = "your-org-id"
environment_id  = "your-env-id"
api_instance_id = "12345678"  # Numeric API instance ID
```

### Step 2: Initialize Terraform

```bash
terraform init
```

### Step 3: Review the Plan

```bash
terraform plan
```

### Step 4: Apply Policies

```bash
terraform apply
```

### Step 5: Verify Policies

```bash
# View applied policies
terraform output policy_summary

# Check in API Manager UI
# Navigate to: API Manager → Your API → Policies
```

## Policy Order

Policies are executed in the order specified. This example uses:

1. Client ID Enforcement (order: 1)
2. JWT Validation (order: 2)
3. IP Allowlist (order: 3)
4. IP Blocklist (order: 4)
5. Basic Auth (order: 5)
6. Rate Limiting (order: 6)
7. Spike Control (order: 7)
8. SLA Rate Limiting (order: 8)
9. CORS (order: 9)
10. Header Injection (order: 10)
11. Header Removal (order: 11)
12. JSON Threat Protection (order: 12)
13. XML Threat Protection (order: 13)
14. Message Logging (order: 14)
15. Response Timeout (order: 15)
16. HTTP Caching (order: 16)

## Common Policy Patterns

### Pattern 1: Basic Security Stack

```hcl
# 1. Client ID enforcement
resource "anypoint_api_policy_client_id_enforcement" "auth" {
  order = 1
  # ... configuration
}

# 2. Rate limiting
resource "anypoint_api_policy_rate_limiting" "rate_limit" {
  order = 2
  # ... configuration
}

# 3. JSON threat protection
resource "anypoint_api_policy_json_threat_protection" "json_protection" {
  order = 3
  # ... configuration
}
```

### Pattern 2: OAuth 2.0 with JWT

```hcl
# 1. JWT validation
resource "anypoint_api_policy_jwt_validation" "jwt" {
  order = 1
  configuration = {
    jwt_origin     = "httpBearerAuthenticationHeader"
    signing_method = "rsa"
    # ... JWT configuration
  }
}

# 2. SLA-based rate limiting
resource "anypoint_api_policy_rate_limiting_sla_based" "sla_rate" {
  order = 2
  # ... configuration
}
```

### Pattern 3: Public API with CORS

```hcl
# 1. IP allowlist (optional)
resource "anypoint_api_policy_ip_allowlist" "allowed_ips" {
  order = 1
  # ... configuration
}

# 2. CORS
resource "anypoint_api_policy_cors" "cors" {
  order = 2
  configuration = {
    public_resource     = true
    support_credentials = true
    origin_groups = [
      {
        name    = "Trusted Origins"
        origins = ["https://app.example.com", "https://www.example.com"]
      }
    ]
  }
}

# 3. Rate limiting
resource "anypoint_api_policy_rate_limiting" "rate_limit" {
  order = 3
  # ... configuration
}
```

## DataWeave Expressions

Many policies support DataWeave expressions for dynamic configuration:

### Common Expressions

```dw
# Get header value
#[attributes.headers['header-name']]

# Get query parameter
#[attributes.queryParams['param-name']]

# Get client IP from X-Forwarded-For
#[attributes.headers['x-forwarded-for']]

# Generate UUID
#[uuid()]

# Current timestamp
#[now()]

# HTTP method
#[attributes.method]

# Request path
#[attributes.requestPath]

# Full query string
#[attributes.queryString]

# Conditional logic
#[if(attributes.method == 'POST') 'write' else 'read']
```

## Best Practices

1. **Order Matters** - Place security policies first, caching last
2. **Use Disabled Flag** - Keep policies configured but disabled for easy testing
3. **Start Conservative** - Begin with strict limits, loosen based on usage
4. **Enable Headers** - Expose rate limit headers for client awareness
5. **Monitor Rejections** - Track policy rejections in API analytics
6. **Version Policies** - Use policy labels to identify versions
7. **Test Thoroughly** - Validate policy behavior before production
8. **Document Changes** - Keep policy configuration changes in version control
9. **Use Expressions Wisely** - Test DataWeave expressions before deployment
10. **Layer Security** - Combine multiple security policies for defense in depth

## Disabling Policies

To disable a policy without removing it:

```hcl
resource "anypoint_api_policy_jwt_validation" "jwt_validation" {
  disabled = true  # Policy configured but not enforced
  # ... configuration
}
```

## Troubleshooting

### Policy Not Applied

**Check:**
1. Policy order conflicts
2. API instance ID is correct
3. Policy configuration is valid
4. Gateway supports the policy

### Policy Rejecting Valid Requests

**Common Issues:**
- Incorrect DataWeave expressions
- Wrong header/parameter names
- Missing required fields
- Mismatched data types

**Debug:**
- Enable message logging policy
- Check gateway logs
- Test expressions in DataWeave playground
- Verify API contract matches policy expectations

### Rate Limit Not Working

**Verify:**
- `clusterizable` is true for clustered environments
- Key selector expression returns consistent values
- Time period is appropriate for traffic pattern
- Headers are being exposed for client visibility

## Additional Resources

- [API Manager Policies Documentation](https://docs.mulesoft.com/api-manager/2.x/policies)
- [Policy Configuration Reference](https://docs.mulesoft.com/api-manager/2.x/policy-mule4-available-policies)
- [DataWeave Language Guide](https://docs.mulesoft.com/dataweave/2.4/)
- [Rate Limiting Best Practices](https://docs.mulesoft.com/api-manager/2.x/rate-limiting-and-throttling)

## Cleanup

To remove all policies:

```bash
terraform destroy
```

**Note:** Policies are removed in reverse order of application.
