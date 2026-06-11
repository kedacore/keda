package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

// Connection::ping
// https://github.com/ClickHouse/ClickHouse/blob/master/src/Client/Connection.cpp
func (c *connect) ping(ctx context.Context) (err error) {
	// set a read deadline - alternative to context.Read operation will fail if no data is received after deadline.
	c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	defer c.conn.SetReadDeadline(time.Time{})
	// context level deadlines override any read deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetDeadline(deadline)
		defer c.conn.SetDeadline(time.Time{})
	}
	c.logger.Debug("ping: sending")
	c.buffer.PutByte(proto.ClientPing)
	if err := c.flush(); err != nil {
		return fmt.Errorf("ping: failed to send ping to %s (conn_id=%d): %w",
			c.conn.RemoteAddr(), c.id, err)
	}

	var packet byte
	for {
		if packet, err = c.reader.ReadByte(); err != nil {
			return fmt.Errorf("ping: failed to read packet from %s (conn_id=%d, age=%s): %w",
				c.conn.RemoteAddr(), c.id, time.Since(c.connectedAt).Round(time.Second), err)
		}
		switch packet {
		case proto.ServerException:
			return c.exception()
		case proto.ServerProgress:
			if _, err = c.progress(); err != nil {
				return err
			}
		case proto.ServerPong:
			c.logger.Debug("ping: received pong")
			return nil
		default:
			return fmt.Errorf("unexpected packet %d", packet)
		}
	}
}
