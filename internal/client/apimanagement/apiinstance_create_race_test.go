package apimanagement

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// createRaceClient serves POST /apis with the given per-attempt responses (status, body);
// the last entry repeats. It returns the client and the POST counter.
func createRaceClient(t *testing.T, responses [][2]string) (*APIInstanceClient, *int32) {
	t.Helper()
	old := createAPIInstanceRetryDelay
	createAPIInstanceRetryDelay = time.Millisecond
	t.Cleanup(func() { createAPIInstanceRetryDelay = old })

	var posts int32
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			n := int(atomic.AddInt32(&posts, 1)) - 1
			if n >= len(responses) {
				n = len(responses) - 1
			}
			status := http.StatusCreated
			if responses[n][0] == "409" {
				status = http.StatusConflict
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(responses[n][1]))
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
	return &APIInstanceClient{AnypointClient: anypointClient}, &posts
}

const raceBody = `{"name":"ConflictError","message":"This API (name) already exists in your organization."}`

// Live: terraform apply of the api_instance example (4 instances on one asset) lost 1-2
// creates to this 409; the same request succeeds once the asset's API record exists.
func TestCreateAPIInstance_RetriesAPINameRace(t *testing.T) {
	c, posts := createRaceClient(t, [][2]string{{"409", raceBody}, {"201", `{"id": 4242}`}})
	inst, err := c.CreateAPIInstance(context.Background(), "test-org-id", "test-env-id", &CreateAPIInstanceRequest{})
	if err != nil {
		t.Fatalf("expected the retry to succeed, got %v", err)
	}
	if inst.ID != 4242 {
		t.Errorf("id = %d, want 4242", inst.ID)
	}
	if got := atomic.LoadInt32(posts); got != 2 {
		t.Errorf("POSTs = %d, want 2", got)
	}
}

// Any other 409 is a real conflict: fail at once, no retry.
func TestCreateAPIInstance_OtherConflictNotRetried(t *testing.T) {
	c, posts := createRaceClient(t, [][2]string{{"409", `{"name":"ConflictError","message":"This instance label already exists in your organization."}`}})
	_, err := c.CreateAPIInstance(context.Background(), "test-org-id", "test-env-id", &CreateAPIInstanceRequest{})
	if err == nil || !strings.Contains(err.Error(), "409") {
		t.Fatalf("expected the 409 error, got %v", err)
	}
	if got := atomic.LoadInt32(posts); got != 1 {
		t.Errorf("POSTs = %d, want 1 (no retry)", got)
	}
}

// A race 409 that never clears gives up after the retry budget with the API's message.
func TestCreateAPIInstance_APINameRaceGivesUp(t *testing.T) {
	c, posts := createRaceClient(t, [][2]string{{"409", raceBody}})
	_, err := c.CreateAPIInstance(context.Background(), "test-org-id", "test-env-id", &CreateAPIInstanceRequest{})
	if err == nil || !strings.Contains(err.Error(), apiNameRaceMessage) {
		t.Fatalf("expected the race error after retries, got %v", err)
	}
	if got := atomic.LoadInt32(posts); got != 6 {
		t.Errorf("POSTs = %d, want 6 (1 + 5 retries)", got)
	}
}
