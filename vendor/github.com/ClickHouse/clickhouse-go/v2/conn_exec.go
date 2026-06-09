package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func (c *connect) exec(ctx context.Context, query string, args ...any) error {
	var (
		options                    = queryOptions(ctx)
		queryParamsProtocolSupport = c.revision >= proto.DBMS_MIN_PROTOCOL_VERSION_WITH_PARAMETERS
		body, err                  = bindQueryOrAppendParameters(queryParamsProtocolSupport, &options, query, c.server.Timezone, args...)
	)
	if err != nil {
		return err
	}
	// set a read deadline - alternative to context.Read operation will fail if no data is received after deadline.
	c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	defer c.conn.SetReadDeadline(time.Time{})
	// context level deadlines override any read deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetDeadline(deadline)
		defer c.conn.SetDeadline(time.Time{})
	}
	if err := c.sendQuery(body, &options); err != nil {
		return err
	}
	return c.process(ctx, options.onProcess())
}
