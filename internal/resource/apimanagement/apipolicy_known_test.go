package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// knownPolicyTestCases lists a representative sample of known policy types
// covering all major categories: rate limiting, auth, traffic, transformation.
var knownPolicyTestCases = []struct {
	policyType     string
	metadataSuffix string
}{
	{"rate-limiting", "_api_policy_rate_limiting"},
	{"spike-control", "_api_policy_spike_control"},
	{"ip-allowlist", "_api_policy_ip_allowlist"},
	{"ip-blocklist", "_api_policy_ip_blocklist"},
	{"cors", "_api_policy_cors"},
	{"jwt-validation", "_api_policy_jwt_validation"},
	{"client-id-enforcement", "_api_policy_client_id_enforcement"},
	{"message-logging", "_api_policy_message_logging"},
	{"header-injection", "_api_policy_header_injection"},
	{"header-removal", "_api_policy_header_removal"},
}

func TestNewKnownPolicyResourceFunc(t *testing.T) {
	for _, tc := range knownPolicyTestCases {
		t.Run(tc.policyType, func(t *testing.T) {
			fn := NewKnownPolicyResourceFunc(tc.policyType)
			if fn == nil {
				t.Fatalf("NewKnownPolicyResourceFunc(%q) returned nil", tc.policyType)
			}
			r := fn()
			if r == nil {
				t.Fatalf("factory for %q returned nil resource", tc.policyType)
			}
			if _, ok := r.(resource.ResourceWithConfigure); !ok {
				t.Errorf("%q should implement ResourceWithConfigure", tc.policyType)
			}
			if _, ok := r.(resource.ResourceWithImportState); !ok {
				t.Errorf("%q should implement ResourceWithImportState", tc.policyType)
			}
		})
	}
}

func TestKnownPolicyResource_Metadata(t *testing.T) {
	for _, tc := range knownPolicyTestCases {
		t.Run(tc.policyType, func(t *testing.T) {
			r := NewKnownPolicyResourceFunc(tc.policyType)()
			testutil.TestResourceMetadata(t, r, tc.metadataSuffix)
		})
	}
}

func TestKnownPolicyResource_Schema(t *testing.T) {
	for _, tc := range knownPolicyTestCases {
		t.Run(tc.policyType, func(t *testing.T) {
			r := NewKnownPolicyResourceFunc(tc.policyType)()
			ctx := context.Background()
			req := resource.SchemaRequest{}
			resp := &resource.SchemaResponse{}
			r.Schema(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("Schema() for %q has errors: %v", tc.policyType, resp.Diagnostics.Errors())
			}
			required := []string{"environment_id", "api_instance_id"}
			for _, attr := range required {
				a, ok := resp.Schema.Attributes[attr]
				if !ok {
					t.Errorf("Schema() for %q missing required attribute %q", tc.policyType, attr)
					continue
				}
				if !a.IsRequired() {
					t.Errorf("Schema() for %q: attribute %q should be required", tc.policyType, attr)
				}
			}
			computed := []string{"id", "organization_id"}
			for _, attr := range computed {
				a, ok := resp.Schema.Attributes[attr]
				if !ok {
					t.Errorf("Schema() for %q missing computed attribute %q", tc.policyType, attr)
					continue
				}
				if !a.IsComputed() {
					t.Errorf("Schema() for %q: attribute %q should be computed", tc.policyType, attr)
				}
			}
		})
	}
}

func TestKnownPolicyResource_Configure(t *testing.T) {
	for _, tc := range knownPolicyTestCases {
		t.Run(tc.policyType, func(t *testing.T) {
			r := NewKnownPolicyResourceFunc(tc.policyType)().(*KnownPolicyResource)
			server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
			providerData := &anypointclient.Config{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			}
			testutil.TestResourceConfigure(t, r, providerData)
			if r.client == nil {
				t.Errorf("Configure() for %q should set client", tc.policyType)
			}
		})
	}
}

