package logs

import (
	"errors"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Logs is used to send log data to the New Relic Log API

const (
	DefaultBatchWorkers = 1
	DefaultBatchSize    = 900
	DefaultBatchTimeout = 60 * time.Second
)

type Logs struct {
	client http.Client
	config config.Config
	logger logging.Logger

	// For queue based log handling
	accountID  int
	logQueue   chan interface{}
	logTimer   *time.Timer
	flushQueue []chan bool

	// These have defaults
	batchWorkers int
	batchSize    int
	batchTimeout time.Duration
}

// New is used to create a new Logs client instance.
func New(cfg config.Config) Logs {
	cfg.Compression = config.Compression.Gzip

	client := http.NewClient(cfg)
	if cfg.InsightsInsertKey != "" {
		client.SetAuthStrategy(&http.LogsInsertKeyAuthorizer{})
	} else {
		client.SetAuthStrategy(&http.LicenseKeyAuthorizer{})
	}

	pkg := Logs{
		client:       client,
		config:       cfg,
		logger:       cfg.GetLogger(),
		batchWorkers: DefaultBatchWorkers,
		batchSize:    DefaultBatchSize,
		batchTimeout: DefaultBatchTimeout,
	}

	return pkg
}

// CreateLogEntry reports a log entry to New Relic.
// It's up to the caller to send a valid Log API payload, no checking done here
func (l *Logs) CreateLogEntry(logEntry interface{}) error {
	if logEntry == nil {
		return errors.New("logs: CreateLogEntry: logEntry is nil, nothing to do")
	}
	_, err := l.client.Post(l.config.Region().LogsURL(), nil, logEntry, nil)

	// If no error is returned then the call succeeded
	if err != nil {
		return err
	}

	return nil
}
