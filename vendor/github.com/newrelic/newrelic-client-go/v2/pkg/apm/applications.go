package apm

import "context"

// ApplicationsInterface interface should be refactored to be a global interface for fetching NR type things
type ApplicationsInterface interface {
	find(accountID int, name string) (*Application, error)
	list(accountID int, params *ListApplicationsParams) ([]*Application, error)

	create(accountID int, name string) (*Application, error)
	get(accountID int, applicationID int) (*Application, error)
	update(accountID int, applicationID int, params UpdateApplicationParams) (*Application, error)
	remove(accountID int, applicationID int) (*Application, error) // delete is a reserved word...
}

// Application represents information about a New Relic application.
type Application struct {
	ID             int                       `json:"id,omitempty"`
	Name           string                    `json:"name,omitempty"`
	Language       string                    `json:"language,omitempty"`
	HealthStatus   string                    `json:"health_status,omitempty"`
	Reporting      bool                      `json:"reporting"`
	LastReportedAt string                    `json:"last_reported_at,omitempty"`
	Summary        ApplicationSummary        `json:"application_summary,omitempty"`
	EndUserSummary ApplicationEndUserSummary `json:"end_user_summary,omitempty"`
	Settings       ApplicationSettings       `json:"settings,omitempty"`
	Links          ApplicationLinks          `json:"links,omitempty"`
}

// ApplicationSummary represents performance information about a New Relic application.
type ApplicationSummary struct {
	ResponseTime            float64 `json:"response_time"`
	Throughput              float64 `json:"throughput"`
	ErrorRate               float64 `json:"error_rate"`
	ApdexTarget             float64 `json:"apdex_target"`
	ApdexScore              float64 `json:"apdex_score"`
	HostCount               int     `json:"host_count"`
	InstanceCount           int     `json:"instance_count"`
	ConcurrentInstanceCount int     `json:"concurrent_instance_count"`
}

// ApplicationEndUserSummary represents performance information about a New Relic application.
type ApplicationEndUserSummary struct {
	ResponseTime float64 `json:"response_time"`
	Throughput   float64 `json:"throughput"`
	ApdexTarget  float64 `json:"apdex_target"`
	ApdexScore   float64 `json:"apdex_score"`
}

// ApplicationSettings represents some of the settings of a New Relic application.
type ApplicationSettings struct {
	AppApdexThreshold        float64 `json:"app_apdex_threshold,omitempty"`
	EndUserApdexThreshold    float64 `json:"end_user_apdex_threshold,omitempty"`
	EnableRealUserMonitoring bool    `json:"enable_real_user_monitoring"`
	UseServerSideConfig      bool    `json:"use_server_side_config"`
}

// ApplicationLinks represents all the links for a New Relic application.
type ApplicationLinks struct {
	ServerIDs     []int `json:"servers,omitempty"`
	HostIDs       []int `json:"application_hosts,omitempty"`
	InstanceIDs   []int `json:"application_instances,omitempty"`
	AlertPolicyID int   `json:"alert_policy"`
}

// ListApplicationsParams represents a set of filters to be
// used when querying New Relic applications.
type ListApplicationsParams struct {
	Name     string `url:"filter[name],omitempty"`
	Host     string `url:"filter[host],omitempty"`
	IDs      []int  `url:"filter[ids],omitempty,comma"`
	Language string `url:"filter[language],omitempty"`
}

// UpdateApplicationParams represents a set of parameters to be
// used when updating New Relic applications.
type UpdateApplicationParams struct {
	Name     string
	Settings ApplicationSettings
}

// ListApplications is used to retrieve New Relic applications.
func (a *APM) ListApplications(params *ListApplicationsParams) ([]*Application, error) {
	return a.ListApplicationsWithContext(context.Background(), params)
}

// ListApplicationsWithContext is used to retrieve New Relic applications.
func (a *APM) ListApplicationsWithContext(ctx context.Context, params *ListApplicationsParams) ([]*Application, error) {
	accountID := 0

	method := applicationsREST{
		parent: a,
	}

	return method.list(ctx, accountID, params)
}

// GetApplication is used to retrieve a single New Relic application.
func (a *APM) GetApplication(applicationID int) (*Application, error) {
	return a.GetApplicationWithContext(context.Background(), applicationID)
}

// GetApplicationWithContext is used to retrieve a single New Relic application.
func (a *APM) GetApplicationWithContext(ctx context.Context, applicationID int) (*Application, error) {
	accountID := 0
	method := applicationsREST{
		parent: a,
	}

	return method.get(ctx, accountID, applicationID)
}

// UpdateApplication is used to update a New Relic application's name and/or settings.
func (a *APM) UpdateApplication(applicationID int, params UpdateApplicationParams) (*Application, error) {
	return a.UpdateApplicationWithContext(context.Background(), applicationID, params)
}

// UpdateApplicationWithContext is used to update a New Relic application's name and/or settings.
func (a *APM) UpdateApplicationWithContext(ctx context.Context, applicationID int, params UpdateApplicationParams) (*Application, error) {
	accountID := 0
	method := applicationsREST{
		parent: a,
	}

	return method.update(ctx, accountID, applicationID, params)
}

// DeleteApplication is used to delete a New Relic application.
// This process will only succeed if the application is no longer reporting data.
func (a *APM) DeleteApplication(applicationID int) (*Application, error) {
	return a.DeleteApplicationWithContext(context.Background(), applicationID)
}

// DeleteApplicationWithContext is used to delete a New Relic application.
// This process will only succeed if the application is no longer reporting data.
func (a *APM) DeleteApplicationWithContext(ctx context.Context, applicationID int) (*Application, error) {
	accountID := 0
	method := applicationsREST{
		parent: a,
	}

	return method.remove(ctx, accountID, applicationID)
}
