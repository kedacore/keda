package driver

type PrepareBatchOptions struct {
	ReleaseConnection bool
	CloseOnFlush      bool
}

type PrepareBatchOption func(options *PrepareBatchOptions)

// WithReleaseConnection releases the underlying connection back to the pool immediately after PrepareBatch.
//
// This is useful for long-lived batches that should not hold a connection open between Flush/Send calls.
// The driver will reacquire a connection when it needs to transmit data.
func WithReleaseConnection() PrepareBatchOption {
	return func(options *PrepareBatchOptions) {
		options.ReleaseConnection = true
	}
}

// WithCloseOnFlush closes the current INSERT and releases the connection whenever Flush is executed.
//
// This can be used to send data incrementally without keeping a server-side INSERT open.
func WithCloseOnFlush() PrepareBatchOption {
	return func(options *PrepareBatchOptions) {
		options.CloseOnFlush = true
	}
}
