package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func strPtr(s string) *string { return &s }

func TestNewAPIInstanceClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *client.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &client.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
			},
			wantErr: true,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
				tt.config.BaseURL = server.URL
			}
			_, err := NewAPIInstanceClient(tt.config)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestAPIInstanceClient_CRUD(t *testing.T) {
	mockInstance := &APIInstance{
		ID:           19876543,
		AssetID:      "myhealth-test",
		AssetVersion: "1.0.0",
		GroupID:      "test-org-id",
		Technology:   "flexGateway",
		Status:       "active",
		Spec: &APIInstanceSpec{
			AssetID: "myhealth-test",
			GroupID: "test-org-id",
			Version: "1.0.0",
		},
		Endpoint: &APIInstanceEndpoint{
			DeploymentType: "HY",
			Type:           "http",
			ProxyURI:       strPtr("http://0.0.0.0:8081/gw-test"),
		},
		Routing: []APIInstanceRoute{
			{
				Label: "gw-1-route-1",
				Upstreams: []APIInstanceUpstream{
					{Weight: 50, URI: "http://www.google.com", Label: "Google"},
					{Weight: 50, URI: "http://www.yahoo.com", Label: "Yahoo"},
				},
				Rules: &APIInstanceRules{Methods: "GET"},
			},
			{
				Label: "gw-1-route-2",
				Upstreams: []APIInstanceUpstream{
					{Weight: 100, URI: "http://www.google.com", Label: "Google1"},
				},
				Rules: &APIInstanceRules{Methods: "POST"},
			},
		},
		Deployment: &APIInstanceDeployment{
			EnvironmentID:  "test-env-id",
			Type:           "HY",
			ExpectedStatus: "deployed",
			TargetID:       "target-gw-id",
			TargetName:     "gw-1",
			GatewayVersion: "1.0.0",
		},
	}

	updatedInstance := *mockInstance
	updatedInstance.Routing = []APIInstanceRoute{
		{
			Label: "updated-route",
			Upstreams: []APIInstanceUpstream{
				{Weight: 100, URI: "http://www.example.com", Label: "Example"},
			},
			Rules: &APIInstanceRules{Methods: "GET,POST"},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.AssertHTTPRequest(t, r, "POST", "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis")
				testutil.JSONResponse(w, http.StatusCreated, mockInstance)
			} else {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis/19876543": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "PATCH":
				testutil.JSONResponse(w, http.StatusOK, &updatedInstance)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/19876543": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.JSONResponse(w, http.StatusOK, mockInstance)
			case "DELETE":
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	anypointClient, err := client.NewAnypointClient(&client.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create anypoint client: %v", err)
	}

	apiClient := &APIInstanceClient{AnypointClient: anypointClient}

	t.Run("Create", func(t *testing.T) {
		createReq := &CreateAPIInstanceRequest{
			Technology: "flexGateway",
			Spec: &APIInstanceSpec{
				AssetID: "myhealth-test",
				GroupID: "test-org-id",
				Version: "1.0.0",
			},
			Endpoint: &APIInstanceEndpoint{
				DeploymentType: "HY",
				Type:           "http",
				ProxyURI:       strPtr("http://0.0.0.0:8081/gw-test"),
			},
			Routing: []APIInstanceRoute{
				{
					Label: "gw-1-route-1",
					Upstreams: []APIInstanceUpstream{
						{Weight: 50, URI: "http://www.google.com", Label: "Google"},
						{Weight: 50, URI: "http://www.yahoo.com", Label: "Yahoo"},
					},
					Rules: &APIInstanceRules{Methods: "GET"},
				},
			},
			Deployment: &APIInstanceDeployment{
				EnvironmentID:  "test-env-id",
				Type:           "HY",
				ExpectedStatus: "deployed",
				TargetID:       "target-gw-id",
				TargetName:     "gw-1",
				GatewayVersion: "1.0.0",
			},
		}

		inst, err := apiClient.CreateAPIInstance(context.Background(), "test-org-id", "test-env-id", createReq)
		if err != nil {
			t.Fatalf("CreateAPIInstance() unexpected error: %v", err)
		}
		if inst.ID != 19876543 {
			t.Errorf("Expected ID 19876543, got %d", inst.ID)
		}
		if inst.Technology != "flexGateway" {
			t.Errorf("Expected Technology flexGateway, got %s", inst.Technology)
		}
		if len(inst.Routing) != 2 {
			t.Errorf("Expected 2 routes, got %d", len(inst.Routing))
		}
		if inst.Routing[0].Upstreams[0].Weight != 50 {
			t.Errorf("Expected first upstream weight 50, got %d", inst.Routing[0].Upstreams[0].Weight)
		}
	})

	t.Run("Read", func(t *testing.T) {
		inst, err := apiClient.GetAPIInstance(context.Background(), "test-org-id", "test-env-id", 19876543)
		if err != nil {
			t.Fatalf("GetAPIInstance() unexpected error: %v", err)
		}
		if inst.Spec.AssetID != "myhealth-test" {
			t.Errorf("Expected AssetID myhealth-test, got %s", inst.Spec.AssetID)
		}
		if inst.Deployment.TargetName != "gw-1" {
			t.Errorf("Expected TargetName gw-1, got %s", inst.Deployment.TargetName)
		}
	})

	t.Run("Update", func(t *testing.T) {
		updateReq := &UpdateAPIInstanceRequest{
			Routing: []APIInstanceRoute{
				{
					Label: "updated-route",
					Upstreams: []APIInstanceUpstream{
						{Weight: 100, URI: "http://www.example.com", Label: "Example"},
					},
					Rules: &APIInstanceRules{Methods: "GET,POST"},
				},
			},
		}

		inst, err := apiClient.UpdateAPIInstance(context.Background(), "test-org-id", "test-env-id", 19876543, updateReq)
		if err != nil {
			t.Fatalf("UpdateAPIInstance() unexpected error: %v", err)
		}
		if len(inst.Routing) != 1 {
			t.Errorf("Expected 1 route after update, got %d", len(inst.Routing))
		}
		if inst.Routing[0].Label != "updated-route" {
			t.Errorf("Expected route label 'updated-route', got %s", inst.Routing[0].Label)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := apiClient.DeleteAPIInstance(context.Background(), "test-org-id", "test-env-id", 19876543)
		if err != nil {
			t.Fatalf("DeleteAPIInstance() unexpected error: %v", err)
		}
	})
}

func TestAPIInstanceClient_ErrorHandling(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/99999": func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "API instance not found")
		},
		"/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid API spec")
			}
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	anypointClient, err := client.NewAnypointClient(&client.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	apiClient := &APIInstanceClient{AnypointClient: anypointClient}

	t.Run("NotFound", func(t *testing.T) {
		_, err := apiClient.GetAPIInstance(context.Background(), "test-org-id", "test-env-id", 99999)
		if err == nil {
			t.Error("Expected error for nonexistent API instance")
		}
	})

	t.Run("BadRequest", func(t *testing.T) {
		_, err := apiClient.CreateAPIInstance(context.Background(), "test-org-id", "test-env-id", &CreateAPIInstanceRequest{})
		if err == nil {
			t.Error("Expected error for invalid request")
		}
	})
}

func TestAPIInstance_JSONSerialization(t *testing.T) {
	inst := &APIInstance{
		ID:         12345,
		Technology: "flexGateway",
		Status:     "active",
		Spec:       &APIInstanceSpec{AssetID: "test", GroupID: "org-1", Version: "1.0.0"},
		Routing: []APIInstanceRoute{
			{
				Label: "route-1",
				Upstreams: []APIInstanceUpstream{
					{Weight: 60, URI: "http://backend1.example.com", Label: "Backend1"},
					{Weight: 40, URI: "http://backend2.example.com", Label: "Backend2"},
				},
				Rules: &APIInstanceRules{Methods: "GET,POST"},
			},
		},
	}

	data, err := json.Marshal(inst)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded APIInstance
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != inst.ID {
		t.Errorf("Expected ID %d, got %d", inst.ID, decoded.ID)
	}
	if len(decoded.Routing) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(decoded.Routing))
	}
	if len(decoded.Routing[0].Upstreams) != 2 {
		t.Errorf("Expected 2 upstreams, got %d", len(decoded.Routing[0].Upstreams))
	}
	if decoded.Routing[0].Upstreams[0].Weight != 60 {
		t.Errorf("Expected weight 60, got %d", decoded.Routing[0].Upstreams[0].Weight)
	}
	if decoded.Routing[0].Rules.Methods != "GET,POST" {
		t.Errorf("Expected methods GET,POST, got %s", decoded.Routing[0].Rules.Methods)
	}
}
