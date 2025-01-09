package agentapplications

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

type AgentApplications struct {
	client http.Client
	logger logging.Logger
}

func New(config config.Config) AgentApplications {
	return AgentApplications{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
