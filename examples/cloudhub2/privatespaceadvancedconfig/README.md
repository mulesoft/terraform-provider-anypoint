# Private Space Advanced Configuration Example

This example demonstrates how to configure advanced settings for an Anypoint Private Space using the `anypoint_privatespace_advanced_config` resource.

## What This Example Does

- Configures ingress settings for a private space including:
  - Read response timeout
  - Protocol settings
  - Log filters and port log level
  - Deployment configuration
- Enables or disables IAM role for the private space

## Prerequisites

- An existing Anypoint Private Space
- Valid Anypoint Platform credentials with appropriate permissions

## Usage

1. Set your Anypoint Platform credentials:
   ```bash
   export TF_VAR_anypoint_client_id="your-client-id"
   export TF_VAR_anypoint_client_secret="your-client-secret"
   ```

2. Set the private space ID:
   ```bash
   export TF_VAR_private_space_id="your-private-space-id"
   ```

3. Optional: Set a custom base URL (defaults to production):
   ```bash
   export TF_VAR_anypoint_base_url="https://stgx.anypoint.mulesoft.com"
   ```

4. Initialize and apply:
   ```bash   
   terraform plan
   terraform apply
   ```

## Configuration Options

### Organization ID

- **organization_id**: Optional organization ID where the private space is located. If not specified, uses the organization from provider credentials. This enables multi-organization management scenarios.

### Ingress Configuration

- **read_response_timeout**: Timeout in seconds for reading responses (default: "300")
- **protocol**: Protocol to use - typically "https-redirect" (default: "https-redirect")
- **logs**: Log configuration including:
  - **port_log_level**: Log level for port logs (default: "ERROR")
  - **filters**: Array of log filters with IP and level
- **deployment**: Deployment configuration including:
  - **status**: Deployment status (default: "APPLIED")
  - **last_seen_timestamp**: Timestamp of last deployment

### IAM Role

- **enable_iam_role**: Boolean to enable/disable IAM role for the private space (default: false)

## Default Values

If not specified, the resource uses these default values:
- Read response timeout: "300"
- Protocol: "https-redirect"
- Port log level: "ERROR"
- Log filters: empty array
- Deployment status: "APPLIED"
- Last seen timestamp: 1753719215000
- Enable IAM role: false

## Resource Management

- **Create**: Applies the advanced configuration to the private space
- **Update**: Updates the configuration with new values
- **Delete**: Resets the configuration to default values
- **Import**: Can be imported using the private space ID

## Notes

- This resource manages only the advanced configuration aspects of a private space
- The private space itself must exist before applying this configuration
- Changes to ingress configuration may affect traffic to applications in the private space
- The resource uses PATCH operations to update only the specified configuration fields