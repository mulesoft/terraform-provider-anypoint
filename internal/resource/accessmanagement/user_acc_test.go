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

func TestAccUserResource_basic(t *testing.T) {
	resourceName := "anypoint_user.test"
	username := "terraform-test-user-basic"
	email := "terraform-test-basic@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUserResource_basic(username, email),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "first_name", "Terraform"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "Test"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"}, // Password is sensitive and not returned
			},
		},
	})
}

func TestAccUserResource_complete(t *testing.T) {
	resourceName := "anypoint_user.test"
	username := "terraform-test-user-complete"
	email := "terraform-test-complete@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUserResource_complete(username, email),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "first_name", "Complete"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "User"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccUserResource_update(t *testing.T) {
	resourceName := "anypoint_user.test"
	username := "terraform-test-user-update"
	email := "terraform-test-update@example.com"
	emailUpdated := "terraform-test-update-modified@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUserResource_basic(username, email),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", email),
				),
			},
			// Update and Read testing
			{
				Config: testAccUserResource_basic(username, emailUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckUserExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", emailUpdated),
				),
			},
		},
	})
}

func TestAccUserResource_invalidEmail(t *testing.T) {
	username := "terraform-test-user-invalid"
	invalidEmail := "invalid-email"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccUserResource_basic(username, invalidEmail),
				ExpectError: regexp.MustCompile(`(invalid.*email|email.*invalid)`),
			},
		},
	})
}

func TestAccUserResource_duplicateUsername(t *testing.T) {
	username := "terraform-test-user-duplicate"
	email1 := "terraform-test-dup1@example.com"
	email2 := "terraform-test-dup2@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			// Create first user
			{
				Config: testAccUserResource_basic(username, email1),
			},
			// Attempt to create second user with same username
			{
				Config:      testAccUserResource_duplicate(username, email1, email2),
				ExpectError: regexp.MustCompile(`(duplicate|already exists|conflict)`),
			},
		},
	})
}

func TestAccUserResource_disappears(t *testing.T) {
	resourceName := "anypoint_user.test"
	username := "terraform-test-user-disappears"
	email := "terraform-test-disappears@example.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResource_basic(username, email),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists(resourceName),
					testAccCheckUserResourceDisappears(resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckUserDestroy(s *terraform.State) error {
	userAnypointClient := acctest.CreateUserTestClient(nil)
	userClient := &accessmanagement.UserClient{
		UserAnypointClient: userAnypointClient,
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "anypoint_user" {
			continue
		}

		_, err := userClient.GetUser(ctx, rs.Primary.Attributes["organization_id"], rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("user %s still exists", rs.Primary.ID)
		}

		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking user %s: %v", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckUserExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		userAnypointClient := acctest.CreateUserTestClient(nil)
		userClient := &accessmanagement.UserClient{
			UserAnypointClient: userAnypointClient,
		}

		_, err := userClient.GetUser(context.Background(), rs.Primary.Attributes["organization_id"], rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error reading user %s: %v", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckUserResourceDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		userAnypointClient := acctest.CreateUserTestClient(nil)
		userClient := &accessmanagement.UserClient{
			UserAnypointClient: userAnypointClient,
		}

		return userClient.DeleteUser(context.Background(), rs.Primary.Attributes["organization_id"], rs.Primary.ID)
	}
}

// Configuration templates
func testAccUserResource_basic(username, email string) string {
	return fmt.Sprintf(`
resource "anypoint_user" "test" {
  username   = %[1]q
  email      = %[2]q
  first_name = "Terraform"
  last_name  = "Test"
  password   = "TerraformTest123!"
}
`, username, email)
}

func testAccUserResource_complete(username, email string) string {
	return fmt.Sprintf(`
resource "anypoint_user" "test" {
  username   = %[1]q
  email      = %[2]q
  first_name = "Complete"
  last_name  = "User"
  password   = "CompleteTest123!"
}
`, username, email)
}

func testAccUserResource_duplicate(username, email1, email2 string) string {
	return fmt.Sprintf(`
resource "anypoint_user" "first" {
  username   = %[1]q
  email      = %[2]q
  first_name = "First"
  last_name  = "User"
  password   = "FirstTest123!"
}

resource "anypoint_user" "second" {
  username   = %[1]q  # Same username - should cause conflict
  email      = %[3]q
  first_name = "Second"
  last_name  = "User"
  password   = "SecondTest123!"
}
`, username, email1, email2)
}
