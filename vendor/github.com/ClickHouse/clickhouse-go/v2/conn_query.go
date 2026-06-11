package clickhouse

import (
	"context"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func (c *connect) query(ctx context.Context, release nativeTransportRelease, query string, args ...any) (*rows, error) {
	var (
		options                    = queryOptions(ctx)
		onProcess                  = options.onProcess()
		queryParamsProtocolSupport = c.revision >= proto.DBMS_MIN_PROTOCOL_VERSION_WITH_PARAMETERS
		body, err                  = bindQueryOrAppendParameters(queryParamsProtocolSupport, &options, query, c.server.Timezone, args...)
	)

	if err != nil {
		c.logger.Error("failed to bind query parameters", slog.Any("error", err))
		release(c, err)
		return nil, err
	}

	if err = c.sendQuery(body, &options); err != nil {
		release(c, err)
		return nil, err
	}

	init, err := c.firstBlock(ctx, onProcess)

	if err != nil {
		c.logger.Error("failed to get first block", slog.Any("error", err))
		release(c, err)
		return nil, err
	}
	bufferSize := c.blockBufferSize
	if options.blockBufferSize > 0 {
		// allow block buffer sze to be overridden per query
		bufferSize = options.blockBufferSize
	}
	var (
		errors = make(chan error, 1)
		stream = make(chan *proto.Block, bufferSize)
	)

	go func() {
		onProcess.data = func(b *proto.Block) {
			stream <- b
		}
		err := c.process(ctx, onProcess)
		if err != nil {
			c.logger.Error("query processing failed", slog.Any("error", err))
			errors <- err
		}
		close(stream)
		close(errors)
		release(c, err)
	}()

	return &rows{
		block:     init,
		stream:    stream,
		errors:    errors,
		columns:   init.ColumnsNames(),
		structMap: c.structMap,
	}, nil
}

func (c *connect) queryRow(ctx context.Context, release nativeTransportRelease, query string, args ...any) *row {
	rows, err := c.query(ctx, release, query, args...)
	if err != nil {
		return &row{
			err: err,
		}
	}
	return &row{
		rows: rows,
	}
}
