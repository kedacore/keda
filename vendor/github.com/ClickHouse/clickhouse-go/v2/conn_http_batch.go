package clickhouse

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func fetchColumnNamesAndTypesForInsert(h *httpConnect, release nativeTransportRelease, ctx context.Context, tableName string, requestedColumnNames []string) ([]ColumnNameAndType, error) {
	describeTableQuery := fmt.Sprintf("DESCRIBE TABLE %s", tableName)
	r, err := h.query(ctx, release, describeTableQuery)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	columnsToTypes := make(map[string]string)
	var allColumns []string
	for r.Next() {
		var (
			colName     string
			colType     string
			defaultType string
			ignore      string
		)

		if err = r.Scan(&colName, &colType, &defaultType, &ignore, &ignore, &ignore, &ignore); err != nil {
			return nil, err
		}
		// these column types cannot be specified in INSERT queries
		if defaultType == "MATERIALIZED" || defaultType == "ALIAS" {
			continue
		}

		columnsToTypes[colName] = colType
		allColumns = append(allColumns, colName)
	}

	// The order of the columns must match the INSERT list, or the DESC table if no insert list was provided
	insertColumns := make([]ColumnNameAndType, 0, len(allColumns))

	if len(requestedColumnNames) > 0 {
		// Validate requested columns present
		for _, colName := range requestedColumnNames {
			colType, ok := columnsToTypes[colName]
			if !ok {
				return nil, fmt.Errorf("column %s is not present in the table %s", colName, tableName)
			}

			insertColumns = append(insertColumns, ColumnNameAndType{
				Name: colName,
				Type: colType,
			})
		}
	} else {
		// Use all columns
		for _, colName := range allColumns {
			colType := columnsToTypes[colName]
			insertColumns = append(insertColumns, ColumnNameAndType{
				Name: colName,
				Type: colType,
			})
		}
	}

	return insertColumns, nil
}

func newBlock(h *httpConnect, release nativeTransportRelease, ctx context.Context, query string) (string, *proto.Block, error) {
	normalizedQuery, tableName, requestedColumnNames, err := extractNormalizedInsertQueryAndColumns(query)
	if err != nil {
		return "", nil, err
	}

	opt := queryOptions(ctx)
	columns := opt.columnNamesAndTypes

	// If the user didn't supply known column names/types, do expensive DESC TABLE logic
	if opt.columnNamesAndTypes == nil {
		fetchedColumns, err := fetchColumnNamesAndTypesForInsert(h, release, ctx, tableName, requestedColumnNames)
		if err != nil {
			return "", nil, fmt.Errorf("failed to determine columns for HTTP insert: %w", err)
		}
		columns = fetchedColumns
	}

	var block proto.Block
	serverContext := serverVersionToContext(h.handshake)
	block.ServerContext = &serverContext
	for _, col := range columns {
		if err := block.AddColumn(col.Name, column.Type(col.Type)); err != nil {
			return "", nil, err
		}
	}

	return normalizedQuery, &block, nil
}

func (h *httpConnect) prepareBatch(ctx context.Context, release nativeTransportRelease, acquire nativeTransportAcquire, query string, opts driver.PrepareBatchOptions) (driver.Batch, error) {
	// release is not used within newBlock since the connection is held for the batch.
	query, block, err := newBlock(h, func(nativeTransport, error) {}, ctx, query)
	if err != nil {
		err = fmt.Errorf("failed to init block for HTTP batch: %w", err)
		release(h, err)
		return nil, err
	}

	return &httpBatch{
		ctx:         ctx,
		conn:        h,
		connRelease: release,
		structMap:   &structMap{},
		block:       block,
		query:       query,
	}, nil
}

type httpBatch struct {
	query       string
	err         error
	ctx         context.Context
	conn        *httpConnect
	released    bool
	connRelease nativeTransportRelease
	structMap   *structMap
	sent        bool
	block       *proto.Block
}

