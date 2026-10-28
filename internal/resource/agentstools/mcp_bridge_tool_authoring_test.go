package agentstools

import (
	"encoding/json"
	"reflect"
	"testing"
)

func mappingPairs(pairs ...bridgeParamMapping) *[]bridgeParamMapping {
	out := append([]bridgeParamMapping(nil), pairs...)
	return &out
}

func strRef(s string) *string { return &s }

// --- input_schema -------------------------------------------------------------

// The derived schema types every property as a string. Authoring the schema directly is
// the only way to express a number, an enum, a per-property description or nesting.
func TestToolInputSchema_ExplicitSchemaReplacesDerived(t *testing.T) {
	tool := bridgeTool{
		Method:      "GET",
		Path:        "/pets/{petId}",
		QueryParams: []string{"limit"},
		InputSchema: `{"type":"object","properties":{"limit":{"type":"integer","description":"page size"}},"required":["limit"]}`,
	}

	got := toolEffectiveInputSchema(tool)
	props, ok := got["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties missing from %v", got)
	}
	limit, ok := props["limit"].(map[string]interface{})
	if !ok {
		t.Fatalf("limit property missing from %v", props)
	}
	if limit["type"] != "integer" {
		t.Errorf("limit.type = %v, want integer — the authored schema must win over the derived one", limit["type"])
	}
	if limit["description"] != "page size" {
		t.Errorf("per-property description lost: %v", limit)
	}
	// The derived schema would have added petId from the path; an explicit schema is
	// taken verbatim, so it must not be merged with the derived one.
	if _, present := props["petId"]; present {
		t.Error("an explicit input_schema must be used verbatim, not merged with the derived properties")
	}
}

func TestToolInputSchema_FallsBackToDerivedWhenAbsent(t *testing.T) {
	tool := bridgeTool{Method: "GET", Path: "/pets/{petId}", QueryParams: []string{"verbose"}}

	got := toolEffectiveInputSchema(tool)
	props := got["properties"].(map[string]interface{})
	if _, ok := props["petId"]; !ok {
		t.Error("derived schema must still expose the path parameter when no input_schema is set")
	}
	req, _ := got["required"].([]string)
	if len(req) != 1 || req[0] != "petId" {
		t.Errorf("required = %v, want [petId]", req)
	}
}

func TestMetadataTool_CarriesAuthoredSchema(t *testing.T) {
	tool := bridgeTool{
		Method: "POST", Path: "/pets", Name: "create_pet",
		InputSchema: `{"type":"object","properties":{"nickname":{"type":"string"}}}`,
	}

	raw, err := json.Marshal(metadataTool(tool))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(raw, &got)

	schema := got["inputSchema"].(map[string]interface{})
	props := schema["properties"].(map[string]interface{})
	if _, ok := props["nickname"]; !ok {
		t.Errorf("published metadata lost the authored schema: %s", raw)
	}
}

// --- http_mapping -------------------------------------------------------------

// The UI lets an HTTP parameter be fed by a differently-named tool parameter. Without an
// override the provider can only emit key -> #[vars.params['key']].
func TestTranscodingToolConfig_HTTPMappingOverridesDerived(t *testing.T) {
	tool := bridgeTool{
		Method: "GET", Path: "/search", Name: "search",
		QueryParams: []string{"q"},
		HTTPMapping: &bridgeHTTPMapping{
			QueryParams: mappingPairs(bridgeParamMapping{Key: "query", Value: "#[vars.params['q']]"}),
		},
	}

	got := transcodingToolConfig(tool)
	want := []map[string]string{{"key": "query", "value": "#[vars.params['q']]"}}
	if !reflect.DeepEqual(got["queryParams"], want) {
		t.Errorf("queryParams = %v, want %v", got["queryParams"], want)
	}
}

// Each section is independent so a caller changing only the body does not have to
// restate every parameter.
func TestTranscodingToolConfig_UnsetSectionsStayDerived(t *testing.T) {
	tool := bridgeTool{
		Method: "POST", Path: "/pets/{petId}", Name: "update_pet",
		QueryParams: []string{"dryRun"},
		HTTPMapping: &bridgeHTTPMapping{Body: strRef("#[vars.params.payload]")},
	}

	got := transcodingToolConfig(tool)
	if got["body"] != "#[vars.params.payload]" {
		t.Errorf("body = %v, want the custom expression", got["body"])
	}
	wantQuery := []map[string]string{{"key": "dryRun", "value": "#[vars.params['dryRun']]"}}
	if !reflect.DeepEqual(got["queryParams"], wantQuery) {
		t.Errorf("queryParams = %v, want the derived mapping %v", got["queryParams"], wantQuery)
	}
	wantURI := []map[string]string{{"key": "petId", "value": "#[vars.params['petId']]"}}
	if !reflect.DeepEqual(got["uriParams"], wantURI) {
		t.Errorf("uriParams = %v, want the derived mapping %v", got["uriParams"], wantURI)
	}
}

// An empty list is a real instruction ("send none"), distinct from omitting the section.
func TestTranscodingToolConfig_EmptyListSuppressesDerivedParams(t *testing.T) {
	tool := bridgeTool{
		Method: "GET", Path: "/pets", Name: "list_pets",
		QueryParams: []string{"limit", "offset"},
		HTTPMapping: &bridgeHTTPMapping{QueryParams: mappingPairs()},
	}

	got := transcodingToolConfig(tool)["queryParams"]
	list, ok := got.([]map[string]string)
	if !ok || len(list) != 0 {
		t.Errorf("queryParams = %v, want an empty list — an explicit [] must suppress the derived params", got)
	}
}

