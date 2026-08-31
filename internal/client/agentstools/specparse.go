package agentstools

import (
	"archive/zip"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParsedTool is a normalized MCP tool derived from one operation of a source API
// spec (OpenAPI/Swagger or RAML). It mirrors the fields of the anypoint_mcp_bridge
// resource's `tools` block so a data source can emit it for direct assignment.
//
// Name is the operationId when the spec supplies one (else "" — the bridge derives
// <method>_<slug(path)>). Path parameters are intentionally NOT included in
// QueryParams/HeaderParams: the bridge derives URI params from the `{...}` braces in
// Path automatically.
type ParsedTool struct {
	Name         string
	Description  string
	Method       string // upper-case HTTP method
	Path         string
	QueryParams  []string
	HeaderParams []string
	HasBody      bool
}

// httpMethods is the set of operation keys recognised in both OAS path items and
// RAML resources, in canonical order.
var httpMethods = []string{"get", "post", "put", "patch", "delete", "options", "head", "trace"}

func methodOrder(m string) int {
	for i, x := range httpMethods {
		if strings.EqualFold(m, x) {
			return i
		}
	}
	return len(httpMethods)
}

// ParseAPISpecFile turns a downloaded spec file into a normalized, deterministically
// ordered tool list. `data` may be a raw spec (JSON/YAML/RAML) or a zip bundle
// (fat-oas / fat-raml / oas / raml packaging); `mainFile` names the entry point
// inside a zip. specType is one of "oas3", "oas2", "raml".
func ParseAPISpecFile(classifier string, data []byte, mainFile string) (tools []ParsedTool, specType string, err error) {
	spec := data
	if isZip(data) {
		spec, err = extractSpecFromZip(data, mainFile)
		if err != nil {
			return nil, "", err
		}
	}
	trimmed := bytes.TrimSpace(spec)
	if len(trimmed) == 0 {
		return nil, "", fmt.Errorf("spec file is empty")
	}
	if bytes.HasPrefix(trimmed, []byte("#%RAML")) || strings.Contains(strings.ToLower(classifier), "raml") && !looksLikeOAS(trimmed) {
		return parseRAML(spec)
	}
	return parseOAS(spec)
}

func isZip(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && (data[2] == 3 || data[2] == 5 || data[2] == 7)
}

func looksLikeOAS(b []byte) bool {
	s := string(b)
	return strings.Contains(s, "\"openapi\"") || strings.Contains(s, "openapi:") ||
		strings.Contains(s, "\"swagger\"") || strings.Contains(s, "swagger:")
}

// extractSpecFromZip returns the spec entry inside a zip bundle: the mainFile when
// named, otherwise the single best spec-looking entry (a .json/.yaml/.yml/.raml that
// is not the exchange.json descriptor).
func extractSpecFromZip(data []byte, mainFile string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open spec zip: %w", err)
	}
	var candidate *zip.File
	for _, f := range zr.File {
		base := f.Name
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		if mainFile != "" && base == mainFile {
			return readZipEntry(f)
		}
		if base == "exchange.json" || strings.HasSuffix(base, "/") {
			continue
		}
		lower := strings.ToLower(base)
		if strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") ||
			strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".raml") {
			if candidate == nil {
				candidate = f
			}
		}
	}
	if candidate == nil {
		return nil, fmt.Errorf("no spec entry found in zip (mainFile=%q)", mainFile)
	}
	return readZipEntry(candidate)
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open zip entry %s: %w", f.Name, err)
	}
	defer func() { _ = rc.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rc); err != nil {
		return nil, fmt.Errorf("failed to read zip entry %s: %w", f.Name, err)
	}
	return buf.Bytes(), nil
}

// --- OpenAPI / Swagger ---

// parseOAS parses an OpenAPI 3.x or Swagger 2.0 document (JSON or YAML — yaml.v3
// parses both) into tools. $ref parameters are resolved against the same document.
func parseOAS(spec []byte) ([]ParsedTool, string, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		return nil, "", fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}
	specType := "oas3"
	if _, ok := doc["swagger"]; ok {
		specType = "oas2"
	}
	paths, _ := doc["paths"].(map[string]interface{})
	if paths == nil {
		return nil, specType, fmt.Errorf("spec has no paths")
	}
	var tools []ParsedTool
	for path, pv := range paths {
		pathItem, ok := pv.(map[string]interface{})
		if !ok {
			continue
		}
		pathParams := extractOASParams(pathItem["parameters"], doc)
		for _, m := range httpMethods {
			opRaw, ok := pathItem[m]
			if !ok {
				continue
			}
			op, ok := opRaw.(map[string]interface{})
			if !ok {
				continue
			}
			opParams := extractOASParams(op["parameters"], doc)
			query, header, bodyFromParams := classifyParams(append(append([]map[string]interface{}{}, pathParams...), opParams...))
			hasBody := bodyFromParams
			if _, ok := op["requestBody"]; ok { // OAS3
				hasBody = true
			}
			name, _ := op["operationId"].(string)
			tools = append(tools, ParsedTool{
				Name:         strings.TrimSpace(name),
				Description:  firstString(op["summary"], op["description"]),
				Method:       strings.ToUpper(m),
				Path:         path,
				QueryParams:  query,
				HeaderParams: header,
				HasBody:      hasBody,
			})
		}
	}
	sortTools(tools)
	return tools, specType, nil
}

