package accessmanagement_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/acctest"
)

func TestAccEnvironmentDataSource_basic(t *testing.T) {
	dataSourceName := "data.anypoint_environment.test"
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-ds-basic"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSource_basic(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match resource
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "organization_id", resourceName, "organization_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "created_at", resourceName, "created_at"),
					resource.TestCheckResourceAttrPair(dataSourceName, "updated_at", resourceName, "updated_at"),
					// Verify data source specific attributes
					resource.TestCheckResourceAttr(dataSourceName, "name", envName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "sandbox"),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "organization_id"),
				),
			},
		},
	})
}

func TestAccEnvironmentDataSource_byID(t *testing.T) {
	dataSourceName := "data.anypoint_environment.test"
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-ds-by-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSource_byID(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify data source attributes match resource
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "organization_id", resourceName, "organization_id"),
					// Verify specific values
					resource.TestCheckResourceAttr(dataSourceName, "name", envName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "production"),
				),
			},
		},
	})
}

func TestAccEnvironmentDataSource_nonExistent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccEnvironmentDataSource_nonExistent(),
				ExpectError: regexp.MustCompile(`(not found|does not exist)`),
			},
		},
	})
}

func TestAccEnvironmentDataSource_withOrganizationID(t *testing.T) {
	dataSourceName := "data.anypoint_environment.test"
	resourceName := "anypoint_environment.test"
	envName := "terraform-test-env-ds-org"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSource_withOrganizationID(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "organization_id", resourceName, "organization_id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", envName),
				),
			},
		},
	})
}

// Configuration templates
func testAccEnvironmentDataSource_basic(name string) string {
	return fmt.Sprintf(`
resource "anypoint_environment" "test" {
  name = %[1]q
  type = "sandbox"
}

data "anypoint_environment" "test" {
  name = anypoint_environment.test.name
}
`, name)
}

func testAccEnvironmentDataSource_byID(name string) string {
	return fmt.Sprintf(`
resource "anypoint_environment" "test" {
  name = %[1]q
  type = "production"
}

data "anypoint_environment" "test" {
  id = anypoint_environment.test.id
}
`, name)
}

func testAccEnvironmentDataSource_nonExistent() string {
	return `
data "anypoint_environment" "test" {
  name = "non-existent-environment-name-12345"
}
`
}

func testAccEnvironmentDataSource_withOrganizationID(name string) string {
	return fmt.Sprintf(`
resource "anypoint_environment" "test" {
  name = %[1]q
  type = "sandbox"
}

data "anypoint_environment" "test" {
  name            = anypoint_environment.test.name
  organization_id = anypoint_environment.test.organization_id
}
`, name)
}