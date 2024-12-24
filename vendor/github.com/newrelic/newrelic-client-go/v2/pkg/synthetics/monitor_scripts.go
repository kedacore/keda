package synthetics

import (
	"context"
	"encoding/base64"
	"fmt"
)

// Deprecated: Use entities.GetEntitySearch instead.
// GetMonitorScript is used to retrieve the script that belongs
// to a New Relic Synthetics scripted monitor.
func (s *Synthetics) GetMonitorScript(monitorID string) (*MonitorScript, error) {
	return s.GetMonitorScriptWithContext(context.Background(), monitorID)
}

// Deprecated: Use entities.GetEntitySearchWithContext instead.
// GetMonitorScriptWithContext is used to retrieve the script that belongs
// to a New Relic Synthetics scripted monitor.
func (s *Synthetics) GetMonitorScriptWithContext(ctx context.Context, monitorID string) (*MonitorScript, error) {
	resp := MonitorScript{}
	url := fmt.Sprintf("/v4/monitors/%s/script", monitorID)
	_, err := s.client.GetWithContext(ctx, s.config.Region().SyntheticsURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.Text)

	if err != nil {
		return nil, err
	}

	resp.Text = string(decoded)

	return &resp, nil
}

// Deprecated: Use one of following instead:
//
//	synthetics.SyntheticsUpdateScriptAPIMonitor
//	synthetics.SyntheticsUpdateScriptBrowserMonitor
//
// UpdateMonitorScript is used to add a script to an existing New Relic Synthetics monitor_script.
func (s *Synthetics) UpdateMonitorScript(monitorID string, script MonitorScript) (*MonitorScript, error) {
	return s.UpdateMonitorScriptWithContext(context.Background(), monitorID, script)
}

// Deprecated: Use one of following instead:
//
//	synthetics.SyntheticsUpdateScriptAPIMonitorWithContext
//	synthetics.SyntheticsUpdateScriptBrowserMonitorWithContext
//
// UpdateMonitorScriptWithContext is used to add a script to an existing New Relic Synthetics monitor_script.
func (s *Synthetics) UpdateMonitorScriptWithContext(ctx context.Context, monitorID string, script MonitorScript) (*MonitorScript, error) {
	script.Text = base64.StdEncoding.EncodeToString([]byte(script.Text))

	_, err := s.client.PutWithContext(ctx, s.config.Region().SyntheticsURL("/v4/monitors", monitorID, "/script"), nil, &script, nil)

	if err != nil {
		return nil, err
	}

	return &script, nil
}
