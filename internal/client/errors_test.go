package client

import (
	"fmt"
	"strings"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "NewNotFoundError", err: NewNotFoundError("my resource"), want: true},
		{name: "wrapped NewNotFoundError", err: fmt.Errorf("operation failed: %w", NewNotFoundError("my resource")), want: true},
		{name: "ErrNotFound sentinel directly", err: ErrNotFound, want: true},
		{name: "wrapped ErrNotFound sentinel", err: fmt.Errorf("wrap: %w", ErrNotFound), want: true},
		{name: "plain string containing 'not found' must NOT match", err: fmt.Errorf("resource not found"), want: false},
		{name: "unrelated error", err: fmt.Errorf("internal server error"), want: false},
		{name: "empty error", err: fmt.Errorf(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestNotFoundErrorMessage(t *testing.T) {
	err := NewNotFoundError("API instance")
	if err.Error() != "API instance not found" {
		t.Errorf("unexpected message: %s", err.Error())
	}
	if err.StatusCode != 404 {
		t.Errorf("unexpected status code: %d", err.StatusCode)
	}
}

func TestAuthContextErrorIfUnauthorized(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantAuth   bool
	}{
		{name: "401 -> auth-context error", statusCode: 401, wantAuth: true},
		{name: "403 -> auth-context error", statusCode: 403, wantAuth: true},
		{name: "404 -> nil (not an auth problem)", statusCode: 404, wantAuth: false},
		{name: "409 -> nil", statusCode: 409, wantAuth: false},
		{name: "500 -> nil", statusCode: 500, wantAuth: false},
		{name: "200 -> nil", statusCode: 200, wantAuth: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AuthContextErrorIfUnauthorized(tt.statusCode, "https://x/gatewaymanager/...", "forbidden")
			if tt.wantAuth {
				if err == nil {
					t.Fatalf("expected an AuthContextError for status %d, got nil", tt.statusCode)
				}
				if !IsAuthContext(err) {
					t.Errorf("IsAuthContext = false for status %d, want true", tt.statusCode)
				}
				// The message must be actionable: point at the real fix (grant the
				// required connected-app scopes), not just "switch auth mode".
				msg := err.Error()
				if !strings.Contains(msg, "scopes") || !strings.Contains(msg, "Manage Servers") {
					t.Errorf("message missing scope guidance: %s", msg)
				}
			} else if err != nil {
				t.Errorf("expected nil for status %d, got %v", tt.statusCode, err)
			}
		})
	}
}

func TestIsAuthContext(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "NewAuthContextError", err: NewAuthContextError(403, "ep", "body"), want: true},
		{name: "wrapped AuthContextError", err: fmt.Errorf("op failed: %w", NewAuthContextError(401, "ep", "body")), want: true},
		{name: "ErrAuthContext sentinel", err: ErrAuthContext, want: true},
		{name: "unrelated error", err: fmt.Errorf("boom"), want: false},
		{name: "NotFound is not AuthContext", err: NewNotFoundError("x"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthContext(tt.err); got != tt.want {
				t.Errorf("IsAuthContext(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
