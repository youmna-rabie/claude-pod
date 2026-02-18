package types

import "net/http"

// Channel defines the interface for an incoming webhook channel.
type Channel interface {
	Name() string
	ValidateRequest(r *http.Request) error
	ParseRequest(r *http.Request) (*Event, error)
}
