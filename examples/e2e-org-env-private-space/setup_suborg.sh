#!/bin/bash
###############################################################################
# Sub-Organization Setup Script
# Automates the setup of environment variables and terraform initialization
###############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Sub-Organization Setup Assistant${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check if terraform is installed
if ! command -v terraform &> /dev/null; then
    echo -e "${RED}Error: terraform is not installed${NC}"
    echo "Please install terraform from: https://www.terraform.io/downloads.html"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "suborg_with_privatespace_complete.tf" ]; then
    echo -e "${RED}Error: Please run this script from the examples/comprehensive-e2e directory${NC}"
    exit 1
fi

echo -e "${YELLOW}Step 1: Configuration File Setup${NC}"
echo "Checking for terraform.tfvars..."

if [ -f "terraform.tfvars" ]; then
    echo -e "${GREEN}✓ terraform.tfvars exists${NC}"
    read -p "Do you want to overwrite it? (y/N): " overwrite
    if [[ $overwrite =~ ^[Yy]$ ]]; then
        cp suborg_with_privatespace.tfvars.example terraform.tfvars
        echo -e "${GREEN}✓ Created terraform.tfvars from example${NC}"
        echo -e "${YELLOW}⚠ Please edit terraform.tfvars with your values${NC}"
    fi
else
    cp suborg_with_privatespace.tfvars.example terraform.tfvars
    echo -e "${GREEN}✓ Created terraform.tfvars from example${NC}"
    echo -e "${YELLOW}⚠ Please edit terraform.tfvars with your values${NC}"
fi

echo ""
echo -e "${YELLOW}Step 2: Environment Variables Setup${NC}"
echo "Setting up authentication environment variables..."
echo ""

# Connected App Credentials
echo "Connected App Authentication (for resource management):"
read -p "Client ID (default: e5a776d9862a4f2d8f61ba8450803908): " client_id
client_id=${client_id:-e5a776d9862a4f2d8f61ba8450803908}

read -s -p "Client Secret: " client_secret
echo ""

if [ -z "$client_secret" ]; then
    echo -e "${RED}Error: Client Secret is required${NC}"
    exit 1
fi

export ANYPOINT_CLIENT_ID="$client_id"
export ANYPOINT_CLIENT_SECRET="$client_secret"

echo -e "${GREEN}✓ Connected App credentials set${NC}"
echo ""

# User Authentication (for scope assignment)
echo "User Authentication (for connected app scope assignment):"
read -p "Admin Username: " admin_username
read -s -p "Admin Password: " admin_password
echo ""

if [ -z "$admin_username" ] || [ -z "$admin_password" ]; then
    echo -e "${YELLOW}⚠ User credentials not provided${NC}"
    echo -e "${YELLOW}  You'll need to set these manually for scope assignment:${NC}"
    echo -e "${YELLOW}  export ANYPOINT_ADMIN_USERNAME='your-username'${NC}"
    echo -e "${YELLOW}  export ANYPOINT_ADMIN_PASSWORD='your-password'${NC}"
else
    export ANYPOINT_ADMIN_USERNAME="$admin_username"
    export ANYPOINT_ADMIN_PASSWORD="$admin_password"
    echo -e "${GREEN}✓ User credentials set${NC}"
fi

echo ""
echo -e "${YELLOW}Step 3: Terraform Initialization${NC}"
echo "Initializing Terraform..."

if terraform init; then
    echo -e "${GREEN}✓ Terraform initialized successfully${NC}"
else
    echo -e "${RED}✗ Terraform initialization failed${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}Step 4: Validation${NC}"
echo "Validating Terraform configuration..."

if terraform validate; then
    echo -e "${GREEN}✓ Configuration is valid${NC}"
else
    echo -e "${RED}✗ Configuration validation failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Setup Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Environment variables have been set for this session:"
echo "  ANYPOINT_CLIENT_ID=$ANYPOINT_CLIENT_ID"
echo "  ANYPOINT_CLIENT_SECRET=***"
if [ ! -z "$admin_username" ]; then
    echo "  ANYPOINT_ADMIN_USERNAME=$admin_username"
    echo "  ANYPOINT_ADMIN_PASSWORD=***"
fi
echo ""
echo "Next steps:"
echo "  1. Edit terraform.tfvars with your specific values"
echo "  2. Review the plan: terraform plan -var-file=terraform.tfvars"
echo "  3. Apply changes:   terraform apply -var-file=terraform.tfvars"
echo ""
echo "For more information, see SUBORG_WITH_PRIVATESPACE_GUIDE.md"
echo ""

# Create a .env file for future reference (without secrets)
cat > .env.example << EOF
# Anypoint Platform Authentication
export ANYPOINT_CLIENT_ID="$client_id"
export ANYPOINT_CLIENT_SECRET="your-client-secret-here"

# User Authentication (for scope assignment)
export ANYPOINT_ADMIN_USERNAME="your-admin-username"
export ANYPOINT_ADMIN_PASSWORD="your-admin-password"
EOF

echo -e "${GREEN}✓ Created .env.example for future reference${NC}"
echo ""

# Offer to run terraform plan
read -p "Would you like to run 'terraform plan' now? (y/N): " run_plan
if [[ $run_plan =~ ^[Yy]$ ]]; then
    echo ""
    echo "Running terraform plan..."
    terraform plan -var-file=terraform.tfvars
fi
