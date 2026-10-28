package agentstools

import (
	"strings"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// This file holds the PURE (Terraform-free) reconstruction of an MCP bridge's source
// APIs + tools from its live wire state (routing + upstreams + policies). It is the
// SINGLE source of truth shared by:
//
//   - the anypoint_mcp_bridge RESOURCE, on import (state has no source_apis yet), and
//   - the anypoint_mcp_bridge DATA SOURCE, which always rebuilds from the platform.
//
// Keeping it here (returning plain Go structs) means the tricky "policies LIST drops
// upstreamIds" quirk is implemented and tested once; each caller only flattens the
// result into its own schema's Terraform types.

// Policy asset IDs whose configuration carries the tool→source mapping. These are the
// stable Exchange asset names of the MCP transcoding policies (versions live with the
// resource; only the names matter for reconstruction).
const (
	bridgeTranscodingAsset       = "mcp-transcoding"
	bridgeTranscodingRouterAsset = "mcp-transcoding-router"
)

// ReconstructedTool is a Terraform-agnostic view of one bridge tool rebuilt from the
// live mcp-transcoding policy definition. Name is the raw value stored on the platform
// (it may equal the derived slug — callers decide whether to surface or null it).
type ReconstructedTool struct {
	Name         string
	Method       string
	Path         string
	QueryParams  []string
	HeaderParams []string
	HasBody      bool

	// The full request mapping exactly as stored, including the DataWeave expression
	// on each entry. The fields above are the key-only view used when the mapping is
	// the plain identity one; these carry enough to detect and reproduce a customised
	// mapping on import.
	QueryMapping  []ReconstructedParamMapping
	URIMapping    []ReconstructedParamMapping
	HeaderMapping []ReconstructedParamMapping
	Body          string
}

// ReconstructedParamMapping is one {key,value} row of a tool's stored request mapping.
type ReconstructedParamMapping struct {
	Key   string
	Value string
}

// identityExpression is the DataWeave template the provider generates for a parameter
// that maps to itself.
func identityExpression(key string) string {
	return "#[vars.params['" + key + "']]"
}

// isIdentityMapping reports whether every entry maps a key to its own tool parameter,
// which is exactly what the provider generates from a plain parameter list.
func isIdentityMapping(pairs []ReconstructedParamMapping) bool {
	for _, p := range pairs {
		if p.Value != identityExpression(p.Key) {
			return false
		}
	}
	return true
}

// HasCustomHTTPMapping reports whether the stored mapping differs from what the plain
// parameter lists would generate. Import uses it to decide between emitting the concise
// query_params/header_params form and an explicit http_mapping block — emitting the
// wrong one would show drift on the next plan.
//
// uriParams are derived from the path's `{...}` placeholders, so a URI mapping is custom
// when its keys no longer match those placeholders or its expressions are not identity.
func (t ReconstructedTool) HasCustomHTTPMapping() bool {
	if !isIdentityMapping(t.QueryMapping) || !isIdentityMapping(t.HeaderMapping) || !isIdentityMapping(t.URIMapping) {
		return true
	}

	uriKeys := make([]string, 0, len(t.URIMapping))
	for _, p := range t.URIMapping {
		uriKeys = append(uriKeys, p.Key)
	}
	if !equalStrings(uriKeys, PathURIParamKeys(t.Path)) {
		return true
	}

	return t.Body != "" && t.Body != "#[vars.params.body]"
}

// PathURIParamKeys returns the ordered `{...}` placeholder names in a path.
func PathURIParamKeys(path string) []string {
	keys := make([]string, 0)
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") && len(seg) > 2 {
			keys = append(keys, seg[1:len(seg)-1])
		}
	}
	return keys
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ReconstructedSource is a Terraform-agnostic view of one source REST API rebuilt from
// a bridge's live routing + upstreams + policies.
type ReconstructedSource struct {
	Label       string
	UpstreamURI string
	AssetID     string
	GroupID     string
	Version     string
	// TLSContextID is the user-facing "secretGroupId/tlsContextId" form, empty when the
	// upstream has no TLS context. Reconstructed so an imported bridge round-trips
	// instead of showing a spurious diff on tls_context_id.
	TLSContextID string
	Tools        []ReconstructedTool
}

