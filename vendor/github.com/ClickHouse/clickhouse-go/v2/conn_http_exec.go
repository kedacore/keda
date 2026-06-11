package clickhouse

import (
	"context"
)

func (h *httpConnect) exec(ctx context.Context, query string, args ...any) error {
	options := queryOptions(ctx)
	query, err := bindQueryOrAppendParameters(true, &options, query, h.handshake.Timezone, args...)
	if err != nil {
		return err
	}

	res, err := h.sendQuery(ctx, query, &options, nil) //nolint:bodyclose // false positive
	if err != nil {
		return err
	}
	defer discardAndClose(res.Body)

	return nil
}
