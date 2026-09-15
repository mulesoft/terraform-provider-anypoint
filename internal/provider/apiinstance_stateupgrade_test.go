package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// apiInstanceStateBeforePortAttribute is a real anypoint_api_instance state
// object captured from a production import performed by the provider build that
// predates the endpoint.port attribute (identifiers substituted). Its endpoint
// object has exactly four keys — base_path, deployment_type, response_timeout,
// type — and no port.
//
// base_path holds the full proxy URI rather than a path because the pre-port
// build recovered it with strings.TrimPrefix(proxyURI, "http://0.0.0.0:8081/"),
// which silently matched nothing for an instance listening on any other port.
// That is the state real users are upgrading from, so it is what the upgrade
// path has to survive.
const apiInstanceStateBeforePortAttribute = `{
  "approval_method": null,
  "asset_id": "tf-mcp-approach-d",
  "asset_version": "1.0.1",
  "consumer_endpoint": null,
  "deployment": {
    "environment_id": "00000000-0000-0000-0000-0000000000env",
    "expected_status": "deployed",
    "gateway_version": "1.13.3",
    "overwrite": false,
    "target_id": "00000000-0000-0000-0000-000000000gtw",
    "target_name": "tf-bridge-test",
    "type": "HY"
  },
  "endpoint": {
    "base_path": "http://0.0.0.0:8088/approach-d",
    "deployment_type": "HY",
    "response_timeout": null,
    "type": "mcp"
  },
  "environment_id": "00000000-0000-0000-0000-0000000000env",
  "gateway_id": "00000000-0000-0000-0000-000000000gtw",
  "id": "21074197",
  "instance_label": null,
  "organization_id": "00000000-0000-0000-0000-0000000000org",
  "product_version": "v1.0",
  "provider_id": null,
  "routing": [
    {
      "label": "tf-fresh-rest-api",
      "rules": {
        "headers": {
          "X-UPSTREAM-NAME": "tf-fresh-rest-api"
        },
        "host": null,
        "methods": null,
        "path": null
      },
      "upstreams": [
        {
          "label": null,
          "tls_context_id": null,
          "uri": "https://sandbox.example.com/petstore/v1",
          "weight": 100
        }
      ]
    }
  ],
  "spec": {
    "asset_id": "tf-mcp-approach-d",
    "group_id": "00000000-0000-0000-0000-0000000000org",
    "version": "1.0.1"
  },
  "status": "inactive",
  "technology": "omniGateway",
  "upstream_uri": null
}`

// TestAPIInstance_UpgradeStateWrittenBeforePortAttribute pins the provider
// upgrade path for anypoint_api_instance.
//
// endpoint.port was added to an existing, widely-used resource without bumping
// the schema version, so every user upgrading the provider hands Terraform a
// state object whose nested endpoint is missing an attribute the current schema
// declares. If the framework rejected that instead of filling the gap with null,
// the resource would be unreadable after upgrade and could only be recovered by
// hand-editing state — so this asserts decoding succeeds rather than assuming it.
func TestAPIInstance_UpgradeStateWrittenBeforePortAttribute(t *testing.T) {
	ctx := context.Background()

	server, err := TestAccProtoV6ProviderFactories["anypoint"]()
	if err != nil {
		t.Fatalf("could not build provider server: %s", err)
	}

	schemaResp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %s", err)
	}
	resourceSchema, ok := schemaResp.ResourceSchemas["anypoint_api_instance"]
	if !ok {
		t.Fatal("anypoint_api_instance is not registered")
	}
	schemaType := resourceSchema.ValueType()

	// A schema version bump would need a matching UpgradeState implementation;
	// this test describes the version-0-to-version-0 path that ships today.
	if resourceSchema.Version != 0 {
		t.Fatalf("schema version is %d, not 0: this test and the upgrade story both "+
			"assume no version bump, so a bump needs an UpgradeState implementation",
			resourceSchema.Version)
	}

	upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "anypoint_api_instance",
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: []byte(apiInstanceStateBeforePortAttribute)},
	})
	if err != nil {
		t.Fatalf("UpgradeResourceState returned a transport error: %s", err)
	}
	for _, d := range upgraded.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("state written before endpoint.port existed failed to upgrade: %s: %s",
				d.Summary, d.Detail)
		}
	}

	value, err := upgraded.UpgradedState.Unmarshal(schemaType)
	if err != nil {
		t.Fatalf("upgraded state does not conform to the current schema: %s", err)
	}

	var attrs map[string]tftypes.Value
	if err := value.As(&attrs); err != nil {
		t.Fatalf("decoding upgraded state: %s", err)
	}
	var endpoint map[string]tftypes.Value
	if err := attrs["endpoint"].As(&endpoint); err != nil {
		t.Fatalf("decoding endpoint: %s", err)
	}

	port, ok := endpoint["port"]
	if !ok {
		t.Fatal("endpoint.port is absent from the upgraded state; the schema declares it")
	}
	// Null is the correct outcome: the attribute is Computed with a default, so
	// the value is resolved during the next plan/refresh rather than invented by
	// the upgrade. Asserting null (not 8081) keeps this test honest about which
	// layer supplies the default.
	if !port.IsNull() {
		t.Errorf("expected endpoint.port to be null after upgrading pre-port state, got %s", port)
	}

	// The pre-port base_path is carried through untouched. Correcting it is the
	// job of the next Read, which parses the proxy URI properly; the upgrade must
	// not silently rewrite state behind the user's back.
	var basePath string
	if err := endpoint["base_path"].As(&basePath); err != nil {
		t.Fatalf("decoding endpoint.base_path: %s", err)
	}
	if want := "http://0.0.0.0:8088/approach-d"; basePath != want {
		t.Errorf("expected upgrade to preserve base_path %q, got %q", want, basePath)
	}
}
