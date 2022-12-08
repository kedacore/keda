package couchdb

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kivik/kivik/v3/driver"
)

type session struct {
	Data    json.RawMessage
	Info    authInfo    `json:"info"`
	UserCtx userContext `json:"userCtx"`
}

type authInfo struct {
	AuthenticationMethod   string   `json:"authenticated"`
	AuthenticationDB       string   `json:"authentiation_db"`
	AuthenticationHandlers []string `json:"authentication_handlers"`
}

type userContext struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

func (s *session) UnmarshalJSON(data []byte) error {
	type alias session
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = session(a)
	s.Data = data
	return nil
}

func (c *client) Session(ctx context.Context) (*driver.Session, error) {
	s := &session{}
	_, err := c.DoJSON(ctx, http.MethodGet, "/_session", nil, s)
	return &driver.Session{
		RawResponse:            s.Data,
		Name:                   s.UserCtx.Name,
		Roles:                  s.UserCtx.Roles,
		AuthenticationMethod:   s.Info.AuthenticationMethod,
		AuthenticationDB:       s.Info.AuthenticationDB,
		AuthenticationHandlers: s.Info.AuthenticationHandlers,
	}, err
}
