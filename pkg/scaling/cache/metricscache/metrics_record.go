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
	mc.metricRecords[scaledObjectIdentifier] = metricsRecords
}

func (mc *MetricsCache) Delete(scaledObjectIdentifier string) {
	mc.lock.Lock()
	defer mc.lock.Unlock()
	delete(mc.metricRecords, scaledObjectIdentifier)
}
