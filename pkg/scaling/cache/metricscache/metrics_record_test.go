package metricscache

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreRecordMergesIntoExistingRecords(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "ScaledObject.test-namespace.test-name"

	first := MetricsRecord{IsMetricActive: true}
	second := MetricsRecord{IsMetricActive: false, ScalerError: errors.New("some error")}

	cache.StoreRecord(identifier, "metric-one", first)
	cache.StoreRecord(identifier, "metric-two", second)

	record, found := cache.ReadRecord(identifier, "metric-one")
	assert.True(t, found)
	assert.Equal(t, first, record)

	record, found = cache.ReadRecord(identifier, "metric-two")
	assert.True(t, found)
	assert.Equal(t, second, record)

	// overwriting one metric must not touch the other
	updated := MetricsRecord{IsMetricActive: false}
	cache.StoreRecord(identifier, "metric-one", updated)

	record, found = cache.ReadRecord(identifier, "metric-one")
	assert.True(t, found)
	assert.Equal(t, updated, record)

	_, found = cache.ReadRecord(identifier, "metric-two")
	assert.True(t, found)
}

func TestStoreRecordsReplacesAllRecords(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "ScaledObject.test-namespace.test-name"

	cache.StoreRecord(identifier, "metric-one", MetricsRecord{IsMetricActive: true})
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		"metric-two": {IsMetricActive: true},
	})

	_, found := cache.ReadRecord(identifier, "metric-one")
	assert.False(t, found)

	_, found = cache.ReadRecord(identifier, "metric-two")
	assert.True(t, found)
}

func TestDeleteRemovesAllRecords(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "ScaledObject.test-namespace.test-name"

	cache.StoreRecord(identifier, "metric-one", MetricsRecord{IsMetricActive: true})
	cache.Delete(identifier)

	_, found := cache.ReadRecord(identifier, "metric-one")
	assert.False(t, found)
}
