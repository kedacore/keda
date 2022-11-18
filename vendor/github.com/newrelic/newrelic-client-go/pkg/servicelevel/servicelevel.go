// Experimental.  For NR internal use only.
package servicelevel

import (
	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

type Servicelevel struct {
	client http.Client
	logger logging.Logger
}

func New(config config.Config) Servicelevel {
	return Servicelevel{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}

type EntityInterface struct {
	// The New Relic account ID associated with this entity.
	AccountID int `json:"accountId,omitempty"`
	// The entity's domain
	Domain string `json:"domain,omitempty"`
	// The name of this entity.
	Name string `json:"name,omitempty"`
	// The url to the entity.
	Permalink string `json:"permalink,omitempty"`
	// The service level defined for the entity.
	ServiceLevel ServiceLevelDefinition `json:"serviceLevel,omitempty"`
	// The entity's type
	Type string `json:"type,omitempty"`
}
