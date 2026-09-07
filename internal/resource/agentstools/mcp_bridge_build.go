package agentstools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// This file holds the PURE wire-assembly for the MCP bridge (Approach A: explicit
// tools). Every shape here is grounded in a live capture of a real bridge on the
// tf-smg-onefile gateway (see .agents/artifacts/mcp-bridge-onefile-capture.md):
//
//   - the generated Exchange asset's mcp-metadata.json  = {transport, tools[]}
//   - the mcp-transcoding-router (inbound) config        = {transcodingPath, routes[]}
//   - the per-upstream mcp-transcoding (outbound) config = {tools[] full mapping}
//
// These are plain funcs on plain structs (no Terraform types) so they are trivially
// unit-testable and independent of the schema/state plumbing.

// MCP policy template GAVs — the exact template.assetVersion values the LIVE bridge
// carries (trusted over the dated strings in apimanagement.KnownPolicies where they
// differ; see the capture doc). All live in the MuleSoft policy org.
const (
	mcpPolicyGroupID = "68ef9520-24e9-4cf2-b2f5-620025690913"

	mcpSupportAsset          = "mcp-support"
	mcpSupportVersion        = "1.0.1"
	mcpSchemaValidationAsset = "mcp-schema-validation"
	mcpSchemaValidationVer   = "1.1.1"
	mcpTranscodingRouterAsst = "mcp-transcoding-router"
	mcpTranscodingRouterVer  = "1.0.1"
	mcpTranscodingAsset      = "mcp-transcoding"
	mcpTranscodingVersion    = "1.0.0"

	// mcpTranscodingPath is the consumer-facing MCP path the router matches on. The
	// live bridge uses "/mcp" regardless of the instance base_path.
	mcpTranscodingPath = "/mcp"
)

// bridgeTool is a normalized (Terraform-free) view of one explicit tool.
type bridgeTool struct {
	Name         string // optional; defaults to bridgeToolName(method, path)
	Description  string // optional; defaults to the name
	Method       string // GET/POST/PUT/PATCH/DELETE
	Path         string // e.g. /pets/{petId}
	QueryParams  []string
	HeaderParams []string
	HasBody      bool

	// InputSchema is a raw JSON Schema object replacing the one derived from the
	// params above. Empty means "derive it". The UI exposes the same thing as a free
	// JSON editor, which is the only way to express non-string types, per-property
	// descriptions, enums or nesting.
	InputSchema string

	// HTTPMapping overrides the REST request mapping the transcoding policy performs.
	// Nil means "derive everything"; individual nil fields inside it fall back to the
	// derived value, so changing only the body expression does not force the caller to
	// restate every parameter.
	HTTPMapping *bridgeHTTPMapping
}

// bridgeParamMapping is one {key,value} pair of the transcoding policy's request
// mapping: key is the name sent upstream, value the DataWeave expression producing it.
type bridgeParamMapping struct {
	Key   string
	Value string
}

// bridgeHTTPMapping mirrors the UI's "HTTP mapping (optional)" tab. A nil field is
// unset (derive it); a non-nil empty slice means "explicitly no parameters here".
type bridgeHTTPMapping struct {
	QueryParams *[]bridgeParamMapping
	URIParams   *[]bridgeParamMapping
	Headers     *[]bridgeParamMapping
	Body        *string
}

// bridgeSource is a normalized view of one source REST API + its tools.
type bridgeSource struct {
	Label        string // routing label + X-UPSTREAM-NAME
	UpstreamURI  string // real backend
	AssetID      string // source Exchange asset
	GroupID      string // source asset group (defaults to org)
	Version      string // source asset version
	TLSContextID string // optional "secretGroupId/tlsContextId" for the upstream
	Tools        []bridgeTool
}

