/*
Copyright 2018 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"fmt"
	"sync"

	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
)

var (
	curMetricsExporter view.Exporter
	curMetricsConfig   *metricsConfig
	metricsMux         sync.Mutex
)

// newMetricsExporter gets a metrics exporter based on the config.
func newMetricsExporter(config *metricsConfig, logger *zap.SugaredLogger) error {
	// If there is a Prometheus Exporter server running, stop it.
	resetCurPromSrv()
	ce := getCurMetricsExporter()
	if ce != nil {
		// UnregisterExporter is idempotent and it can be called multiple times for the same exporter
		// without side effects.
		view.UnregisterExporter(ce)
	}
	var err error
	var e view.Exporter
	switch config.backendDestination {
	case Stackdriver:
		e, err = newStackdriverExporter(config, logger)
	case Prometheus:
		e, err = newPrometheusExporter(config, logger)
	default:
		err = fmt.Errorf("Unsupported metrics backend %v", config.backendDestination)
	}
	if err != nil {
		return err
	}
	existingConfig := getCurMetricsConfig()
	setCurMetricsExporterAndConfig(e, config)
	logger.Infof("Successfully updated the metrics exporter; old config: %v; new config %v", existingConfig, config)
	return nil
}

func getCurMetricsExporter() view.Exporter {
	metricsMux.Lock()
	defer metricsMux.Unlock()
	return curMetricsExporter
}

func setCurMetricsExporterAndConfig(e view.Exporter, c *metricsConfig) {
	metricsMux.Lock()
	defer metricsMux.Unlock()
	view.RegisterExporter(e)
	if c != nil {
		view.SetReportingPeriod(c.reportingPeriod)
	} else {
		// Setting to 0 enables the default behavior.
		view.SetReportingPeriod(0)
	}
	curMetricsExporter = e
	curMetricsConfig = c
}

func getCurMetricsConfig() *metricsConfig {
	metricsMux.Lock()
	defer metricsMux.Unlock()
	return curMetricsConfig
}
