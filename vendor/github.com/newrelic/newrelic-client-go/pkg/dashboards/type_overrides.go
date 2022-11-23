// DashboardBillboardWidgetThresholdInput - Billboard widget threshold input.
package dashboards

import "github.com/newrelic/newrelic-client-go/pkg/entities"

type DashboardBillboardWidgetThresholdInput struct {
	// alert severity.
	AlertSeverity entities.DashboardAlertSeverity `json:"alertSeverity,omitempty"`
	// value.
	Value *float64 `json:"value,omitempty"`
}
