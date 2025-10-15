package newrelic

import (
	"net/http"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/pkg/entityrelationship"
	"github.com/newrelic/newrelic-client-go/v2/pkg/users"

	"github.com/newrelic/newrelic-client-go/v2/pkg/accountmanagement"
	"github.com/newrelic/newrelic-client-go/v2/pkg/accounts"
	"github.com/newrelic/newrelic-client-go/v2/pkg/agent"
	"github.com/newrelic/newrelic-client-go/v2/pkg/agentapplications"
	"github.com/newrelic/newrelic-client-go/v2/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/v2/pkg/apiaccess"
	"github.com/newrelic/newrelic-client-go/v2/pkg/apm"
	"github.com/newrelic/newrelic-client-go/v2/pkg/authorizationmanagement"
	"github.com/newrelic/newrelic-client-go/v2/pkg/changetracking"
	"github.com/newrelic/newrelic-client-go/v2/pkg/cloud"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/customeradministration"
	"github.com/newrelic/newrelic-client-go/v2/pkg/dashboards"
	"github.com/newrelic/newrelic-client-go/v2/pkg/edge"
	"github.com/newrelic/newrelic-client-go/v2/pkg/entities"
	"github.com/newrelic/newrelic-client-go/v2/pkg/events"
	"github.com/newrelic/newrelic-client-go/v2/pkg/eventstometrics"
	"github.com/newrelic/newrelic-client-go/v2/pkg/installevents"
	"github.com/newrelic/newrelic-client-go/v2/pkg/keytransaction"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logconfigurations"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logs"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nerdgraph"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nerdstorage"
	"github.com/newrelic/newrelic-client-go/v2/pkg/notifications"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrqldroprules"
	"github.com/newrelic/newrelic-client-go/v2/pkg/organization"
	"github.com/newrelic/newrelic-client-go/v2/pkg/pipelinecontrol"
	"github.com/newrelic/newrelic-client-go/v2/pkg/plugins"
	"github.com/newrelic/newrelic-client-go/v2/pkg/servicelevel"
	"github.com/newrelic/newrelic-client-go/v2/pkg/synthetics"
	"github.com/newrelic/newrelic-client-go/v2/pkg/usermanagement"
	"github.com/newrelic/newrelic-client-go/v2/pkg/workflows"
	"github.com/newrelic/newrelic-client-go/v2/pkg/workloads"
)

// NewRelic is a collection of New Relic APIs.
type NewRelic struct {
	AccountManagement       accountmanagement.Accountmanagement
	Accounts                accounts.Accounts
	Agent                   agent.Agent
	AgentApplications       agentapplications.AgentApplications
	Alerts                  alerts.Alerts
	APIAccess               apiaccess.APIAccess
	APM                     apm.APM
	AuthorizationManagement authorizationmanagement.Authorizationmanagement
	ChangeTracking          changetracking.Changetracking
	Cloud                   cloud.Cloud
	CustomerAdministration  customeradministration.Customeradministration
	Dashboards              dashboards.Dashboards
	Edge                    edge.Edge
	Entities                entities.Entities
	Events                  events.Events
	EventsToMetrics         eventstometrics.EventsToMetrics
	InstallEvents           installevents.Installevents
	Logs                    logs.Logs
	Logconfigurations       logconfigurations.Logconfigurations
	NerdGraph               nerdgraph.NerdGraph
	NerdStorage             nerdstorage.NerdStorage
	Notifications           notifications.Notifications
	Nrdb                    nrdb.Nrdb
	Nrqldroprules           nrqldroprules.Nrqldroprules
	Organization            organization.Organization
	Pipelinecontrol         pipelinecontrol.Pipelinecontrol
	Plugins                 plugins.Plugins
	ServiceLevel            servicelevel.Servicelevel
	Synthetics              synthetics.Synthetics
	UserManagement          usermanagement.Usermanagement
	Workflows               workflows.Workflows
	Workloads               workloads.Workloads
	KeyTransaction          keytransaction.Keytransaction
	EntityRelationship      entityrelationship.Entityrelationship
	Users                   users.Users

	config config.Config
}

