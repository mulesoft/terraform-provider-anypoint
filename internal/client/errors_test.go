package client

import (
	"fmt"
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
