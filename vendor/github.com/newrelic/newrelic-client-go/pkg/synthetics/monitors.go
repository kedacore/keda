package synthetics

import (
	"context"
	"path"
)

const (
	listMonitorsLimit = 100
)

// Monitor represents a New Relic Synthetics monitor.
type Monitor struct {
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name"`
	Type         MonitorType       `json:"type"`
	Frequency    uint              `json:"frequency"`
	URI          string            `json:"uri"`
	Locations    []string          `json:"locations"`
	Status       MonitorStatusType `json:"status"`
	SLAThreshold float64           `json:"slaThreshold"`
	UserID       uint              `json:"userId,omitempty"`
	APIVersion   string            `json:"apiVersion,omitempty"`
	ModifiedAt   *Time             `json:"modifiedAt,omitempty"`
	CreatedAt    *Time             `json:"createdAt,omitempty"`
	Options      MonitorOptions    `json:"options,omitempty"`
}

// MonitorScriptLocation represents a New Relic Synthetics monitor script location.
type MonitorScriptLocation struct {
	Name string `json:"name"`
	HMAC string `json:"hmac"`
}

// MonitorScript represents a New Relic Synthetics monitor script.
type MonitorScript struct {
	Text      string                  `json:"scriptText"`
	Locations []MonitorScriptLocation `json:"scriptLocations"`
}

// MonitorType represents a Synthetics monitor type.
type MonitorType string

// MonitorStatusType represents a Synthetics monitor status type.
type MonitorStatusType string

// MonitorOptions represents the options for a New Relic Synthetics monitor.
type MonitorOptions struct {
	ValidationString       string `json:"validationString,omitempty"`
	VerifySSL              bool   `json:"verifySSL,omitempty"`
	BypassHEADRequest      bool   `json:"bypassHEADRequest,omitempty"`
	TreatRedirectAsFailure bool   `json:"treatRedirectAsFailure,omitempty"`
}

var (
	// MonitorTypes specifies the possible types for a Synthetics monitor.
	MonitorTypes = struct {
		Ping            MonitorType
		Browser         MonitorType
		ScriptedBrowser MonitorType
		APITest         MonitorType
	}{
		Ping:            "SIMPLE",
		Browser:         "BROWSER",
		ScriptedBrowser: "SCRIPT_BROWSER",
		APITest:         "SCRIPT_API",
	}

	// MonitorStatus specifies the possible Synthetics monitor status types.
	MonitorStatus = struct {
		Enabled  MonitorStatusType
		Muted    MonitorStatusType
		Disabled MonitorStatusType
	}{
		Enabled:  "ENABLED",
		Muted:    "MUTED",
		Disabled: "DISABLED",
	}
)

// ListMonitors is used to retrieve New Relic Synthetics monitors.
func (s *Synthetics) ListMonitors() ([]*Monitor, error) {
	return s.ListMonitorsWithContext(context.Background())
}

// ListMonitorsWithContext is used to retrieve New Relic Synthetics monitors.
func (s *Synthetics) ListMonitorsWithContext(ctx context.Context) ([]*Monitor, error) {
	results := []*Monitor{}
	nextURL := s.config.Region().SyntheticsURL("/v4/monitors")
	queryParams := listMonitorsParams{
		Limit: listMonitorsLimit,
	}

	for nextURL != "" {
		response := listMonitorsResponse{}

		resp, err := s.client.GetWithContext(ctx, nextURL, &queryParams, &response)

		if err != nil {
			return nil, err
		}

		results = append(results, response.Monitors...)

		paging := s.pager.Parse(resp)
		nextURL = paging.Next
	}

	return results, nil
}

// GetMonitor is used to retrieve a specific New Relic Synthetics monitor.
func (s *Synthetics) GetMonitor(monitorID string) (*Monitor, error) {
	return s.GetMonitorWithContext(context.Background(), monitorID)
}

