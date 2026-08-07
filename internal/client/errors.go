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

// ErrConflict is the sentinel that all 409/already-exists errors wrap.
var ErrConflict = errors.New("resource already exists")

// ConflictError carries HTTP status and a human-readable message while
// satisfying errors.Is(err, ErrConflict).
type ConflictError struct {
	StatusCode int
	Message    string
}

func (e *ConflictError) Error() string {
	return e.Message
}

func (e *ConflictError) Is(target error) bool {
	return target == ErrConflict
}

// NewConflictError constructs a ConflictError for the given resource description.
func NewConflictError(resourceDesc string) *ConflictError {
	return &ConflictError{
		StatusCode: http.StatusConflict,
		Message:    fmt.Sprintf("%s already exists", resourceDesc),
	}
}

// IsConflict returns true when err (or any error in its chain) is a
// ConflictError or the ErrConflict sentinel.
func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrConflict)
}

// ErrAuthContext is the sentinel that all "insufficient authorization" (401/403
// from a control-plane API) errors wrap.
var ErrAuthContext = errors.New("insufficient authorization for control-plane API")

// AuthContextError carries an HTTP 401/403 returned by a control-plane API
// (Gateway Manager, self-managed / standalone gateways, CloudHub 2.0 private
// spaces / Runtime Manager) together with actionable guidance.
//
// These APIs DO accept a client-credentials connected-app token — but only when
// the connected app has been granted the scopes the endpoint requires. A 401/403
// here almost always means the token's principal is missing those scopes (verified
// live on devx: the same endpoints return 200 once the app is granted, e.g.,
// "Manage Servers" + "Read Servers" for Gateway Manager, or "Cloudhub Organization
// Admin" for private spaces). It does NOT mean user-context (auth_type = "user") is
// mandatory — a user token merely tends to work because a human admin already holds
// those permissions. Surfacing this as a distinct, self-explaining error turns an
// opaque 401/403 into a clear "grant these scopes" action.
type AuthContextError struct {
	StatusCode int
	Endpoint   string
	Body       string
}

func (e *AuthContextError) Error() string {
	return fmt.Sprintf(
		"received HTTP %d from control-plane API %s. The authenticating principal is missing "+
			"the scopes/permissions this endpoint requires. If you use a connected app "+
			"(client_credentials), grant it the required scopes: Gateway Manager / self-managed "+
			"gateways need 'Manage Servers', 'Read Servers', 'View Organization' (plus API Manager "+
			"scopes such as 'Manage APIs Configuration', 'Manage Policies', 'Deploy API Proxies' for "+
			"Omni Gateway); CloudHub 2.0 private spaces need 'Cloudhub Organization Admin'. "+
			"Alternatively use auth_type = \"user\" with a user that already has these permissions. "+
			"Server response: %s",
		e.StatusCode, e.Endpoint, e.Body)
}

func (e *AuthContextError) Is(target error) bool {
	return target == ErrAuthContext
}

// NewAuthContextError constructs an AuthContextError for the given endpoint/body.
func NewAuthContextError(statusCode int, endpoint, body string) *AuthContextError {
	return &AuthContextError{StatusCode: statusCode, Endpoint: endpoint, Body: body}
}

// AuthContextErrorIfUnauthorized returns an *AuthContextError when statusCode is
// 401 (Unauthorized) or 403 (Forbidden), otherwise nil. Control-plane call sites
// use it in their non-OK branch to convert a bare auth rejection into actionable
// "grant the required scopes" guidance before falling back to a generic status error:
//
//	if resp.StatusCode != http.StatusOK {
//	    body, _ := io.ReadAll(resp.Body)
//	    if authErr := client.AuthContextErrorIfUnauthorized(resp.StatusCode, url, string(body)); authErr != nil {
//	        return nil, authErr
//	    }
//	    return nil, fmt.Errorf("failed to ... with status %d: %s", resp.StatusCode, string(body))
//	}
func AuthContextErrorIfUnauthorized(statusCode int, endpoint, body string) error {
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return NewAuthContextError(statusCode, endpoint, body)
	}
	return nil
}

// IsAuthContext returns true when err (or any error in its chain) is an
// AuthContextError or the ErrAuthContext sentinel.
func IsAuthContext(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrAuthContext)
}
