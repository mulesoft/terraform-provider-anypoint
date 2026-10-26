package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// opaqueTokenPlane models the control planes where the OAuth2 password grant succeeds
// but hands back a token that cannot read /accounts/api/me. Only the token minted by
// /accounts/login carries user context. This is the shape observed live on staging.
func opaqueTokenPlane(t *testing.T, hits *[]string) *httptest.Server {
	t.Helper()
	const loginToken = "login-token"

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits = append(*hits, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/accounts/api/v2/oauth2/token":
			_, _ = w.Write([]byte(`{"access_token":"opaque-token"}`))

		case r.URL.Path == "/accounts/login":
			_, _ = w.Write([]byte(`{"access_token":"` + loginToken + `"}`))

		case r.URL.Path == "/accounts/api/me":
			if r.Header.Get("Authorization") != "Bearer "+loginToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"user":{"organization":{"id":"org-from-login"}}}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Without the fallback every area built on AnypointClient is unusable with
// auth_type = "user" on these control planes — including the team and connected-app
// operations the documentation says require user auth.
func TestAuthenticate_UserAuthFallsBackToLoginWhenTokenLacksUserContext(t *testing.T) {
	var hits []string
	srv := opaqueTokenPlane(t, &hits)
	defer srv.Close()

	c, err := NewAnypointClient(&Config{
		BaseURL:      srv.URL,
		ClientID:     "cid",
		ClientSecret: "csec",
		Username:     "admin",
		Password:     "pw",
	})
	if err != nil {
		t.Fatalf("user authentication must succeed via the login fallback: %v", err)
	}
	if c.OrgID != "org-from-login" {
		t.Errorf("OrgID = %q, want org-from-login", c.OrgID)
	}
	if c.Token != "login-token" {
		t.Errorf("Token = %q, want the user-context token from /accounts/login", c.Token)
	}

	var sawLogin bool
	for _, p := range hits {
		if p == "/accounts/login" {
			sawLogin = true
		}
	}
	if !sawLogin {
		t.Errorf("expected a call to /accounts/login, got %v", hits)
	}
}

// The happy path must not regress into an extra round trip: when the password-grant
// token can already read /accounts/api/me, the login endpoint is never touched.
func TestAuthenticate_UserAuthSkipsFallbackWhenTokenWorks(t *testing.T) {
	var hits []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/accounts/api/v2/oauth2/token":
			_, _ = w.Write([]byte(`{"access_token":"jwt-token"}`))
		case "/accounts/api/me":
			_, _ = w.Write([]byte(`{"user":{"organization":{"id":"org-from-jwt"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewAnypointClient(&Config{
		BaseURL: srv.URL, ClientID: "cid", ClientSecret: "csec",
		Username: "admin", Password: "pw",
	})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if c.OrgID != "org-from-jwt" {
		t.Errorf("OrgID = %q, want org-from-jwt", c.OrgID)
	}
	for _, p := range hits {
		if p == "/accounts/login" {
			t.Error("the login fallback must not run when the password-grant token already works")
		}
	}
}

// A connected app acting on its own behalf has no user to log in as, so the fallback
// must not fire for client_credentials — it would send an empty username/password.
func TestAuthenticate_ClientCredentialsNeverCallsLogin(t *testing.T) {
	var hits []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/accounts/api/v2/oauth2/token":
			body, _ := json.Marshal(map[string]string{"access_token": "cc-token"})
			_, _ = w.Write(body)
		case "/accounts/api/me":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_, err := NewAnypointClient(&Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "csec"})
	if err == nil {
		t.Fatal("client_credentials with an unusable token must still fail rather than silently recover")
	}
	if !strings.Contains(err.Error(), "user info") {
		t.Errorf("error should point at the identity lookup, got: %v", err)
	}
	for _, p := range hits {
		if p == "/accounts/login" {
			t.Error("client_credentials has no user credentials; it must never hit /accounts/login")
		}
	}
}

// When both the grant and the fallback fail the practitioner needs to see both causes,
// not just the last one.
func TestAuthenticate_ReportsBothFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/accounts/api/v2/oauth2/token":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	_, err := NewAnypointClient(&Config{
		BaseURL: srv.URL, ClientID: "cid", ClientSecret: "csec",
		Username: "admin", Password: "pw",
	})
	if err == nil {
		t.Fatal("expected authentication to fail")
	}
	msg := err.Error()
	if !strings.Contains(msg, "invalid_grant") || !strings.Contains(msg, "login") {
		t.Errorf("error must name both the grant failure and the fallback failure, got: %v", err)
	}
}
