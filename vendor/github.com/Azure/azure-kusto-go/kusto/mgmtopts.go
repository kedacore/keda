package kusto

import (
	"time"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
)

type mgmtOptions struct {
	requestProperties *requestProperties
	queryIngestion    bool
}

// Deprecated: Writing mode is now the default. Use the `RequestReadonly` option to make a read-only request.
func AllowWrite() MgmtOption {
	return func(m *mgmtOptions) error {
		return nil
	}
}

// IngestionEndpoint will instruct the Mgmt call to connect to the ingest-[endpoint] instead of [endpoint].
// This is not often used by end users and can only be used with a Mgmt() call.
func IngestionEndpoint() MgmtOption {
	return func(m *mgmtOptions) error {
		m.queryIngestion = true
		return nil
	}
}

// mgmtServerTimeout is the amount of time the server will allow a call to take.
// NOTE: I have made the serverTimeout private. For the moment, I'm going to use the context.Context timer
// to set timeouts via this private method.
func mgmtServerTimeout(d time.Duration) MgmtOption {
	return func(m *mgmtOptions) error {
		if d > 1*time.Hour {
			return errors.ES(errors.OpQuery, errors.KClientArgs, "ServerTimeout option was set to %v, but can't be more than 1 hour", d)
		}
		m.requestProperties.Options["servertimeout"] = value.Timespan{Valid: true, Value: d}.Marshal()
		return nil
	}
}
