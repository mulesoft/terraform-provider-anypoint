package agentstools

import (
	"archive/zip"
	"bytes"
	"reflect"
	"testing"
)

const oas3Petstore = `{
  "openapi": "3.0.0",
  "info": {"title": "Petstore", "version": "1.0.0"},
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "summary": "List all pets",
        "parameters": [
          {"name": "limit", "in": "query", "schema": {"type": "integer"}},
          {"name": "X-Trace", "in": "header", "schema": {"type": "string"}}
        ]
      },
      "post": {
        "operationId": "createPet",
        "requestBody": {"content": {"application/json": {"schema": {"type": "object"}}}}
      }
    },
    "/pets/{petId}": {
      "parameters": [{"name": "petId", "in": "path", "required": true, "schema": {"type": "string"}}],
      "get": {
        "operationId": "getPet",
        "parameters": [{"$ref": "#/components/parameters/Expand"}]
      },
      "delete": {"operationId": "deletePet"}
    }
  },
  "components": {
    "parameters": {
      "Expand": {"name": "expand", "in": "query", "schema": {"type": "string"}}
    }
  }
}`

func findTool(tools []ParsedTool, method, path string) *ParsedTool {
	for i := range tools {
		if tools[i].Method == method && tools[i].Path == path {
			return &tools[i]
		}
	}
	return nil
}

func TestParseOAS3(t *testing.T) {
	tools, specType, err := ParseAPISpecFile("oas", []byte(oas3Petstore), "")
	if err != nil {
		t.Fatalf("ParseAPISpecFile() error = %v", err)
	}
	if specType != "oas3" {
		t.Errorf("specType = %q, want oas3", specType)
	}
	if len(tools) != 4 {
		t.Fatalf("expected 4 tools, got %d: %+v", len(tools), tools)
	}
	// Deterministic order: /pets (get, post), /pets/{petId} (get, delete)
	wantOrder := [][2]string{{"GET", "/pets"}, {"POST", "/pets"}, {"GET", "/pets/{petId}"}, {"DELETE", "/pets/{petId}"}}
	for i, w := range wantOrder {
		if tools[i].Method != w[0] || tools[i].Path != w[1] {
			t.Errorf("tool[%d] = %s %s, want %s %s", i, tools[i].Method, tools[i].Path, w[0], w[1])
		}
	}

	get := findTool(tools, "GET", "/pets")
	if get.Name != "listPets" {
		t.Errorf("name = %q, want listPets", get.Name)
	}
	if get.Description != "List all pets" {
		t.Errorf("description = %q", get.Description)
	}
	if !reflect.DeepEqual(get.QueryParams, []string{"limit"}) {
		t.Errorf("query = %v, want [limit]", get.QueryParams)
	}
	if !reflect.DeepEqual(get.HeaderParams, []string{"X-Trace"}) {
		t.Errorf("header = %v, want [X-Trace]", get.HeaderParams)
	}
	if get.HasBody {
		t.Error("GET /pets should not have a body")
	}

	post := findTool(tools, "POST", "/pets")
	if !post.HasBody {
		t.Error("POST /pets should have a body (requestBody)")
	}

	// $ref query param resolved; path param (petId) must NOT be in query.
	getOne := findTool(tools, "GET", "/pets/{petId}")
	if !reflect.DeepEqual(getOne.QueryParams, []string{"expand"}) {
		t.Errorf("getOne query = %v, want [expand]", getOne.QueryParams)
	}
	for _, q := range getOne.QueryParams {
		if q == "petId" {
			t.Error("path param petId leaked into query params")
		}
	}
}

func TestParseOAS3_Deterministic(t *testing.T) {
	// Parse twice; identical order (guards against Go map iteration nondeterminism).
	a, _, _ := ParseAPISpecFile("oas", []byte(oas3Petstore), "")
	b, _, _ := ParseAPISpecFile("oas", []byte(oas3Petstore), "")
	if !reflect.DeepEqual(a, b) {
		t.Error("parse output is not deterministic")
	}
}

const swagger2 = `{
  "swagger": "2.0",
  "info": {"title": "Legacy", "version": "1.0"},
  "paths": {
    "/items": {
      "post": {
        "operationId": "addItem",
        "parameters": [
          {"name": "body", "in": "body", "schema": {"type": "object"}},
          {"name": "q", "in": "query", "type": "string"}
        ]
      }
    }
  }
}`

