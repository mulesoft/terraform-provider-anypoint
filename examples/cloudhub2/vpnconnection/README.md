# Anypoint Platform - CloudHub 2.0 VPN Connection

This example demonstrates how to use the Anypoint Platform Terraform provider to create a secure, site-to-site VPN connection between your on-premises data center and a CloudHub 2.0 private space.

This setup is ideal for hybrid cloud deployments where you need to establish a secure and reliable connection between your private network and the Anypoint Platform.

## Architecture

The following diagram illustrates the architecture of the resources created by this example:

![Architecture Diagram](https://mermaid.ink/svg/pako:eNqNVMtqwzAQ_JWRZxVqfBDRSjsohUKhlNLHK2dssbZkzCVpEP_eGTeNpIc9LNjdmWd3Zk-MhXoQJ5mH-Yx0C_T-8DSpW_X61cO0bK78x8AAG0XkQYgXGZJ-eJ-t_8A90v_uT1F_20K78e63t18f5-YgJ1hJcQpWz33c5w07hVl2qg6D7bN-27_QkF-oWp4e3Q80g7aN-Kq4QGvCg3S9x_Kz5Pew843j9oP0QWJ1Cq9r2wzFw0JqH6RjNEl8CjG92w_5KjF28kF50u2_cR6qFh-R-x3K8N6-mR3R8Q_2361T9wD56k5z6fJ95aWq9R2Q8g577a79hN0fJgB1Fq24T9pG6FpA-A_8M-S9j)

## How it Works

The Terraform configuration in this example performs the following steps:

1.  **Creates a Private Space:** A private space is an isolated network environment within CloudHub 2.0 where you can deploy your applications.
2.  **Configures a Private Network:** A private network is created within the private space with a specified CIDR block.
3.  **Introduces a Delay:** A 10-second delay is introduced to ensure the private network is fully initialized before proceeding.
4.  **Creates a VPN Connection:** A VPN connection is established between your on-premises network and the private space, using the details you provide.

## Organization ID Support

This example supports multi-organization management through the optional `organization_id` parameter:

- **Default behavior**: Uses the organization from your provider credentials
- **Multi-org scenarios**: Specify `organization_id` to create VPN connections in a different organization
- **Cross-org access**: Requires appropriate permissions in the target organization

## Prerequisites

Before running this example, you will need:

-   An Anypoint Platform account with the necessary permissions to create private spaces and VPN connections.
-   Your Anypoint Platform client ID and client secret.
-   The network details for your on-premises VPN endpoint, including the remote IP address and ASN.
-   (Optional) Target organization ID if managing resources across multiple organizations.

## How to Run the Demo

1.  **Configure your credentials:**
    Create a `terraform.tfvars` file and add your Anypoint Platform credentials and network details:
    ```tfvars
    anypoint_client_id     = "YOUR_CLIENT_ID"
    anypoint_client_secret = "YOUR_CLIENT_SECRET"
    organization_id        = "YOUR_ORGANIZATION_ID"  # Optional: specify target organization
    region_id              = "us-east-1"
    cidr_block             = "10.0.0.0/22"
    connection_name        = "my-vpn-connection"
    local_asn              = "64512"
    remote_asn             = "65001"
    remote_ip_address      = "YOUR_REMOTE_IP_ADDRESS"
    psk_1                  = "YOUR_PRE_SHARED_KEY_1"
    ptp_cidr_1             = "169.254.1.0/30"
    psk_2                  = "YOUR_PRE_SHARED_KEY_2"
    ptp_cidr_2             = "169.254.2.0/30"
    startup_action         = "start"
    ```

2.  **Apply the configuration:**
    ```sh
    terraform apply
    ```
    This will provision the private space, private network, and VPN connection in your Anypoint Platform account.

3.  **Review the outputs:**
    Terraform will display the details of the created resources as outputs, including the private space ID, network details, and VPN connection status.

## Cleanup

To tear down the resources created by this demo, run:

```sh
terraform destroy
``` 