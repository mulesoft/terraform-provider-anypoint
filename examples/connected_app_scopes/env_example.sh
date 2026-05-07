#!/bin/bash

# Example of using environment variables for admin user authentication
# This approach is recommended for CI/CD pipelines and automation
# Note: All variables have defaults in variables.tf, so you can override only what's needed

# Set your admin user credentials (required for connected_app_scopes)
export ANYPOINT_USERNAME="<admin_username_here>"
export ANYPOINT_PASSWORD="<admin_password_here>"

# Set your admin connected app credentials
export TF_VAR_anypoint_admin_client_id="<anypoint_admin_client_id>"
export TF_VAR_anypoint_admin_client_secret="<anypoint_admin_client_secret>"
export TF_VAR_anypoint_admin_username="<admin_username_here>"
export TF_VAR_anypoint_admin_password="<admin_password_here>"

# Set your target configuration
export TF_VAR_connected_app_id="<anypoint_connected_app_client_id>"
export TF_VAR_target_organization_id="<org_id>"

# Set base URL (staging environment)
export TF_VAR_anypoint_base_url="https://stgx.anypoint.mulesoft.com"

echo "Environment variables set for connected app scopes management"
echo "You can now run: terraform plan"

# Example usage:
# source env_example.sh
# terraform plan
# terraform apply