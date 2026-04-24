# Private Space Upgrade Resource and Data Source

This example demonstrates how to use the `anypoint_private_space_upgrade` resource and data source to manage and monitor upgrades for CloudHub 2.0 private spaces.

## Resource Configuration

The `anypoint_private_space_upgrade` resource schedules an upgrade for a private space using the Anypoint Platform API.

### Required Arguments

- `private_space_id` - The ID of the private space to upgrade
- `date` - The date when the upgrade should be scheduled (format: YYYY-MM-DD)
- `opt_in` - Whether to opt in to the upgrade (boolean)

### Optional Arguments

- `organization_id` - The ID of the target organization (defaults to provider's organization)

### Computed Attributes

- `id` - The unique identifier for the upgrade operation
- `scheduled_update_time` - The scheduled update time returned by the API
- `status` - The status of the upgrade operation (e.g., "QUEUED")

## Data Source Configuration

The `anypoint_private_space_upgrade` data source retrieves the current upgrade status for a private space.

### Required Arguments

- `private_space_id` - The ID of the private space to get upgrade status for

### Optional Arguments

- `organization_id` - The ID of the target organization (defaults to provider's organization)

### Computed Attributes

- `id` - Identifier for this data source (same as private_space_id)
- `scheduled_update_time` - The scheduled update time for the upgrade
- `status` - The current status of the upgrade (e.g., QUEUED, IN_PROGRESS, COMPLETED)

## Usage Examples

### Resource Usage
```hcl
resource "anypoint_private_space_upgrade" "example" {
  private_space_id = "your-private-space-id"
  organization_id  = "your-organization-id"  # Optional: specify target organization
  date             = "2025-08-12"
  opt_in           = true
}
```

### Data Source Usage
```hcl
data "anypoint_private_space_upgrade" "current_status" {
  private_space_id = "your-private-space-id"
  organization_id  = "your-organization-id"  # Optional: specify target organization
}
```

### Combined Usage
```hcl
# Schedule an upgrade
resource "anypoint_private_space_upgrade" "example" {
  private_space_id = "your-private-space-id"
  organization_id  = "your-organization-id"  # Optional: specify target organization
  date             = "2025-08-12"
  opt_in           = true
}

# Check the upgrade status
data "anypoint_private_space_upgrade" "status_check" {
  private_space_id = "your-private-space-id"
  organization_id  = "your-organization-id"  # Optional: specify target organization
  depends_on       = [anypoint_private_space_upgrade.example]
}
```

## API Details

### Resource API
- **Method**: PATCH
- **URL**: `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/upgrade`
- **Query Parameters**: `?date=2025-08-12&optIn=true`

### Data Source API
- **Method**: GET
- **URL**: `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/upgradestatus`

### Delete API
- **Method**: DELETE
- **URL**: `https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/upgrade`

## Important Notes

1. **One-time Operation**: Private space upgrades are one-time operations. Any changes to the resource will force a replacement.

2. **Date Format**: The date must be in YYYY-MM-DD format.

3. **Status Monitoring**: Use the data source to monitor the upgrade progress after scheduling with the resource.

4. **Upgrade Cancellation**: Scheduled upgrades can be cancelled/deleted using the DELETE operation. This will remove the scheduled upgrade from the system.

5. **Import Format**: To import an existing upgrade resource, use the format: `private_space_id:date:opt_in`
   ```
   terraform import anypoint_private_space_upgrade.example my-space-id:2025-08-12:true
   ```

## Upgrade Lifecycle Management

### Scheduling an Upgrade
```bash
terraform apply
```

### Cancelling a Scheduled Upgrade
To cancel a scheduled upgrade, you can destroy the specific resource:
```bash
# Cancel a specific upgrade
terraform destroy -target=anypoint_private_space_upgrade.example

# Or remove the resource from your configuration and apply
terraform apply
```

### Preventing Accidental Cancellation
To prevent accidental cancellation of critical upgrades:
```hcl
resource "anypoint_private_space_upgrade" "critical_upgrade" {
  private_space_id = "your-private-space-id"
  date             = "2025-08-12"
  opt_in           = true

  lifecycle {
    prevent_destroy = true
  }
}
```

### Monitoring Upgrade Status
Use the data source to check upgrade status without managing the upgrade itself:
```hcl
data "anypoint_private_space_upgrade" "monitor_only" {
  private_space_id = "your-private-space-id"
}

output "current_status" {
  value = data.anypoint_private_space_upgrade.monitor_only.status
}
```

## Use Cases

1. **Schedule and Monitor**: Use the resource to schedule an upgrade and the data source to monitor its progress.

2. **Status Check Only**: Use just the data source to check the status of an upgrade that was scheduled outside of Terraform.

3. **Conditional Logic**: Use the data source in conditional expressions to make decisions based on upgrade status.

4. **Emergency Cancellation**: Quickly cancel a problematic upgrade by destroying the resource.

5. **Automated Upgrade Management**: Integrate with CI/CD pipelines to schedule upgrades during maintenance windows.

## Common Workflows

### Basic Upgrade Workflow
1. **Schedule**: Create the `anypoint_private_space_upgrade` resource
2. **Monitor**: Use the data source to check status periodically
3. **Complete**: The upgrade completes automatically on the scheduled date
4. **Cleanup**: Optionally destroy the resource after completion

### Emergency Cancellation Workflow
1. **Identify Issue**: Determine that an upgrade needs to be cancelled
2. **Cancel**: Run `terraform destroy -target=anypoint_private_space_upgrade.resource_name`
3. **Verify**: Check that the upgrade is no longer scheduled
4. **Reschedule**: Create a new resource for a future date if needed

### Status Monitoring Workflow
1. **Check Status**: Use data source to get current upgrade status
2. **Conditional Actions**: Make decisions based on status
3. **Alerting**: Set up monitoring based on status changes

## Running the Example

1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Update the variables with your actual values
3. Run the following commands:

```bash
terraform plan
terraform apply
```

## Example Output

After applying, you'll get information from both the resource and data source:

### Resource Output
```hcl
upgrade_resource_details = {
  id                    = "my-space-id-1234567890"
  private_space_id      = "my-space-id"
  date                  = "2025-08-12"
  opt_in                = true
  scheduled_update_time = "2024-02-28T17:57:36.000+00:00"
  status                = "QUEUED"
}
```

### Data Source Output
```hcl
upgrade_data_source_status = {
  id                    = "my-space-id"
  private_space_id      = "my-space-id"
  scheduled_update_time = "2024-02-28T17:57:36.000+00:00"
  status                = "QUEUED"
}
```

## Files in this Example

- `main.tf` - Complete example with both resource and data source
- `datasource.tf` - Standalone data source example  
- `variables.tf` - Variable definitions
- `terraform.tfvars.example` - Example variable values
- `README.md` - This documentation