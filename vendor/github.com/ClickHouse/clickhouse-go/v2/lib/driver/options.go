package driver

type PrepareBatchOptions struct {
	ReleaseConnection bool
	CloseOnFlush      bool
}

type PrepareBatchOption func(options *PrepareBatchOptions)

func WithReleaseConnection() PrepareBatchOption {
	return func(options *PrepareBatchOptions) {
		options.ReleaseConnection = true
	}
}

// WithCloseOnFlush closes batch INSERT query when Flush is executed
func WithCloseOnFlush() PrepareBatchOption {
	return func(options *PrepareBatchOptions) {
		options.CloseOnFlush = true
	}
}