// BridgeToolName derives the MCP tool name from a method + path exactly as the platform
// does: lower(method) + "_" + slug(path), where each path segment is lower-cased and
// its `{...}` braces stripped. GET /pets/{petId} -> get_pets_petid.
func BridgeToolName(method, path string) string {
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

// BridgeToolMappingFromPolicies extracts, from a bridge's live policy list, the data
// needed to map tools back to their source API. It returns:
//   - toolDefByName:    every mcp-transcoding tool definition keyed by tool name
//     (names are globally unique across a bridge).
//   - toolDefsOrdered:  the same defs in policy order (used by the single-source
//     fallback so reconstruction is order-stable — no post-import churn).
//   - toolNamesByLabel: from the mcp-transcoding-router policy's routes, the tool
//     names each source-API label (X-UPSTREAM-NAME) serves.
//
// This deliberately does NOT rely on the transcoding policy's upstreamIds: the policies
// LIST endpoint returns the tool DEFINITIONS but drops upstreamIds for outbound policies
// (confirmed live), so the router policy is the only reliable tool→source mapping.
func BridgeToolMappingFromPolicies(policies []apimanagement.APIPolicy) (map[string]interface{}, []interface{}, map[string][]string) {
	toolDefByName := map[string]interface{}{}
	toolDefsOrdered := []interface{}{}
	toolNamesByLabel := map[string][]string{}
	for _, p := range policies {
		switch p.AssetID {
		case bridgeTranscodingAsset:
			if raw, ok := p.ConfigurationData["tools"].([]interface{}); ok {
				for _, item := range raw {
					if m, ok := item.(map[string]interface{}); ok {
						if name, _ := m["name"].(string); name != "" {
							toolDefByName[name] = item
							toolDefsOrdered = append(toolDefsOrdered, item)
						}
					}
				}
			}
		case bridgeTranscodingRouterAsset:
			routes, _ := p.ConfigurationData["routes"].([]interface{})
			for _, rt := range routes {
				rm, ok := rt.(map[string]interface{})
				if !ok {
					continue
				}
				label, _ := rm["upstreamName"].(string)
				names := []string{}
				if toolsRaw, ok := rm["tools"].([]interface{}); ok {
					for _, tn := range toolsRaw {
						if s, ok := tn.(string); ok {
							names = append(names, s)
						}
					}
				}
				if label != "" {
					toolNamesByLabel[label] = names
				}
			}
		}
	}
	return toolDefByName, toolDefsOrdered, toolNamesByLabel
}

// ReconstructBridgeSources rebuilds the list of source APIs (with their tools) from a
// bridge's live instance (routing), its upstreams sub-resource (backend URIs +
// connection to the source Exchange asset), and its policies (tool definitions).
// groupID defaults each source's group to orgID when the connection omits it.
func ReconstructBridgeSources(orgID string, inst *MCPBridge, ups []MCPBridgeUpstreamDetail, policies []apimanagement.APIPolicy) []ReconstructedSource {
	toolDefByName, toolDefsOrdered, toolNamesByLabel := BridgeToolMappingFromPolicies(policies)

	// upstreamID -> route label (from routing rules / X-UPSTREAM-NAME).
	labelByUpstream := map[string]string{}
	if inst != nil {
		for _, route := range inst.Routing {
			for _, u := range route.Upstreams {
				if u.ID != "" {
					labelByUpstream[u.ID] = route.Label
				}
			}
		}
	}

	// Fallback: if the router policy yielded no per-label mapping but there is a single
	// upstream, assign every pooled tool def to it (single-source bridges).
	singleSourceFallback := len(toolNamesByLabel) == 0 && len(ups) == 1

	out := make([]ReconstructedSource, 0, len(ups))
	for _, u := range ups {
		label := labelByUpstream[u.ID]
		aid, gid, ver := "", orgID, ""
		if u.Connection != nil {
			aid, ver = u.Connection.AssetID, u.Connection.Version
			if u.Connection.GroupID != "" {
				gid = u.Connection.GroupID
			}
			if label == "" {
				label = u.Connection.Label
			}
		}

		var toolDefs []interface{}
		if singleSourceFallback {
			toolDefs = toolDefsOrdered
		} else {
			for _, name := range toolNamesByLabel[label] {
				if def, ok := toolDefByName[name]; ok {
					toolDefs = append(toolDefs, def)
				}
			}
		}

		tlsCtx := ""
		if u.TLSContext != nil && u.TLSContext.SecretGroupID != "" && u.TLSContext.TLSContextID != "" {
			tlsCtx = u.TLSContext.SecretGroupID + "/" + u.TLSContext.TLSContextID
		}

		out = append(out, ReconstructedSource{
			Label:        label,
			UpstreamURI:  u.URI,
			AssetID:      aid,
			GroupID:      gid,
			Version:      ver,
			TLSContextID: tlsCtx,
			Tools:        reconstructBridgeTools(toolDefs),
		})
	}
	return out
}

// reconstructBridgeTools rebuilds tool views from a transcoding policy's tools[].
func reconstructBridgeTools(raw []interface{}) []ReconstructedTool {
	out := make([]ReconstructedTool, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		method, _ := m["method"].(string)
		pathVal, _ := m["path"].(string)
		name, _ := m["name"].(string)
		body, hasBody := m["body"].(string)
		out = append(out, ReconstructedTool{
			Name:          name,
			Method:        method,
			Path:          pathVal,
			QueryParams:   bridgeParamKeysFromConfig(m["queryParams"]),
			HeaderParams:  bridgeParamKeysFromConfig(m["headers"]),
			HasBody:       hasBody,
			QueryMapping:  bridgeParamMappingFromConfig(m["queryParams"]),
			URIMapping:    bridgeParamMappingFromConfig(m["uriParams"]),
			HeaderMapping: bridgeParamMappingFromConfig(m["headers"]),
			Body:          body,
		})
	}
	return out
}

// bridgeParamMappingFromConfig pulls the ordered {key,value} pairs out of a policy param
// list, preserving the DataWeave expression so a customised mapping can be detected.
func bridgeParamMappingFromConfig(v interface{}) []ReconstructedParamMapping {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]ReconstructedParamMapping, 0, len(arr))
	for _, e := range arr {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := m["key"].(string)
		val, _ := m["value"].(string)
		if key != "" {
			out = append(out, ReconstructedParamMapping{Key: key, Value: val})
		}
	}
	return out
}

// bridgeParamKeysFromConfig pulls the ordered `key` values out of a policy param list
// ([]{key,value}). Non-list / malformed entries are skipped.
func bridgeParamKeysFromConfig(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(arr))
	for _, e := range arr {
		if m, ok := e.(map[string]interface{}); ok {
			if k, ok := m["key"].(string); ok {
				keys = append(keys, k)
			}
		}
	}
	return keys
}
