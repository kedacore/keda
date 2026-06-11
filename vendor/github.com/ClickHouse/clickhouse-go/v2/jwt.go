package clickhouse

import (
	"context"
)

// jwtAuthMarker is the marker for JSON Web Token authentication in ClickHouse Cloud.
// At the protocol level this is used in place of a username.
const jwtAuthMarker = " JWT AUTHENTICATION "

type GetJWTFunc = func(ctx context.Context) (string, error)

// useJWTAuth returns true if the client should use JWT auth
func useJWTAuth(opt *Options) bool {
	return opt.GetJWT != nil
}