// extractOASParams resolves a `parameters` array (path- or operation-level) into a
// slice of parameter maps, following local `$ref` pointers.
func extractOASParams(raw interface{}, doc map[string]interface{}) []map[string]interface{} {
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if ref, ok := m["$ref"].(string); ok {
			if resolved := resolveLocalRef(doc, ref); resolved != nil {
				m = resolved
			}
		}
		out = append(out, m)
	}
	return out
}

// classifyParams splits parameters into query names, header names, and whether a
// request body is present (Swagger 2.0 in:body / in:formData). Deduped by name;
// path/cookie params are ignored (the bridge derives URI params from the path).
func classifyParams(params []map[string]interface{}) (query, header []string, hasBody bool) {
	seenQ, seenH := map[string]bool{}, map[string]bool{}
	for _, p := range params {
		name, _ := p["name"].(string)
		in, _ := p["in"].(string)
		switch in {
		case "query":
			if name != "" && !seenQ[name] {
				seenQ[name] = true
				query = append(query, name)
			}
		case "header":
			if name != "" && !seenH[name] {
				seenH[name] = true
				header = append(header, name)
			}
		case "body", "formData":
			hasBody = true
		}
	}
	sort.Strings(query)
	sort.Strings(header)
	return query, header, hasBody
}

// resolveLocalRef resolves a same-document JSON pointer like
// "#/components/parameters/Foo" or "#/parameters/Foo".
func resolveLocalRef(doc map[string]interface{}, ref string) map[string]interface{} {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var node interface{} = doc
	for _, seg := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		seg = strings.ReplaceAll(strings.ReplaceAll(seg, "~1", "/"), "~0", "~")
		m, ok := node.(map[string]interface{})
		if !ok {
			return nil
		}
		node = m[seg]
	}
	if m, ok := node.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// --- RAML (best-effort) ---

// parseRAML parses a RAML 1.0 document best-effort: it walks nested resources
// (keys starting with "/") and their methods. RAML with unresolved !include tags
// (non-fat) cannot be parsed and yields a clear error directing the user to declare
// tools explicitly (Approach A).
func parseRAML(spec []byte) ([]ParsedTool, string, error) {
	// Drop the "#%RAML 1.0" version comment; yaml.v3 tolerates it as a comment but
	// be explicit.
	body := spec
	if i := bytes.IndexByte(body, '\n'); bytes.HasPrefix(bytes.TrimSpace(body), []byte("#%RAML")) && i >= 0 {
		body = body[i+1:]
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, "raml", fmt.Errorf("could not parse RAML spec automatically (%v); declare tools explicitly via source_apis[].tools", err)
	}
	var tools []ParsedTool
	walkRAMLResources(doc, "", &tools)
	if len(tools) == 0 {
		return nil, "raml", fmt.Errorf("no RAML resources found; declare tools explicitly via source_apis[].tools")
	}
	sortTools(tools)
	return tools, "raml", nil
}

func walkRAMLResources(node map[string]interface{}, prefix string, out *[]ParsedTool) {
	for k, v := range node {
		if !strings.HasPrefix(k, "/") {
			continue
		}
		child, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		full := prefix + k
		for _, m := range httpMethods {
			mv, ok := child[m]
			if !ok {
				continue
			}
			mm, _ := mv.(map[string]interface{})
			query := mapKeys(mm["queryParameters"])
			header := mapKeys(mm["headers"])
			_, hasBody := mm["body"]
			*out = append(*out, ParsedTool{
				Description:  strVal(mm["description"]),
				Method:       strings.ToUpper(m),
				Path:         full,
				QueryParams:  query,
				HeaderParams: header,
				HasBody:      hasBody,
			})
		}
		walkRAMLResources(child, full, out)
	}
}

// --- shared helpers ---

func sortTools(tools []ParsedTool) {
	sort.SliceStable(tools, func(i, j int) bool {
		if tools[i].Path != tools[j].Path {
			return tools[i].Path < tools[j].Path
		}
		return methodOrder(tools[i].Method) < methodOrder(tools[j].Method)
	})
}

func firstString(vals ...interface{}) string {
	for _, v := range vals {
		if s, ok := v.(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return ""
}

func strVal(v interface{}) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// mapKeys returns the sorted keys of a YAML mapping value (used for RAML
// queryParameters / headers, which are maps). Deterministic ordering.
func mapKeys(v interface{}) []string {
	m, ok := v.(map[string]interface{})
	if !ok || len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