// has_body writes the default body expression; an explicit empty override removes it.
func TestTranscodingToolConfig_EmptyBodyOverrideRemovesBody(t *testing.T) {
	tool := bridgeTool{
		Method: "POST", Path: "/pets", Name: "create_pet", HasBody: true,
		HTTPMapping: &bridgeHTTPMapping{Body: strRef("")},
	}

	if _, present := transcodingToolConfig(tool)["body"]; present {
		t.Error("an explicit empty body override must omit the body key entirely")
	}
}

func TestTranscodingToolConfig_DefaultBodyStillApplies(t *testing.T) {
	tool := bridgeTool{Method: "POST", Path: "/pets", Name: "create_pet", HasBody: true}
	if got := transcodingToolConfig(tool)["body"]; got != "#[vars.params.body]" {
		t.Errorf("body = %v, want the default expression", got)
	}
}

// --- update routing ----------------------------------------------------------

// An http_mapping edit changes the policies but not the published metadata. If Update
// derived the policy resync from the metadata comparison, that edit would be dropped.
func TestBridgeTranscodingSignature_ChangesWhenOnlyMappingChanges(t *testing.T) {
	base := []bridgeSource{{
		Label: "petstore",
		Tools: []bridgeTool{{Method: "GET", Path: "/search", Name: "search", QueryParams: []string{"q"}}},
	}}
	edited := []bridgeSource{{
		Label: "petstore",
		Tools: []bridgeTool{{
			Method: "GET", Path: "/search", Name: "search", QueryParams: []string{"q"},
			HTTPMapping: &bridgeHTTPMapping{
				QueryParams: mappingPairs(bridgeParamMapping{Key: "query", Value: "#[vars.params['q']]"}),
			},
		}},
	}}

	if bridgeTranscodingSignature(base) == bridgeTranscodingSignature(edited) {
		t.Error("changing only http_mapping must change the transcoding signature, or the policy resync is skipped")
	}

	// The same edit must NOT look like a metadata change, or it would burn an asset version.
	baseMeta, _ := json.Marshal(buildBridgeMetadata("http://0.0.0.0:8081/", base))
	editedMeta, _ := json.Marshal(buildBridgeMetadata("http://0.0.0.0:8081/", edited))
	if string(baseMeta) != string(editedMeta) {
		t.Error("http_mapping is not published in the asset metadata; it must not trigger a republish")
	}
}

// The mirror image: an input_schema edit changes the metadata but not the policies.
func TestInputSchemaEdit_ChangesMetadataOnly(t *testing.T) {
	base := []bridgeSource{{
		Label: "petstore",
		Tools: []bridgeTool{{Method: "GET", Path: "/pets", Name: "list_pets"}},
	}}
	edited := []bridgeSource{{
		Label: "petstore",
		Tools: []bridgeTool{{
			Method: "GET", Path: "/pets", Name: "list_pets",
			InputSchema: `{"type":"object","properties":{"limit":{"type":"integer"}}}`,
		}},
	}}

	baseMeta, _ := json.Marshal(buildBridgeMetadata("http://0.0.0.0:8081/", base))
	editedMeta, _ := json.Marshal(buildBridgeMetadata("http://0.0.0.0:8081/", edited))
	if string(baseMeta) == string(editedMeta) {
		t.Error("changing input_schema must change the published metadata")
	}
	if bridgeTranscodingSignature(base) != bridgeTranscodingSignature(edited) {
		t.Error("input_schema does not affect the request mapping; it must not resync policies")
	}
}

// ModifyPlan uses bridgeToolSignature to decide whether the computed platform-managed
// fields must be re-planned as unknown. If it misses a field that Update acts on, the
// apply dies with "Provider produced inconsistent result after apply" — which is exactly
// what happened when input_schema was first added.
func TestBridgeToolSignature_CoversEverythingUpdateActsOn(t *testing.T) {
	base := []bridgeSource{{
		Label: "petstore",
		Tools: []bridgeTool{{Method: "GET", Path: "/pets", Name: "list_pets", QueryParams: []string{"limit"}}},
	}}

	variants := map[string]bridgeTool{
		"input_schema": {
			Method: "GET", Path: "/pets", Name: "list_pets", QueryParams: []string{"limit"},
			InputSchema: `{"type":"object","properties":{"limit":{"type":"integer"}}}`,
		},
		"http_mapping": {
			Method: "GET", Path: "/pets", Name: "list_pets", QueryParams: []string{"limit"},
			HTTPMapping: &bridgeHTTPMapping{
				QueryParams: mappingPairs(bridgeParamMapping{Key: "max", Value: "#[vars.params['limit']]"}),
			},
		},
		"description": {
			Method: "GET", Path: "/pets", Name: "list_pets", QueryParams: []string{"limit"},
			Description: "List every pet",
		},
		"query_params": {
			Method: "GET", Path: "/pets", Name: "list_pets", QueryParams: []string{"limit", "offset"},
		},
		"has_body": {
			Method: "GET", Path: "/pets", Name: "list_pets", QueryParams: []string{"limit"}, HasBody: true,
		},
	}

	for field, tool := range variants {
		t.Run(field, func(t *testing.T) {
			edited := []bridgeSource{{Label: "petstore", Tools: []bridgeTool{tool}}}
			if bridgeToolSignature(base) == bridgeToolSignature(edited) {
				t.Errorf("changing %s does not change the tool signature, so ModifyPlan leaves stale "+
					"computed values in the plan and the apply fails as inconsistent", field)
			}
		})
	}
}
