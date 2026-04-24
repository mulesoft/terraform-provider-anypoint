#!/bin/bash

# Example of using environment variables for admin user authentication
# This approach is recommended for CI/CD pipelines and automation
# Note: All variables have defaults in variables.tf, so you can override only what's needed

# Set your admin user credentials (required for connected_app_scopes)
export ANYPOINT_USERNAME="ankitsarda_anypointstgx"
export ANYPOINT_PASSWORD="Dreamz@007"

# Set your admin connected app credentials
export TF_VAR_anypoint_admin_client_id="a66da37ba83d4c599264347952d4d533"
export TF_VAR_anypoint_admin_client_secret="0de4EA9E5bae4651B599a2071bFDD4E1"
export TF_VAR_anypoint_admin_username="ankitsarda_anypointstgx"
export TF_VAR_anypoint_admin_password="Dreamz@007"

# Set your target configuration
export TF_VAR_connected_app_id="e5a776d9862a4f2d8f61ba8450803908"
export TF_VAR_target_organization_id="542cc7e3-2143-40ce-90e9-cf69da9b4da6"

# Set base URL (staging environment)
export TF_VAR_anypoint_base_url="https://stgx.anypoint.mulesoft.com"

echo "Environment variables set for connected app scopes management"
echo "You can now run: terraform plan"

# Example usage:
# source env_example.sh
# terraform plan
# terraform apply