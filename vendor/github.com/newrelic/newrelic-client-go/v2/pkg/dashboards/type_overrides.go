// DashboardBillboardWidgetThresholdInput - Billboard widget threshold input.
package dashboards

import "github.com/newrelic/newrelic-client-go/v2/pkg/entities"

type DashboardBillboardWidgetThresholdInput struct {
	// alert severity.
	AlertSeverity entities.DashboardAlertSeverity `json:"alertSeverity,omitempty"`
	// value.
	Value *float64 `json:"value,omitempty"`
}

type DashboardLineWidgetThresholdInput struct {
	IsLabelVisible *bool                                        `json:"isLabelVisible,omitempty"`
	Thresholds     []DashboardLineWidgetThresholdThresholdInput `json:"thresholds,omitempty"`
}

type DashboardLineWidgetThresholdThresholdInput struct {
	From     string                                 `json:"from,omitempty"`
	To       string                                 `json:"to,omitempty"`
	Name     string                                 `json:"name,omitempty"`
	Severity DashboardLineTableWidgetsAlertSeverity `json:"severity,omitempty"`
}

type DashboardTableWidgetThresholdInput struct {
	From       string                                 `json:"from,omitempty"`
	To         string                                 `json:"to,omitempty"`
	ColumnName string                                 `json:"columnName,omitempty"`
	Severity   DashboardLineTableWidgetsAlertSeverity `json:"severity,omitempty"`
}

type DashboardLineTableWidgetsAlertSeverity string

var DashboardLineTableWidgetsAlertSeverityTypes = struct {
	SUCCESS     DashboardLineTableWidgetsAlertSeverity
	WARNING     DashboardLineTableWidgetsAlertSeverity
	UNAVAILABLE DashboardLineTableWidgetsAlertSeverity
	SEVERE      DashboardLineTableWidgetsAlertSeverity
	CRITICAL    DashboardLineTableWidgetsAlertSeverity
}{
	SUCCESS:     "success",
	WARNING:     "warning",
	UNAVAILABLE: "unavailable",
	SEVERE:      "severe",
	CRITICAL:    "critical",
}
