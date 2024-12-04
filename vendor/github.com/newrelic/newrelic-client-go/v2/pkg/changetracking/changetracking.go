package changetracking

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

type Changetracking struct {
	client http.Client
	config config.Config
	logger logging.Logger
	pager  http.Pager
}

func New(config config.Config) Changetracking {

	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := Changetracking{
		client: client,
		config: config,
		logger: config.GetLogger(),
		pager:  &http.LinkHeaderPager{},
	}

	return pkg
}