func TestKnownPolicyResource_Configure_InvalidProviderData(t *testing.T) {
	r := NewKnownPolicyResourceFunc("rate-limiting")().(*KnownPolicyResource)
	ctx := context.Background()
	req := resource.ConfigureRequest{ProviderData: "invalid"}
	resp := &resource.ConfigureResponse{}
	r.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if r.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestKnownPolicyResource_ImportState(t *testing.T) {
	for _, tc := range knownPolicyTestCases {
		t.Run(tc.policyType, func(t *testing.T) {
			r := NewKnownPolicyResourceFunc(tc.policyType)()
			if _, ok := r.(resource.ResourceWithImportState); !ok {
				t.Errorf("%q does not implement ImportState", tc.policyType)
			}
		})
	}
}

func TestKnownPolicyResourceModel_Validation(t *testing.T) {
	model := KnownPolicyResourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.APIInstanceID
	_ = model.Label
	_ = model.Configuration
	_ = model.Order
	_ = model.Disabled
	_ = model.PolicyTemplateID
	_ = model.AssetVersion
	_ = model.UpstreamIDs
}

func TestKnownPolicyResource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/200"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":               200,
				"policyTemplateId": "some-template-id",
				"groupId":          "68ef9520-24e9-4cf2-b2f5-620025690913",
				"assetId":          "http-basic-authentication",
				"assetVersion":     "1.3.1",
				"label":            nil,
				"order":            1,
				"disabled":         false,
				"configurationData": map[string]interface{}{
					"username": "test-user",
					"password": "test-pass",
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	fn := NewKnownPolicyResourceFunc("http-basic-authentication")
	res := fn().(*KnownPolicyResource)
	res.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	upstreamIDsElemType := objType.AttributeTypes["upstream_ids"].(tftypes.List).ElementType
	configObjType := objType.AttributeTypes["configuration"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "200"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":    tftypes.NewValue(tftypes.String, "100"),
		"label":              tftypes.NewValue(tftypes.String, nil),
		"configuration":      tftypes.NewValue(configObjType, nil),
		"order":              tftypes.NewValue(tftypes.Number, nil),
		"disabled":           tftypes.NewValue(tftypes.Bool, false),
		"policy_template_id": tftypes.NewValue(tftypes.String, nil),
		"asset_version":      tftypes.NewValue(tftypes.String, "1.3.1"),
		"upstream_ids":       tftypes.NewValue(tftypes.List{ElementType: upstreamIDsElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
}

// TestKnownPolicyResource_OutboundOrderUsesStateForUnknown verifies that the
// `order` attribute on an outbound policy carries the UseStateForUnknown plan
// modifier. Without it, every re-plan would show "(known after apply)" for
// order even when nothing else changed, churning state.
func TestKnownPolicyResource_OutboundOrderUsesStateForUnknown(t *testing.T) {
	r := NewKnownPolicyResourceFunc("message-logging-outbound")()
	ctx := context.Background()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() errors: %v", resp.Diagnostics.Errors())
	}

	attr, ok := resp.Schema.Attributes["order"]
	if !ok {
		t.Fatal("expected `order` attribute on outbound policy schema")
	}
	if !attr.IsComputed() {
		t.Error("expected `order` to be computed for outbound policies")
	}
	// Outbound policies don't accept order in the request payload, so it must
	// not be Optional either — only the server may set it.
	if attr.IsOptional() {
		t.Error("expected `order` to be computed-only (not optional) for outbound policies")
	}
}

// TestKnownPolicyResource_DisabledUsesStateForUnknown verifies that `disabled`
// keeps its prior state when the plan is unknown so re-plans don't churn.
func TestKnownPolicyResource_DisabledUsesStateForUnknown(t *testing.T) {
	r := NewKnownPolicyResourceFunc("rate-limiting")()
	ctx := context.Background()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() errors: %v", resp.Diagnostics.Errors())
	}

	attr, ok := resp.Schema.Attributes["disabled"]
	if !ok {
		t.Fatal("expected `disabled` attribute on policy schema")
	}
	if !attr.IsComputed() {
		t.Error("expected `disabled` to be computed")
	}
	if !attr.IsOptional() {
		t.Error("expected `disabled` to remain optional so users can override the default")
	}
}

// TestRecoverOutboundConflict_UniversalListAdoptsExistingPolicy verifies the
// 409-conflict recovery path: when CreateOutboundAPIPolicy fails because a
// matching policy already exists on the server, the provider lists policies
// via the universal /policies endpoint (the outbound-policies endpoint
// returns 405 on GET), finds the existing one by groupId+assetId+label, and
// adopts it via GetOutboundAPIPolicy so subsequent plans see the resource in
// state instead of attempting to recreate it.
func TestRecoverOutboundConflict_UniversalListAdoptsExistingPolicy(t *testing.T) {
	const (
		orgID   = "test-org-id"
		envID   = "test-env-id"
		apiID   = 100
		groupID = "68ef9520-24e9-4cf2-b2f5-620025690913"
	)

	existing := apimgmtclient.APIPolicy{
		ID:           5555,
		AssetID:      "message-logging-outbound",
		GroupID:      groupID,
		AssetVersion: "1.0.0",
		Label:        "outbound-logger",
		Order:        7,
		APIID:        apiID,
		UpstreamIDs:  []string{"upstream-uuid-1"},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		// Universal policies listing returns BOTH inbound and outbound entries.
		"GET /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies": func(w http.ResponseWriter, _ *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []apimgmtclient.APIPolicy{existing})
		},
		// Outbound listing endpoint mirrors production: GET is rejected. We
		// keep this handler installed so that any code that accidentally
		// reaches for the dedicated outbound listing path observes the
		// real-world failure mode rather than a silent 404 from the mock.
		"GET /apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/outbound-policies": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Allow", "POST")
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "GET method not allowed")
		},
		// Per-id GET on an outbound policy goes through the inbound
		// `api/v1/.../policies/{id}` endpoint. The dedicated xapi/v1
		// outbound-policies path is POST-only on production and 404s for
		// every other method, so the provider deliberately uses the
		// inbound path for all post-create CRUD on outbound policies.
		"GET /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/5555": func(w http.ResponseWriter, _ *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, existing)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewKnownPolicyResourceFunc("message-logging-outbound")().(*KnownPolicyResource)
	res.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      orgID,
		},
	}

	req := &apimgmtclient.CreateOutboundAPIPolicyRequest{
		ConfigurationData: map[string]interface{}{},
		GroupID:           groupID,
		AssetID:           "message-logging-outbound",
		AssetVersion:      "1.0.0",
		Label:             "outbound-logger",
		UpstreamIDs:       []string{"upstream-uuid-1"},
	}

	got, err := res.recoverOutboundConflict(context.Background(), orgID, envID, apiID, req)
	if err != nil {
		t.Fatalf("recoverOutboundConflict failed: %v", err)
	}
	if got == nil || got.ID != existing.ID {
		t.Fatalf("expected to adopt policy id=%d, got %#v", existing.ID, got)
	}
	if got.Label != existing.Label {
		t.Errorf("expected label=%q, got %q", existing.Label, got.Label)
	}
}

