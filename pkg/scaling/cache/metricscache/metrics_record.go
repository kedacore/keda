package metricscache

import (
	"sync"

	"k8s.io/metrics/pkg/apis/external_metrics"
)

type MetricsRecord struct {
	IsActive    bool
	Metric      []external_metrics.ExternalMetricValue
	ScalerError error
}

type MetricsCache struct {
	metricRecords map[string]map[string]MetricsRecord
	lock          *sync.RWMutex
}

func NewMetricsCache() MetricsCache {
	return MetricsCache{
		metricRecords: map[string]map[string]MetricsRecord{},
		lock:          &sync.RWMutex{},
	}
}

func (mc *MetricsCache) ReadRecord(scaledObjectIdentifier, metricName string) (MetricsRecord, bool) {
	mc.lock.RLock()
	defer mc.lock.RUnlock()
	record, ok := mc.metricRecords[scaledObjectIdentifier][metricName]

	return record, ok
}

func (mc *MetricsCache) StoreRecords(scaledObjectIdentifier string, metricsRecords map[string]MetricsRecord) {
	mc.lock.Lock()
	defer mc.lock.Unlock()
	if len(metricsRecords) == 0 {
		// Nothing to store; keep existing records intact
		return
	}

	// Ensure inner map exists
	if _, exists := mc.metricRecords[scaledObjectIdentifier]; !exists {
		mc.metricRecords[scaledObjectIdentifier] = map[string]MetricsRecord{}
	}

	existing := mc.metricRecords[scaledObjectIdentifier]

	for metricName, newRecord := range metricsRecords {
		if oldRecord, existsOld := existing[metricName]; existsOld {
			// Preserve last good metrics when incoming record has an error or no metrics.
			if newRecord.ScalerError != nil || len(newRecord.Metric) == 0 {
				merged := oldRecord
				// Always update error and activity flags to reflect latest probe,
				// but keep the previous metric values.
				merged.ScalerError = newRecord.ScalerError
				merged.IsActive = newRecord.IsActive
				existing[metricName] = merged
				continue
			}
		}
		// Happy path: update with fresh metrics (or first-time write)
		existing[metricName] = newRecord
	}
}

func (mc *MetricsCache) Delete(scaledObjectIdentifier string) {
	mc.lock.Lock()
	defer mc.lock.Unlock()
	delete(mc.metricRecords, scaledObjectIdentifier)
}
