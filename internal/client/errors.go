package client

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrNotFound is the sentinel that all 404/missing-resource errors wrap.
var ErrNotFound = errors.New("resource not found")

// NotFoundError carries HTTP status and a human-readable message while
// satisfying errors.Is(err, ErrNotFound).
type NotFoundError struct {
	StatusCode int
	Message    string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// NewNotFoundError constructs a NotFoundError for the given resource
// description (e.g. "private space", "API instance").
func NewNotFoundError(resourceDesc string) *NotFoundError {
	return &NotFoundError{
		StatusCode: http.StatusNotFound,
		Message:    fmt.Sprintf("%s not found", resourceDesc),
	}
}

// IsNotFound returns true when err (or any error in its chain) is a
// NotFoundError or the ErrNotFound sentinel.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrNotFound)
}
