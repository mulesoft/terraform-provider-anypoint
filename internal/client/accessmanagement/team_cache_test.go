package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// TestListTeams_CacheDroppedOnTeamWrite: team Create lists teams first (root team
// lookup), which caches the list for the rest of the run. Without invalidation an
// anypoint_teams data source read later in the same apply misses the new team.
func TestListTeams_CacheDroppedOnTeamWrite(t *testing.T) {
	var gets int32
	names := []string{"root"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			names = append(names, "new")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"team_id":"t2","team_name":"new"}`))
			return
		}
		atomic.AddInt32(&gets, 1)
		var data []map[string]interface{}
		for _, n := range names {
			data = append(data, map[string]interface{}{"team_id": n, "team_name": n})
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": data, "total": len(data)})
	}))
	defer server.Close()
	c := &TeamClient{AnypointClient: &client.AnypointClient{
		BaseURL: server.URL, Token: "t", HTTPClient: &http.Client{}, Cache: client.NewResponseCache(),
	}}
	ctx := context.Background()

	first, err := c.ListTeams(ctx, "org")
	if err != nil || len(first) != 1 {
		t.Fatalf("first list: %v %v", first, err)
	}
	if _, err := c.ListTeams(ctx, "org"); err != nil || atomic.LoadInt32(&gets) != 1 {
		t.Fatalf("second list should be served from cache, gets=%d err=%v", gets, err)
	}
	if _, err := c.CreateTeam(ctx, "org", &CreateTeamRequest{TeamName: "new"}); err != nil {
		t.Fatal(err)
	}
	after, err := c.ListTeams(ctx, "org")
	if err != nil || len(after) != 2 {
		t.Fatalf("list after create must include the new team, got %v (err %v, gets=%d)", after, err, gets)
	}
}