// splitTLSContextID splits the user-facing "secretGroupId/tlsContextId" form into its
// two halves, mirroring anypoint_mcp_server. Anything that is not exactly two non-empty
// segments is reported as absent rather than sent half-populated — the platform rejects
// a partial pair, and silently dropping it is easier to diagnose than a 400.
func splitTLSContextID(s string) (secretGroupID, tlsContextID string, ok bool) {
	parts := strings.Split(strings.TrimSpace(s), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// bridgeToolName derives the MCP tool name from a method + path exactly as the
// platform does: lower(method) + "_" + slug(path), where each path segment is
// lower-cased and its `{...}` braces stripped. GET /pets/{petId} -> get_pets_petid.
func bridgeToolName(method, path string) string {
	name := strings.ToLower(strings.TrimSpace(method))
	for _, seg := range strings.Split(path, "/") {
		if seg == "" {
			continue
		}
		seg = strings.ReplaceAll(seg, "{", "")
		seg = strings.ReplaceAll(seg, "}", "")
		name += "_" + strings.ToLower(seg)
	}
	return name
}

// toolEffectiveName returns the explicit name if set, else the derived slug.
func (t bridgeTool) toolEffectiveName() string {
	if strings.TrimSpace(t.Name) != "" {
		return t.Name
	}
	return bridgeToolName(t.Method, t.Path)
}

// toolEffectiveDescription returns the explicit description if set, else the name.
func (t bridgeTool) toolEffectiveDescription() string {
	if strings.TrimSpace(t.Description) != "" {
		return t.Description
	}
	return t.toolEffectiveName()
}

// pathURIParamKeys returns the ordered `{...}` placeholder names in a path
// (original case): /pets/{petId}/vaccinations -> ["petId"].
func pathURIParamKeys(path string) []string {
	keys := make([]string, 0)
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") && len(seg) > 2 {
			keys = append(keys, seg[1:len(seg)-1])
		}
	}
	return keys
}

// dataweaveParam wraps a param key in the mechanical DataWeave template the
// transcoding policy uses: #[vars.params['<key>']].
func dataweaveParam(key string) map[string]string {
	return map[string]string{"key": key, "value": fmt.Sprintf("#[vars.params['%s']]", key)}
}

// paramList builds the {key,value} wrapper list for a set of param keys. Always a
// non-nil slice so it serializes as [] (never null), matching the live payload.
func paramList(keys []string) []map[string]string {
	out := make([]map[string]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, dataweaveParam(k))
	}
	return out
}

// mappingList renders an explicit override list into the policy's {key,value} shape.
// Always non-nil so it serializes as [] rather than null, matching the live payload.
func mappingList(pairs []bridgeParamMapping) []map[string]string {
	out := make([]map[string]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, map[string]string{"key": p.Key, "value": p.Value})
	}
	return out
}

// resolveParamMapping picks the explicit override when one is set, else the identity
// mapping derived from the declared parameter keys.
func resolveParamMapping(override *[]bridgeParamMapping, derivedKeys []string) []map[string]string {
	if override != nil {
		return mappingList(*override)
	}
	return paramList(derivedKeys)
}

// transcodingToolConfig builds ONE entry of a per-upstream mcp-transcoding policy's
// `tools[]` — the full REST mapping. uriParams are derived from the path's `{...}`
// unless the tool overrides them.
func transcodingToolConfig(t bridgeTool) map[string]interface{} {
	var q, u, h *[]bridgeParamMapping
	var body *string
	if t.HTTPMapping != nil {
		q, u, h, body = t.HTTPMapping.QueryParams, t.HTTPMapping.URIParams, t.HTTPMapping.Headers, t.HTTPMapping.Body
	}

	m := map[string]interface{}{
		"name":        t.toolEffectiveName(),
		"path":        t.Path,
		"method":      strings.ToUpper(strings.TrimSpace(t.Method)),
		"queryParams": resolveParamMapping(q, t.QueryParams),
		"uriParams":   resolveParamMapping(u, pathURIParamKeys(t.Path)),
		"headers":     resolveParamMapping(h, t.HeaderParams),
	}

	switch {
	case body != nil:
		// An explicit empty expression means "send no body", so the key is omitted
		// entirely rather than written as "".
		if *body != "" {
			m["body"] = *body
		}
	case t.HasBody:
		m["body"] = "#[vars.params.body]"
	}
	return m
}

// transcodingConfig builds a per-upstream mcp-transcoding (outbound) policy config
// {tools:[...]} for one source API.
func transcodingConfig(src bridgeSource) map[string]interface{} {
	tools := make([]interface{}, 0, len(src.Tools))
	for _, t := range src.Tools {
		tools = append(tools, transcodingToolConfig(t))
	}
	return map[string]interface{}{"tools": tools}
}

