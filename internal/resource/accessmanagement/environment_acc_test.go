package accessmanagement_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mulesoft/terraform-provider-anypoint/internal/acctest"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

func TestAccEnvironmentResource_basic(t *testing.T) {
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-basic"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResource_basic(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", envName),
					resource.TestCheckResourceAttr(resourceName, "type", "sandbox"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEnvironmentResource_complete(t *testing.T) {
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-complete"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResource_complete(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", envName),
					resource.TestCheckResourceAttr(resourceName, "type", "production"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
				),
			},
		},
	})
}

func TestAccEnvironmentResource_update(t *testing.T) {
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-update"
	envNameUpdated := "terraform-test-env-update-modified"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResource_basic(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", envName),
					resource.TestCheckResourceAttr(resourceName, "type", "sandbox"),
				),
			},
			// Update and Read testing
			{
				Config: testAccEnvironmentResource_basic(envNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", envNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "type", "sandbox"),
				),
			},
		},
	})
}

func TestAccEnvironmentResource_withOrganizationID(t *testing.T) {
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-org"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResource_withOrganizationID(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", envName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
				),
			},
		},
	})
}

func TestAccEnvironmentResource_invalidType(t *testing.T) {
	envName := "terraform-test-env-invalid"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccEnvironmentResource_invalidType(envName),
				ExpectError: regexp.MustCompile(`(invalid|unsupported)`),
			},
		},
	})
}

func TestAccEnvironmentResource_disappears(t *testing.T) {
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-disappears"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResource_basic(envName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(resourceName),
					testAccCheckResourceDisappears(resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckEnvironmentDestroy(s *terraform.State) error {
	userClient := acctest.CreateUserTestClient(nil)
	envClient := &accessmanagement.EnvironmentClient{
		UserAnypointClient: userClient,
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "anypoint_environment" {
			continue
		}

		_, err := envClient.GetEnvironment(ctx, rs.Primary.Attributes["organization_id"], rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("environment %s still exists", rs.Primary.ID)
		}

		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking environment %s: %v", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckEnvironmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		userClient := acctest.CreateUserTestClient(nil)
		envClient := &accessmanagement.EnvironmentClient{
			UserAnypointClient: userClient,
		}

		_, err := envClient.GetEnvironment(context.Background(), rs.Primary.Attributes["organization_id"], rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error reading environment %s: %v", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckResourceDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		userClient := acctest.CreateUserTestClient(nil)
		envClient := &accessmanagement.EnvironmentClient{
			UserAnypointClient: userClient,
		}

		return envClient.DeleteEnvironment(context.Background(), rs.Primary.Attributes["organization_id"], rs.Primary.ID)
	}
}

// Configuration templates
func testAccEnvironmentResource_basic(name string) string {
	return fmt.Sprintf(`
resource "anypoint_environment" "test" {
  name = %[1]q
  type = "sandbox"
}
`, name)
}

func testAccEnvironmentResource_complete(name string) string {
	return fmt.Sprintf(`
resource "anypoint_environment" "test" {
  name = %[1]q
  type = "production"
}
`, name)
}

func testAccEnvironmentResource_withOrganizationID(name string) string {
	return fmt.Sprintf(`
resource "anypoint_environment" "test" {
  name            = %[1]q
  type            = "sandbox"
  organization_id = "test-org-id"
}
`, name)
}

func testAccEnvironmentResource_invalidType(name string) string {
	return fmt.Sprintf(`
resource "anypoint_environment" "test" {
  name = %[1]q
  type = "invalid_type"
}
`, name)
}