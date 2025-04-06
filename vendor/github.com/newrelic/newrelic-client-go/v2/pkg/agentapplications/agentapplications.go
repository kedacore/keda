package agentapplications

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/entities"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
	mock "github.com/newrelic/newrelic-client-go/v2/pkg/testhelpers"
	"testing"
)

type AgentApplications struct {
	client http.Client
	logger logging.Logger
	config config.Config
}

func New(config config.Config) AgentApplications {
	return AgentApplications{
		client: http.NewClient(config),
		logger: config.GetLogger(),
		config: config,
	}
}

func newMockResponseApm(t *testing.T, mockJSONResponse string, statusCode int) AgentApplications {
	ts := mock.NewMockServer(t, mockJSONResponse, statusCode)
	tc := mock.NewTestConfig(t, ts)

	return New(tc)
}

// nolint
func newMockResponse(t *testing.T, mockJSONResponse string, statusCode int) entities.Entities {
	ts := mock.NewMockServer(t, mockJSONResponse, statusCode)
	tc := mock.NewTestConfig(t, ts)

	return entities.New(tc)
}