// bridgeTranscodingSignature is a stable rendering of every source's transcoding policy
// config. Update compares it to decide whether the policies need re-syncing.
//
// This is deliberately separate from the asset metadata: an http_mapping edit changes
// the policy but not the published metadata, while an input_schema edit does the
// reverse. Deriving one decision from the other would silently drop half the edits.
func bridgeTranscodingSignature(sources []bridgeSource) string {
	configs := make([]interface{}, 0, len(sources))
	for _, src := range sources {
		configs = append(configs, map[string]interface{}{
			"label": src.Label,
			"tools": transcodingConfig(src)["tools"],
		})
	}
	out, err := json.Marshal(configs)
	if err != nil {
		// A marshal failure must not be read as "nothing changed"; returning a unique
		// value forces the resync rather than skipping it.
		return fmt.Sprintf("unmarshalable-%v", err)
	}
	return string(out)
}

// routerConfig builds the inbound mcp-transcoding-router policy config across all
// source APIs: {transcodingPath, routes:[{upstreamName,tools:[names]}]}.
func routerConfig(sources []bridgeSource) map[string]interface{} {
	routes := make([]interface{}, 0, len(sources))
	for _, src := range sources {
		names := make([]interface{}, 0, len(src.Tools))
		for _, t := range src.Tools {
			names = append(names, t.toolEffectiveName())
		}
		routes = append(routes, map[string]interface{}{
			"upstreamName": src.Label,
			"tools":        names,
		})
	}
	return map[string]interface{}{
		"transcodingPath": mcpTranscodingPath,
		"routes":          routes,
	}
}

// toolInputSchema builds the JSON-Schema `inputSchema` for a tool's metadata entry.
// uri params + body are required; query/header params are optional (mirrors live).
func toolInputSchema(t bridgeTool) map[string]interface{} {
	properties := map[string]interface{}{}
	required := make([]string, 0)

	for _, k := range pathURIParamKeys(t.Path) {
		properties[k] = map[string]interface{}{"type": "string"}
		required = append(required, k)
	}
	for _, k := range t.QueryParams {
		properties[k] = map[string]interface{}{"type": "string"}
	}
	for _, k := range t.HeaderParams {
		properties[k] = map[string]interface{}{"type": "string"}
	}
	if t.HasBody {
		properties["body"] = map[string]interface{}{"type": "object"}
		required = append(required, "body")
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// toolEffectiveInputSchema returns the caller's raw JSON Schema when one is declared,
// otherwise the schema derived from the tool's parameters. An unparseable override
// falls back to the derived schema rather than corrupting the published asset;
// ValidateConfig rejects that case at plan time so it cannot reach here in practice.
func toolEffectiveInputSchema(t bridgeTool) map[string]interface{} {
	if strings.TrimSpace(t.InputSchema) == "" {
		return toolInputSchema(t)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(t.InputSchema), &parsed); err != nil {
		return toolInputSchema(t)
	}
	return parsed
}

// metadataTool builds ONE entry of the generated asset's mcp-metadata.json tools[].
func metadataTool(t bridgeTool) map[string]interface{} {
	return map[string]interface{}{
		"name":        t.toolEffectiveName(),
		"description": t.toolEffectiveDescription(),
		"inputSchema": toolEffectiveInputSchema(t),
	}
}

// buildBridgeMetadata builds the full mcp-metadata.json body for the generated MCP
// Exchange asset, matching the live shape: {transport:{kind,path}, tools:[...]}.
// transportPath is the instance proxy URI (http://0.0.0.0:<port>/<base_path>).
func buildBridgeMetadata(transportPath string, sources []bridgeSource) map[string]interface{} {
	tools := make([]interface{}, 0)
	for _, src := range sources {
		for _, t := range src.Tools {
			tools = append(tools, metadataTool(t))
		}
	}
	return map[string]interface{}{
		"transport": map[string]interface{}{
			"kind": "streamableHttp",
			"path": transportPath,
		},
		"tools": tools,
	}
}
