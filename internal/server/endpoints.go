package server

import (
	"net/http"
	"strings"
)

// EndpointSet represents a list of endpoint names to disable.
type EndpointSet []string

// IsAllowed checks if a given HTTP request's endpoint is allowed.
func (e EndpointSet) IsAllowed(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	endpoint := parts[len(parts)-1]
	return !e.Contains(endpoint)
}

// Contains checks if a given endpoint name is present in the disabled set.
func (e EndpointSet) Contains(endpoint string) bool {
	for _, name := range e {
		if endpoint == name {
			return true
		}
	}
	return false
}
