package exchange

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

// api_version, classifier and main_file are all read straight off the asset payload —
// api_version from the attributes list, the other two from the uploaded file — so an
// asset inspected through the data source must report them, not just a managed one.
func TestAssetDataSource_ExposesReadableFileMetadata(t *testing.T) {
	resp := &datasource.SchemaResponse{}
	NewAssetDataSource().Schema(context.Background(), datasource.SchemaRequest{}, resp)

	for _, name := range []string{"api_version", "classifier", "main_file"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("%s is readable from the asset payload but missing from the data source", name)
		}
	}
}

// Guard the shared readers, since both the resource and the data source now depend on
// them landing on the user's file rather than one of Exchange's derived ones.
func TestExtractFileMetadata_PicksTheUploadedFile(t *testing.T) {
	files := []exchange.AssetFile{
		{Classifier: "", Packaging: "pom"},                           // always present
		{Classifier: "fat-oas", Packaging: "zip", IsGenerated: true}, // derived
		{Classifier: "oas", Packaging: "json", MainFile: "petstore.json"},
	}

	classifier, mainFile := exchange.ExtractFileMetadata(files)
	if classifier != "oas" || mainFile != "petstore.json" {
		t.Errorf("got (%q, %q), want (oas, petstore.json)", classifier, mainFile)
	}
}

func TestExtractFileMetadata_MetadataOnlyAssetHasNoFile(t *testing.T) {
	classifier, mainFile := exchange.ExtractFileMetadata([]exchange.AssetFile{
		{Classifier: "", Packaging: "pom"},
	})
	if classifier != "" || mainFile != "" {
		t.Errorf("got (%q, %q), want empty for an asset with no uploaded file", classifier, mainFile)
	}
}

func TestExtractAttributeValue(t *testing.T) {
	attrs := []interface{}{
		map[string]interface{}{"key": "unrelated", "value": "x"},
		map[string]interface{}{"key": "api-version", "value": "v2"},
	}
	if got := exchange.ExtractAttributeValue(attrs, "api-version"); got != "v2" {
		t.Errorf("api-version = %q, want v2", got)
	}
	if got := exchange.ExtractAttributeValue(attrs, "absent"); got != "" {
		t.Errorf("missing key should yield empty, got %q", got)
	}
}
