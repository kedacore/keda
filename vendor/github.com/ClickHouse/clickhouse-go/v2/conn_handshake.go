package clickhouse

import (
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func (c *connect) handshake(auth Auth) error {
	defer c.buffer.Reset()
	c.logger.Debug("handshake: sending client hello",
		slog.Int("protocol_version", ClientTCPProtocolVersion),
		slog.String("client_name", c.opt.ClientInfo.String()))
	// set a read deadline - alternative to context.Read operation will fail if no data is received after deadline.
	c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	defer c.conn.SetReadDeadline(time.Time{})
	// context level deadlines override any read deadline
	c.conn.SetDeadline(time.Now().Add(c.opt.DialTimeout))
	defer c.conn.SetDeadline(time.Time{})
	{
		c.buffer.PutByte(proto.ClientHello)
		handshake := &proto.ClientHandshake{
			ProtocolVersion: ClientTCPProtocolVersion,
			ClientName:      c.opt.ClientInfo.String(),
			ClientVersion:   proto.Version{ClientVersionMajor, ClientVersionMinor, ClientVersionPatch}, //nolint:govet
		}
		handshake.Encode(c.buffer)
		{
			c.buffer.PutString(auth.Database)
			c.buffer.PutString(auth.Username)
			c.buffer.PutString(auth.Password)
		}
		if err := c.flush(); err != nil {
			return fmt.Errorf("handshake: failed to send hello to %s (conn_id=%d): %w",
				c.conn.RemoteAddr(), c.id, err)
		}
	}
	{
		packet, err := c.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("handshake: failed to read packet from %s (conn_id=%d, auth_db=%s): %w",
				c.conn.RemoteAddr(), c.id, auth.Database, err)
		}
		switch packet {
		case proto.ServerException:
			return c.exception()
		case proto.ServerHello:
			if err := c.server.Decode(c.reader); err != nil {
				return fmt.Errorf("handshake: failed to decode server hello from %s (conn_id=%d): %w",
					c.conn.RemoteAddr(), c.id, err)
			}
		case proto.ServerEndOfStream:
			c.logger.Debug("handshake: received end of stream")
			return nil
		default:
			return fmt.Errorf("[handshake] unexpected packet [%d] from server", packet)
		}
	}
	if c.server.Revision < proto.DBMS_MIN_REVISION_WITH_CLIENT_INFO {
		return ErrUnsupportedServerRevision
	}

	if c.revision > c.server.Revision {
		c.revision = c.server.Revision
		c.logger.Debug("handshake: downgrading client protocol",
			slog.Uint64("from_revision", c.revision),
			slog.Uint64("to_revision", c.server.Revision))
	}
	c.logger.Debug("handshake complete",
		slog.String("server_name", c.server.Name),
		slog.String("server_version", c.server.Version.String()),
		slog.Uint64("server_revision", c.server.Revision),
		slog.String("server_timezone", c.server.Timezone.String()))
	return nil
}

func (c *connect) sendAddendum() error {
	if c.revision >= proto.DBMS_MIN_PROTOCOL_VERSION_WITH_QUOTA_KEY {
		c.buffer.PutString("") // todo quota key support
	}

	return c.flush()
}