// New returns a collection of New Relic APIs.
func New(opts ...ConfigOption) (*NewRelic, error) {
	cfg := config.New()

	err := cfg.Init(opts)
	if err != nil {
		return nil, err
	}

	nr := &NewRelic{
		config: cfg,

		AccountManagement:       accountmanagement.New(cfg),
		Accounts:                accounts.New(cfg),
		Agent:                   agent.New(cfg),
		AgentApplications:       agentapplications.New(cfg),
		Alerts:                  alerts.New(cfg),
		APIAccess:               apiaccess.New(cfg),
		APM:                     apm.New(cfg),
		AuthorizationManagement: authorizationmanagement.New(cfg),
		ChangeTracking:          changetracking.New(cfg),
		Cloud:                   cloud.New(cfg),
		CustomerAdministration:  customeradministration.New(cfg),
		Dashboards:              dashboards.New(cfg),
		Edge:                    edge.New(cfg),
		Entities:                entities.New(cfg),
		Events:                  events.New(cfg),
		EventsToMetrics:         eventstometrics.New(cfg),
		InstallEvents:           installevents.New(cfg),
		Logs:                    logs.New(cfg),
		Logconfigurations:       logconfigurations.New(cfg),
		NerdGraph:               nerdgraph.New(cfg),
		NerdStorage:             nerdstorage.New(cfg),
		Notifications:           notifications.New(cfg),
		Nrdb:                    nrdb.New(cfg),
		Nrqldroprules:           nrqldroprules.New(cfg),
		Organization:            organization.New(cfg),
		Pipelinecontrol:         pipelinecontrol.New(cfg),
		Plugins:                 plugins.New(cfg),
		ServiceLevel:            servicelevel.New(cfg),
		Synthetics:              synthetics.New(cfg),
		UserManagement:          usermanagement.New(cfg),
		Workflows:               workflows.New(cfg),
		Workloads:               workloads.New(cfg),
		KeyTransaction:          keytransaction.New(cfg),
		EntityRelationship:      entityrelationship.New(cfg),
		Users:                   users.New(cfg),
	}

	return nr, nil
}

func (nr *NewRelic) SetLogLevel(levelName string) {
	nr.config.Logger.SetLevel(levelName)
}

// TestEndpoints makes a few calls to determine if the NewRelic enpoints are reachable.
func (nr *NewRelic) TestEndpoints() error {
	endpoints := []string{
		//	nr.config.Region().InfrastructureURL(),
		nr.config.Region().LogsURL(),
		nr.config.Region().NerdGraphURL(),
		nr.config.Region().RestURL(),
	}

	for _, e := range endpoints {
		_, err := http.Get(e)
		if err != nil {
			return err
		}
	}

	return nil
}

// ConfigOption configures the Config when provided to NewApplication.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
type ConfigOption = config.ConfigOption

// ConfigPersonalAPIKey sets the New Relic Admin API key this client will use.
// This key should be used to create a client instance.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
func ConfigPersonalAPIKey(apiKey string) ConfigOption {
	return config.ConfigPersonalAPIKey(apiKey)
}

// ConfigInsightsInsertKey sets the New Relic Insights insert key this client will use.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
func ConfigInsightsInsertKey(insightsInsertKey string) ConfigOption {
	return config.ConfigInsightsInsertKey(insightsInsertKey)
}

// ConfigAdminAPIKey sets the New Relic Admin API key this client will use.
// Deprecated.  Use a personal API key for authentication.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
func ConfigAdminAPIKey(adminAPIKey string) ConfigOption {
	return config.ConfigAdminAPIKey(adminAPIKey)
}

// ConfigRegion sets the New Relic Region this client will use.
func ConfigRegion(r string) ConfigOption {
	return config.ConfigRegion(r)
}

// ConfigHTTPTimeout sets the timeout for HTTP requests.
func ConfigHTTPTimeout(t time.Duration) ConfigOption {
	return config.ConfigHTTPTimeout(t)
}

// ConfigHTTPTransport sets the HTTP Transporter.
func ConfigHTTPTransport(transport http.RoundTripper) ConfigOption {
	return config.ConfigHTTPTransport(transport)
}

// ConfigUserAgent sets the HTTP UserAgent for API requests.
func ConfigUserAgent(ua string) ConfigOption {
	return config.ConfigUserAgent(ua)
}

// ConfigServiceName sets the service name logged
func ConfigServiceName(name string) ConfigOption {
	return config.ConfigServiceName(name)
}

// ConfigBaseURL sets the base URL used to make requests to the REST API V2.
func ConfigBaseURL(url string) ConfigOption {
	return config.ConfigBaseURL(url)
}

// ConfigInfrastructureBaseURL sets the base URL used to make requests to the Infrastructure API.
func ConfigInfrastructureBaseURL(url string) ConfigOption {
	return config.ConfigInfrastructureBaseURL(url)
}

// ConfigSyntheticsBaseURL sets the base URL used to make requests to the Synthetics API.
func ConfigSyntheticsBaseURL(url string) ConfigOption {
	return config.ConfigSyntheticsBaseURL(url)
}

// ConfigNerdGraphBaseURL sets the base URL used to make requests to the NerdGraph API.
func ConfigNerdGraphBaseURL(url string) ConfigOption {
	return config.ConfigNerdGraphBaseURL(url)
}

// ConfigLogLevel sets the log level for the client.
func ConfigLogLevel(logLevel string) ConfigOption {
	return config.ConfigLogLevel(logLevel)
}

// ConfigLogJSON toggles JSON formatting on for the logger if set to true.
func ConfigLogJSON(logJSON bool) ConfigOption {
	return config.ConfigLogJSON(logJSON)
}

// ConfigLogger can be used to customize the client's logger.
// Custom loggers must conform to the logging.Logger interface.
func ConfigLogger(logger logging.Logger) ConfigOption {
	return config.ConfigLogger(logger)
}