// GetMonitorWithContext is used to retrieve a specific New Relic Synthetics monitor.
func (s *Synthetics) GetMonitorWithContext(ctx context.Context, monitorID string) (*Monitor, error) {
	resp := Monitor{}

	_, err := s.client.GetWithContext(ctx, s.config.Region().SyntheticsURL("/v4/monitors", monitorID), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateMonitor is used to create a New Relic Synthetics monitor.
//Deprecated: Use one of the following methods instead based on your needs -
//syntheticsCreateBrokenLinksMonitor(Broken links monitor),
//syntheticsCreateCertCheckMonitor(Cert Check Monitor),
// syntheticsCreateScriptBrowserMonitor(Script Browser Monitor),
//syntheticsCreateSimpleBrowserMonitor(Simple Browser Monitor),
//syntheticsCreateSimpleMonitor(Simple Monitor),
//syntheticsCreateStepMonitor(Step Monitor).
func (s *Synthetics) CreateMonitor(monitor Monitor) (*Monitor, error) {
	return s.CreateMonitorWithContext(context.Background(), monitor)
}

// CreateMonitorWithContext is used to create a New Relic Synthetics monitor.
//Deprecated: Use one of the following methods instead based on your needs -
//syntheticsCreateBrokenLinksMonitorWithContext(Broken links monitor),
//syntheticsCreateCertCheckMonitorWithContext(Cert Check Monitor),
// syntheticsCreateScriptBrowserMonitorWithContext(Script Browser Monitor),
//syntheticsCreateSimpleBrowserMonitorWithContext(Simple Browser Monitor),
//syntheticsCreateSimpleMonitorWithContext(Simple Monitor),
//syntheticsCreateStepMonitorWithContext(Step Monitor).
func (s *Synthetics) CreateMonitorWithContext(ctx context.Context, monitor Monitor) (*Monitor, error) {
	resp, err := s.client.PostWithContext(ctx, s.config.Region().SyntheticsURL("/v4/monitors"), nil, &monitor, nil)

	if err != nil {
		return nil, err
	}

	l := resp.Header.Get("location")
	monitorID := path.Base(l)

	monitor.ID = monitorID

	return &monitor, nil
}

// UpdateMonitor is used to update a New Relic Synthetics monitor.
//Deprecated: Use one of the following methods instead based on your needs -
//syntheticsUpdateBrokenLinksMonitor(Broken links monitor),
//syntheticsUpdateCertCheckMonitor(Cert Check Monitor),
// syntheticsUpdateScriptBrowserMonitor(Script Browser Monitor),
//syntheticsUpdateSimpleBrowserMonitor(Simple Browser Monitor),
//syntheticsUpdateSimpleMonitor(Simple Monitor),
//syntheticsUpdateStepMonitor(Step Monitor).
func (s *Synthetics) UpdateMonitor(monitor Monitor) (*Monitor, error) {
	return s.UpdateMonitorWithContext(context.Background(), monitor)
}

// UpdateMonitorWithContext is used to update a New Relic Synthetics monitor.
//Deprecated: Use one of the following methods instead based on your needs -
//syntheticsUpdateBrokenLinksMonitorWithContext(Broken links monitor),
//syntheticsUpdateCertCheckMonitorWithContext(Cert Check Monitor),
// syntheticsUpdateScriptBrowserMonitorWithContext(Script Browser Monitor),
//syntheticsUpdateSimpleBrowserMonitorWithContext(Simple Browser Monitor),
//syntheticsUpdateSimpleMonitorWithContext(Simple Monitor),
//syntheticsUpdateStepMonitorWithContext(Step Monitor).
func (s *Synthetics) UpdateMonitorWithContext(ctx context.Context, monitor Monitor) (*Monitor, error) {
	_, err := s.client.PutWithContext(ctx, s.config.Region().SyntheticsURL("/v4/monitors", monitor.ID), nil, &monitor, nil)

	if err != nil {
		return nil, err
	}

	return &monitor, nil
}

// DeleteMonitor is used to delete a New Relic Synthetics monitor.
// Deprecated: Use the following method to delete all New Relic Synthetics Monitors.
//SyntheticsDeleteMonitor
func (s *Synthetics) DeleteMonitor(monitorID string) error {
	return s.DeleteMonitorWithContext(context.Background(), monitorID)
}

// DeleteMonitorWithContext is used to delete a New Relic Synthetics monitor.
// Deprecated: Use the following method to delete all New Relic Synthetics Monitors.
//SyntheticsDeleteMonitorWithContext
func (s *Synthetics) DeleteMonitorWithContext(ctx context.Context, monitorID string) error {
	_, err := s.client.DeleteWithContext(ctx, s.config.Region().SyntheticsURL("/v4/monitors", monitorID), nil, nil)

	if err != nil {
		return err
	}

	return nil
}

type listMonitorsResponse struct {
	Monitors []*Monitor `json:"monitors,omitempty"`
}

type listMonitorsParams struct {
	Limit int `url:"limit,omitempty"`
}