func (b *httpBatch) release(err error) {
	if !b.released {
		b.released = true
		b.connRelease(b.conn, err)
	}
}

func (b *httpBatch) Flush() error {
	// Flush and Send are effectively the same for HTTP, but users should just use Send until we
	// figure out a way to do proper streaming.
	return nil
}

func (b *httpBatch) Close() error {
	if b.sent || b.released {
		return nil
	}

	b.sent = true
	b.release(nil)

	return nil
}

func (b *httpBatch) Abort() error {
	defer func() {
		b.sent = true
		b.release(os.ErrProcessDone)
	}()
	if b.sent {
		return ErrBatchAlreadySent
	}
	return nil
}

func (b *httpBatch) Append(v ...any) error {
	if b.sent {
		return ErrBatchAlreadySent
	}
	if b.err != nil {
		return b.err
	}

	if err := b.block.Append(v...); err != nil {
		b.err = fmt.Errorf("%w: %w", ErrBatchInvalid, err)
		b.release(err)
		return err
	}

	return nil
}

func (b *httpBatch) AppendStruct(v any) error {
	if b.err != nil {
		return b.err
	}
	values, err := b.structMap.Map("AppendStruct", b.block.ColumnsNames(), v, false)
	if err != nil {
		return err
	}
	return b.Append(values...)
}

func (b *httpBatch) Column(idx int) driver.BatchColumn {
	if len(b.block.Columns) <= idx {
		return &batchColumn{
			err: &OpError{
				Op:  "batch.Column",
				Err: fmt.Errorf("invalid column index %d", idx),
			},
		}
	}
	return &batchColumn{
		batch:  b,
		column: b.block.Columns[idx],
		release: func(err error) {
			b.err = err
		},
	}
}

func (b *httpBatch) IsSent() bool {
	return b.sent
}

func (b *httpBatch) Send() (err error) {
	defer func() {
		b.sent = true
		b.release(err)
	}()
	if b.sent {
		return ErrBatchAlreadySent
	}
	if b.err != nil {
		return b.err
	}
	if b.block.Rows() == 0 {
		return nil
	}

	options := queryOptions(b.ctx)
	headers := make(map[string]string)
	switch b.conn.compression {
	case CompressionGZIP, CompressionDeflate, CompressionBrotli:
		headers["Content-Encoding"] = b.conn.compression.String()
	case CompressionZSTD, CompressionLZ4:
		options.settings["decompress"] = "1"
		options.settings["compress"] = "1"
	}

	compressionWriter := b.conn.compressionPool.Get()
	defer b.conn.compressionPool.Put(compressionWriter)
	pipeReader, pipeWriter := io.Pipe()
	connWriter := compressionWriter.reset(pipeWriter)

	go func() {
		var err error
		defer pipeWriter.CloseWithError(err)
		defer connWriter.Close()
		b.conn.buffer.Reset()
		if err = b.conn.writeData(b.block); err != nil {
			return
		}
		if _, err = connWriter.Write(b.conn.buffer.Buf); err != nil {
			return
		}
	}()

	options.settings["query"] = b.query
	headers["Content-Type"] = "application/octet-stream"

	b.conn.logger.Debug("batch: sending via HTTP",
		slog.Int("columns", len(b.block.Columns)),
		slog.Int("rows", b.block.Rows()))
	res, err := b.conn.sendStreamQuery(b.ctx, pipeReader, &options, headers) //nolint:bodyclose // false positive
	if err != nil {
		return fmt.Errorf("batch sendStreamQuery: %w", err)
	}
	discardAndClose(res.Body)

	b.conn.logger.Debug("batch: send complete")
	b.block.Reset()

	return nil
}

func (b *httpBatch) Rows() int {
	return b.block.Rows()
}

func (b *httpBatch) Columns() []column.Interface {
	return slices.Clone(b.block.Columns)
}

var _ driver.Batch = (*httpBatch)(nil)
