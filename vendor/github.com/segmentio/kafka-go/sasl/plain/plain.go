package plain

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go/sasl"
)

// Mechanism implements the PLAIN mechanism and passes the credentials in clear
// text.
type Mechanism struct {
	Username string
	Password string
}

func (Mechanism) Name() string {
	return "PLAIN"
}

func (m Mechanism) Start(ctx context.Context) (sasl.StateMachine, []byte, error) {
	// Mechanism is stateless, so it can also implement sasl.Session
	return m, []byte(fmt.Sprintf("\x00%s\x00%s", m.Username, m.Password)), nil
}

func (m Mechanism) Next(ctx context.Context, challenge []byte) (bool, []byte, error) {
	// kafka will return error if it rejected the credentials, so we'd only
	// arrive here on success.
	return true, nil, nil
}
