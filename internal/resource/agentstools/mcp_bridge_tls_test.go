package agentstools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

// TestSplitTLSContextID pins the "secretGroupId/tlsContextId" parsing, which mirrors
// anypoint_mcp_server. Anything that is not exactly two non-empty halves is reported as
// absent rather than sent half-populated: the platform rejects a partial pair, and a
// dropped context is easier to diagnose than a 400 mid-apply.
func TestSplitTLSContextID(t *testing.T) {
	cases := []struct {
		in     string
		wantSG string
		wantTL string
		wantOK bool
	}{
		{"sg-123/tls-456", "sg-123", "tls-456", true},
		{"  sg-123/tls-456  ", "sg-123", "tls-456", true},
		{"", "", "", false},
		{"sg-123", "", "", false},       // missing half
		{"sg-123/", "", "", false},      // empty tls id
		{"/tls-456", "", "", false},     // empty secret group
		{"sg/tls/extra", "", "", false}, // too many segments
	}
	for _, tc := range cases {
		sg, tl, ok := splitTLSContextID(tc.in)
		if ok != tc.wantOK || sg != tc.wantSG || tl != tc.wantTL {
			t.Errorf("splitTLSContextID(%q) = (%q,%q,%v), want (%q,%q,%v)",
				tc.in, sg, tl, ok, tc.wantSG, tc.wantTL, tc.wantOK)
		}
	}
}

// TestBuildBridgeRouting_ThreadsTLSContext is the one that matters: the schema attribute
// is worthless unless it reaches the wire. A bridge upstream must carry tlsContext when
// the source API declares one, and must omit it entirely otherwise (the field is
// `omitempty`, so an empty struct would serialise as a partial pair).
func TestBuildBridgeRouting_ThreadsTLSContext(t *testing.T) {
	sources := []bridgeSource{
		{Label: "secure", UpstreamURI: "https://backend", AssetID: "a", GroupID: "g",
			Version: "1.0.0", TLSContextID: "sg-123/tls-456"},
		{Label: "plain", UpstreamURI: "http://backend", AssetID: "b", GroupID: "g",
			Version: "1.0.0"},
		{Label: "malformed", UpstreamURI: "https://backend", AssetID: "c", GroupID: "g",
			Version: "1.0.0", TLSContextID: "not-a-pair"},
	}

	routes := buildBridgeRouting(sources)
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	secure := routes[0].Upstreams[0]
	if secure.TLSContext == nil {
		t.Fatal("declared tls_context_id did not reach the upstream: the attribute would " +
			"be silently ignored and the backend connection would not use the TLS context")
	}
	if secure.TLSContext.SecretGroupID != "sg-123" || secure.TLSContext.TLSContextID != "tls-456" {
		t.Errorf("tlsContext = %+v, want {sg-123 tls-456}", *secure.TLSContext)
	}

	if routes[1].Upstreams[0].TLSContext != nil {
		t.Error("a source API without tls_context_id must not send a tlsContext")
	}
	if routes[2].Upstreams[0].TLSContext != nil {
		t.Error("a malformed tls_context_id must be dropped, not sent half-populated")
	}
}

// TestBridgeStructuralSignature_IncludesTLSContext guards the replace semantics.
//
// buildBridgeRouting has exactly ONE call site — Create. Update never sends routing, so
// a changed TLS context cannot be patched in place; it has to force replacement the same
// way upstream_uri does. If the signature ignored it, `terraform apply` would report
// success while the platform kept the old TLS context.
func TestBridgeStructuralSignature_IncludesTLSContext(t *testing.T) {
	base := []bridgeSource{{Label: "a", UpstreamURI: "https://u", AssetID: "x", GroupID: "g",
		Version: "1.0.0", TLSContextID: "sg-1/tls-1"}}
	changed := []bridgeSource{{Label: "a", UpstreamURI: "https://u", AssetID: "x", GroupID: "g",
		Version: "1.0.0", TLSContextID: "sg-1/tls-2"}}
	removed := []bridgeSource{{Label: "a", UpstreamURI: "https://u", AssetID: "x", GroupID: "g",
		Version: "1.0.0"}}

	if bridgeStructuralSignature(base) == bridgeStructuralSignature(changed) {
		t.Error("changing tls_context_id must change the structural signature, or the " +
			"plan would show an in-place update that the Update path cannot perform")
	}
	if bridgeStructuralSignature(base) == bridgeStructuralSignature(removed) {
		t.Error("removing tls_context_id must change the structural signature")
	}
}

// TestReconstructBridgeSources_RecoversTLSContext covers the import/refresh direction.
// Without it the attribute would read back null on an imported bridge that really does
// have a TLS context, and — because it is structural — the next plan would propose a
// destroy-and-recreate of a healthy bridge.
func TestReconstructBridgeSources_RecoversTLSContext(t *testing.T) {
	label := "secure"
	ups := []agentstools.MCPBridgeUpstreamDetail{
		{
			ID:    "u1",
			URI:   "https://backend",
			Label: &label,
			Connection: &agentstools.MCPBridgeConnection{
				Label: label, AssetID: "a", GroupID: "g", Version: "1.0.0",
			},
			TLSContext: &agentstools.MCPBridgeUpstreamTLS{
				SecretGroupID: "sg-123", TLSContextID: "tls-456",
			},
		},
	}

	got := agentstools.ReconstructBridgeSources("org", nil, ups, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 reconstructed source, got %d", len(got))
	}
	if want := "sg-123/tls-456"; got[0].TLSContextID != want {
		t.Errorf("TLSContextID = %q, want %q — an imported bridge would show a spurious "+
			"diff and, since the field is structural, plan a replacement", got[0].TLSContextID, want)
	}
}

// TestReconstructBridgeSources_NoTLSReadsBackEmpty keeps the absent case honest: an
// upstream with no TLS context must reconstruct as empty (which the resource maps to
// null), not as a partial "/" string.
func TestReconstructBridgeSources_NoTLSReadsBackEmpty(t *testing.T) {
	label := "plain"
	ups := []agentstools.MCPBridgeUpstreamDetail{
		{ID: "u1", URI: "http://backend", Label: &label,
			Connection: &agentstools.MCPBridgeConnection{Label: label, AssetID: "a", GroupID: "g", Version: "1.0.0"}},
	}
	got := agentstools.ReconstructBridgeSources("org", nil, ups, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 reconstructed source, got %d", len(got))
	}
	if got[0].TLSContextID != "" {
		t.Errorf("TLSContextID = %q, want empty for an upstream with no TLS context", got[0].TLSContextID)
	}
}

// TestMCPBridgeSchema_TLSContextIsOptional pins the schema contract: adding TLS must not
// become a required field on a resource that already shipped without it.
func TestMCPBridgeSchema_TLSContextIsOptional(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewMCPBridgeResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	sources, ok := resp.Schema.Attributes["source_apis"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("source_apis: expected ListNestedAttribute, got %T", resp.Schema.Attributes["source_apis"])
	}
	attr, ok := sources.NestedObject.Attributes["tls_context_id"].(schema.StringAttribute)
	if !ok {
		t.Fatal("source_apis.tls_context_id is missing from the schema")
	}
	if attr.Required {
		t.Error("tls_context_id must be Optional: plain-http backends do not need a TLS context")
	}
	if !attr.Optional {
		t.Error("tls_context_id must be Optional so it can be declared when needed")
	}
}
