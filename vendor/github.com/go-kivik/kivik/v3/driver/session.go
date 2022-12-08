package driver

import (
	"context"
	"encoding/json"
)

// Session is a copy of kivik.Session
type Session struct {
	// Name is the name of the authenticated user.
	Name string
	// Roles is a list of roles the user belongs to.
	Roles []string
	// AuthenticationMethod is the authentication method that was used for this
	// session.
	AuthenticationMethod string
	// AuthenticationDB is the user database against which authentication was
	// performed.
	AuthenticationDB string
	// AuthenticationHandlers is a list of authentication handlers configured on
	// the server.
	AuthenticationHandlers []string
	// RawResponse is the raw JSON response sent by the server, useful for
	// custom backends which may provide additional fields.
	RawResponse json.RawMessage
}

// Sessioner is an optional interface that a Client may satisfy to provide
// access to the authenticated session information.
type Sessioner interface {
	// Session returns information about the authenticated user.
	Session(ctx context.Context) (*Session, error)
}
