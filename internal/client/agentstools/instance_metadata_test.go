package agentstools

import (
	"encoding/json"
	"testing"
)

// The bridge LIST endpoint returns every API Manager instance in the environment,
// not just bridges. Other instance types carry structured metadata — an LLM proxy
// stores a nested globalRouting object — and decoding that into map[string]string
// used to fail the whole listing, making anypoint_mcp_bridges unusable in any
// environment that also had an LLM proxy.
func TestInstanceMetadata_SkipsNonStringValues(t *testing.T) {
	// Shape captured from a live environment.
	raw := []byte(`{
		"generatedBy": "mcp_bridge",
		"globalRouting": {"llmConfigs": {"routingType": "model-based"}}
	}`)

	var m InstanceMetadata
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("a nested metadata value must not fail the decode: %v", err)
	}
	if m["generatedBy"] != "mcp_bridge" {
		t.Errorf("generatedBy = %q, want mcp_bridge — the marker must survive", m["generatedBy"])
	}
	if _, present := m["globalRouting"]; present {
		t.Error("a non-string value cannot be the marker; it should be skipped, not coerced")
	}
}

// The listing must survive an instance whose metadata is entirely structured.
func TestInstanceMetadata_AllNonStringDecodesEmpty(t *testing.T) {
	var m InstanceMetadata
	if err := json.Unmarshal([]byte(`{"a": {"x": 1}, "b": [1,2]}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected no usable entries, got %v", m)
	}
}

func TestInstanceMetadata_KeepsPlainStrings(t *testing.T) {
	var m InstanceMetadata
	if err := json.Unmarshal([]byte(`{"generatedBy":"mcp_bridge","owner":"platform"}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["generatedBy"] != "mcp_bridge" || m["owner"] != "platform" {
		t.Errorf("string values must round-trip, got %v", m)
	}
}

// Malformed metadata is still an error — only the value types are tolerated.
func TestInstanceMetadata_RejectsNonObject(t *testing.T) {
	var m InstanceMetadata
	if err := json.Unmarshal([]byte(`"not-an-object"`), &m); err == nil {
		t.Error("metadata that is not a JSON object should still fail to decode")
	}
}

// A bridge decoded out of a listing that contains a foreign instance must still be
// recognised as a bridge.
func TestMCPBridge_DecodesAlongsideForeignMetadata(t *testing.T) {
	raw := []byte(`{
		"id": 42,
		"assetId": "orders-mcp",
		"metadata": {"generatedBy": "mcp_bridge", "globalRouting": {"llmConfigs": {}}}
	}`)

	var b MCPBridge
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b.Metadata["generatedBy"] != "mcp_bridge" {
		t.Errorf("bridge marker lost: %v", b.Metadata)
	}
	if b.ID != 42 {
		t.Errorf("ID = %d, want 42", b.ID)
	}
}
