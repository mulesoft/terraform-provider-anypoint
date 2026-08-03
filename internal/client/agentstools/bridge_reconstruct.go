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
}

// ReconstructedSource is a Terraform-agnostic view of one source REST API rebuilt from
// a bridge's live routing + upstreams + policies.
type ReconstructedSource struct {
	Label       string
	UpstreamURI string
	AssetID     string
	GroupID     string
	Version     string
	Tools       []ReconstructedTool
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

		out = append(out, ReconstructedSource{
			Label:       label,
			UpstreamURI: u.URI,
			AssetID:     aid,
			GroupID:     gid,
			Version:     ver,
			Tools:       reconstructBridgeTools(toolDefs),
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
		_, hasBody := m["body"]
		out = append(out, ReconstructedTool{
			Name:         name,
			Method:       method,
			Path:         pathVal,
			QueryParams:  bridgeParamKeysFromConfig(m["queryParams"]),
			HeaderParams: bridgeParamKeysFromConfig(m["headers"]),
			HasBody:      hasBody,
		})
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