func TestParseOAS2_BodyParam(t *testing.T) {
	tools, specType, err := ParseAPISpecFile("oas", []byte(swagger2), "")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if specType != "oas2" {
		t.Errorf("specType = %q, want oas2", specType)
	}
	if len(tools) != 1 {
		t.Fatalf("want 1 tool, got %d", len(tools))
	}
	if !tools[0].HasBody {
		t.Error("Swagger 2.0 in:body param should set has_body")
	}
	if !reflect.DeepEqual(tools[0].QueryParams, []string{"q"}) {
		t.Errorf("query = %v, want [q]", tools[0].QueryParams)
	}
}

const oas3YAML = `openapi: 3.0.0
info:
  title: YAML API
  version: 1.0.0
paths:
  /health:
    get:
      operationId: getHealth
`

func TestParseOAS3_YAML(t *testing.T) {
	tools, specType, err := ParseAPISpecFile("oas", []byte(oas3YAML), "")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if specType != "oas3" || len(tools) != 1 || tools[0].Path != "/health" {
		t.Errorf("unexpected YAML parse: type=%s tools=%+v", specType, tools)
	}
}

const ramlSpec = `#%RAML 1.0
title: Orders API
version: v1
/orders:
  get:
    queryParameters:
      status:
      page:
  post:
    body:
      application/json:
  /{orderId}:
    get:
      headers:
        X-Request-Id:
`

func TestParseRAML(t *testing.T) {
	tools, specType, err := ParseAPISpecFile("raml", []byte(ramlSpec), "")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if specType != "raml" {
		t.Errorf("specType = %q, want raml", specType)
	}
	if len(tools) != 3 {
		t.Fatalf("want 3 tools, got %d: %+v", len(tools), tools)
	}
	get := findTool(tools, "GET", "/orders")
	if get == nil {
		t.Fatal("missing GET /orders")
	}
	if !reflect.DeepEqual(get.QueryParams, []string{"page", "status"}) { // sorted
		t.Errorf("raml query = %v, want [page status]", get.QueryParams)
	}
	post := findTool(tools, "POST", "/orders")
	if post == nil || !post.HasBody {
		t.Error("POST /orders should have a body")
	}
	nested := findTool(tools, "GET", "/orders/{orderId}")
	if nested == nil {
		t.Fatal("nested resource /orders/{orderId} not walked")
	}
	if !reflect.DeepEqual(nested.HeaderParams, []string{"X-Request-Id"}) {
		t.Errorf("nested header = %v", nested.HeaderParams)
	}
}

func TestParseAPISpecFile_Zip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// exchange.json descriptor must be ignored in favour of the mainFile.
	fd, _ := zw.Create("exchange.json")
	_, _ = fd.Write([]byte(`{"main":"petstore.json"}`))
	fp, _ := zw.Create("petstore.json")
	_, _ = fp.Write([]byte(oas3Petstore))
	_ = zw.Close()

	tools, specType, err := ParseAPISpecFile("fat-oas", buf.Bytes(), "petstore.json")
	if err != nil {
		t.Fatalf("zip parse error = %v", err)
	}
	if specType != "oas3" || len(tools) != 4 {
		t.Errorf("zip parse: type=%s tools=%d", specType, len(tools))
	}
}

func TestParseAPISpecFile_ZipNoMainFile(t *testing.T) {
	// No mainFile hint: the single non-exchange.json spip entry is chosen.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fd, _ := zw.Create("exchange.json")
	_, _ = fd.Write([]byte(`{}`))
	fp, _ := zw.Create("api.json")
	_, _ = fp.Write([]byte(oas3YAML))
	_ = zw.Close()

	tools, _, err := ParseAPISpecFile("fat-oas", buf.Bytes(), "")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("want 1 tool, got %d", len(tools))
	}
}

func TestSelectSpecFile_Preference(t *testing.T) {
	// fat-oas beats oas beats raml.
	got := selectSpecFileClassifier([]string{"raml", "oas", "fat-oas", "rest-api-metadata"})
	if got != "fat-oas" {
		t.Errorf("selected %q, want fat-oas", got)
	}
	got = selectSpecFileClassifier([]string{"raml", "fat-raml"})
	if got != "fat-raml" {
		t.Errorf("selected %q, want fat-raml", got)
	}
	got = selectSpecFileClassifier([]string{"rest-api-metadata", "pom"})
	if got != "" {
		t.Errorf("selected %q, want empty (no spec)", got)
	}
}
