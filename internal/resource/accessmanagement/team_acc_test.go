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

func TestAccTeamResource_basic(t *testing.T) {
	resourceName := "anypoint_team.test"
	teamName := "terraform-test-team-basic"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamResource_basic(teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "team_name", teamName),
					resource.TestCheckResourceAttr(resourceName, "team_type", "internal"),
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

func TestAccTeamResource_complete(t *testing.T) {
	resourceName := "anypoint_team.test"
	teamName := "terraform-test-team-complete"
	parentTeamName := "terraform-test-parent-team"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamDestroy,
		Steps: []resource.TestStep{
			// Create parent team first
			{
				Config: testAccTeamResource_withParent(parentTeamName, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "team_name", teamName),
					resource.TestCheckResourceAttr(resourceName, "team_type", "internal"),
					resource.TestCheckResourceAttrSet(resourceName, "parent_team_id"),
				),
			},
		},
	})
}

func TestAccTeamResource_update(t *testing.T) {
	resourceName := "anypoint_team.test"
	teamName := "terraform-test-team-update"
	teamNameUpdated := "terraform-test-team-update-modified"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamResource_basic(teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "team_name", teamName),
				),
			},
			// Update and Read testing
			{
				Config: testAccTeamResource_basic(teamNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "team_name", teamNameUpdated),
				),
			},
		},
	})
}

func TestAccTeamResource_invalidType(t *testing.T) {
	teamName := "terraform-test-team-invalid"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTeamResource_invalidType(teamName),
				ExpectError: regexp.MustCompile(`(invalid|unsupported)`),
			},
		},
	})
}

func TestAccTeamResource_disappears(t *testing.T) {
	resourceName := "anypoint_team.test"
	teamName := "terraform-test-team-disappears"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamResource_basic(teamName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					testAccCheckTeamResourceDisappears(resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckTeamDestroy(s *terraform.State) error {
	anypointClient := acctest.CreateTestClient(nil) // This will use env vars
	teamClient := &accessmanagement.TeamClient{
		AnypointClient: anypointClient,
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "anypoint_team" {
			continue
		}

		_, err := teamClient.GetTeam(context.Background(), rs.Primary.Attributes["organization_id"], rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("team %s still exists", rs.Primary.ID)
		}

		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking team %s: %v", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckTeamExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		anypointClient := acctest.CreateTestClient(nil) // This will use env vars
		teamClient := &accessmanagement.TeamClient{
			AnypointClient: anypointClient,
		}

		_, err := teamClient.GetTeam(context.Background(), rs.Primary.Attributes["organization_id"], rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error reading team %s: %v", rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccCheckTeamResourceDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		anypointClient := acctest.CreateTestClient(nil) // This will use env vars
		teamClient := &accessmanagement.TeamClient{
			AnypointClient: anypointClient,
		}

		return teamClient.DeleteTeam(context.Background(), rs.Primary.Attributes["organization_id"], rs.Primary.ID)
	}
}

// Configuration templates
func testAccTeamResource_basic(name string) string {
	return fmt.Sprintf(`
resource "anypoint_team" "test" {
  team_name = %[1]q
  team_type = "internal"
}
`, name)
}

func testAccTeamResource_withParent(parentName, childName string) string {
	return fmt.Sprintf(`
resource "anypoint_team" "parent" {
  team_name = %[1]q
  team_type = "internal"
}

resource "anypoint_team" "test" {
  team_name      = %[2]q
  team_type      = "internal"
  parent_team_id = anypoint_team.parent.id
}
`, parentName, childName)
}

func testAccTeamResource_invalidType(name string) string {
	return fmt.Sprintf(`
resource "anypoint_team" "test" {
  team_name = %[1]q
  team_type = "invalid_type"
}
`, name)
}
