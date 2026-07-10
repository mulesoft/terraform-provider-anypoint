package client

import "sync"

// ResponseCache provides thread-safe, per-apply caching for expensive API calls
// whose results are stable within a single terraform apply (e.g. the available-roles
// catalog, the org user list, and team lists). All resources sharing the same
// *Config / *UserClientConfig instance share the same cache — so 5 team resources
// in one .tf file make ONE ListAvailableRoles call, not 15.
//
// The cache lives on UserClientConfig (passed to every AM sub-client). It is NOT
// persisted between applies — each `terraform apply` starts fresh.
//
// Keys use the format "<method>:<args>" (e.g. "available_roles", "org_users:orgId",
// "teams:orgId").
type ResponseCache struct {
	mu    sync.RWMutex
	store map[string]interface{}
}

// NewResponseCache creates a new empty cache.
func NewResponseCache() *ResponseCache {
	return &ResponseCache{
		store: make(map[string]interface{}),
	}
}

// Get retrieves a cached value. Returns (value, true) if found, (nil, false) otherwise.
func (c *ResponseCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.store[key]
	return v, ok
}

// Set stores a value in the cache.
func (c *ResponseCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
}

// GetOrFetch atomically checks the cache and, if missing, calls fetch() to populate it.
// This is the recommended way to use the cache — avoids the thundering-herd problem
// where N goroutines all miss the cache and fire N identical API calls.
func (c *ResponseCache) GetOrFetch(key string, fetch func() (interface{}, error)) (interface{}, error) {
	// Fast path: read lock
	c.mu.RLock()
	if v, ok := c.store[key]; ok {
		c.mu.RUnlock()
		return v, nil
	}
	c.mu.RUnlock()

	// Slow path: write lock + double-check
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.store[key]; ok {
		return v, nil
	}

	v, err := fetch()
	if err != nil {
		return nil, err
	}
	c.store[key] = v
	return v, nil
}