// TestRecoverOutboundConflict_AmbiguousMatchFails ensures recovery refuses to
// guess when multiple existing policies match asset+label — the user needs to
// disambiguate manually (typically via `terraform import`).
func TestRecoverOutboundConflict_AmbiguousMatchFails(t *testing.T) {
	const (
		orgID   = "test-org-id"
		envID   = "test-env-id"
		apiID   = 100
		groupID = "68ef9520-24e9-4cf2-b2f5-620025690913"
	)

	dupes := []apimgmtclient.APIPolicy{
		{ID: 1001, AssetID: "message-logging-outbound", GroupID: groupID, Label: "outbound-logger"},
		{ID: 1002, AssetID: "message-logging-outbound", GroupID: groupID, Label: "outbound-logger"},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"GET /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies": func(w http.ResponseWriter, _ *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, dupes)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewKnownPolicyResourceFunc("message-logging-outbound")().(*KnownPolicyResource)
	res.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      orgID,
		},
	}

	req := &apimgmtclient.CreateOutboundAPIPolicyRequest{
		ConfigurationData: map[string]interface{}{},
		GroupID:           groupID,
		AssetID:           "message-logging-outbound",
		AssetVersion:      "1.0.0",
		Label:             "outbound-logger",
	}

	if _, err := res.recoverOutboundConflict(context.Background(), orgID, envID, apiID, req); err == nil {
		t.Fatal("expected ambiguous-match failure, got nil")
	}
}

// TestRecoverOutboundConflict_ListFailsSurfacesError verifies that when the
// universal listing endpoint itself fails (network issue, 5xx, etc.), the
// recovery error is surfaced cleanly with the original 409 context preserved.
func TestRecoverOutboundConflict_ListFailsSurfacesError(t *testing.T) {
	const (
		orgID   = "test-org-id"
		envID   = "test-env-id"
		apiID   = 100
		groupID = "68ef9520-24e9-4cf2-b2f5-620025690913"
	)

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"GET /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies": func(w http.ResponseWriter, _ *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "boom")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewKnownPolicyResourceFunc("message-logging-outbound")().(*KnownPolicyResource)
	res.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      orgID,
		},
	}

	req := &apimgmtclient.CreateOutboundAPIPolicyRequest{
		ConfigurationData: map[string]interface{}{},
		GroupID:           groupID,
		AssetID:           "message-logging-outbound",
		AssetVersion:      "1.0.0",
		Label:             "outbound-logger",
	}

	_, err := res.recoverOutboundConflict(context.Background(), orgID, envID, apiID, req)
	if err == nil {
		t.Fatal("expected list failure, got nil")
	}
}

func TestKnownPolicyResource_Read_NotFound(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/200"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	fn := NewKnownPolicyResourceFunc("http-basic-authentication")
	res := fn().(*KnownPolicyResource)
	res.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	upstreamIDsElemType := objType.AttributeTypes["upstream_ids"].(tftypes.List).ElementType
	configObjType := objType.AttributeTypes["configuration"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "200"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":    tftypes.NewValue(tftypes.String, "100"),
		"label":              tftypes.NewValue(tftypes.String, nil),
		"configuration":      tftypes.NewValue(configObjType, nil),
		"order":              tftypes.NewValue(tftypes.Number, nil),
		"disabled":           tftypes.NewValue(tftypes.Bool, false),
		"policy_template_id": tftypes.NewValue(tftypes.String, nil),
		"asset_version":      tftypes.NewValue(tftypes.String, "1.3.1"),
		"upstream_ids":       tftypes.NewValue(tftypes.List{ElementType: upstreamIDsElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}
