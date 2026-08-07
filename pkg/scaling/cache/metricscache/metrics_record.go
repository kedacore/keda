package metricscache

import (
	"sync"

	"k8s.io/metrics/pkg/apis/external_metrics"
)

// MetricsRecord is the raw outcome of one metric query, after fallback processing: the values,
// the raw activity the scaler reported, and whether the values came from the fallback.
type MetricsRecord struct {
	// IsMetricActive is the activity the scaler itself reported, without fallback applied.
	IsMetricActive bool
	Metric         []external_metrics.ExternalMetricValue
	ScalerError    error
	// FallbackActive indicates that the metric values were produced by the fallback configuration
	// because the scaler itself was failing.
	FallbackActive bool
}

// MetricsCache carries metric observations from the path that queries the trigger sources to the
// path that doesn't. Per ScaledObject there is exactly one writer: the scale loop for triggers
// with useCachedMetrics (StoreRecords), or the HPA-driven metrics path when pollingInterval is
// not relevant (StoreRecord). The two modes are mutually exclusive, since useCachedMetrics makes
// pollingInterval relevant (see ScaledObject.IsPollingRelevant).
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

// StoreRecords replaces all records of the ScaledObject with the given ones.
func (mc *MetricsCache) StoreRecords(scaledObjectIdentifier string, metricsRecords map[string]MetricsRecord) {
	mc.lock.Lock()
	defer mc.lock.Unlock()
	mc.metricRecords[scaledObjectIdentifier] = metricsRecords
}

// StoreRecord stores a single metric record, merging it into the existing records of the
// ScaledObject instead of replacing them.
func (mc *MetricsCache) StoreRecord(scaledObjectIdentifier, metricName string, record MetricsRecord) {
	mc.lock.Lock()
	defer mc.lock.Unlock()
	records, ok := mc.metricRecords[scaledObjectIdentifier]
	if !ok {
		records = map[string]MetricsRecord{}
		mc.metricRecords[scaledObjectIdentifier] = records
	}
	records[metricName] = record
}

func (mc *MetricsCache) Delete(scaledObjectIdentifier string) {
	mc.lock.Lock()
	defer mc.lock.Unlock()
	delete(mc.metricRecords, scaledObjectIdentifier)
}
