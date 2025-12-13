/*
Copyright 2021 The KEDA Authors

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

package metricscache

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

func TestStoreRecords_FirstTimeWrite(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metricName := "test-metric"

	metricValue := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(100, resource.DecimalSI),
	}

	records := map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{metricValue},
			ScalerError: nil,
		},
	}

	cache.StoreRecords(identifier, records)

	// Verify record was stored
	storedRecord, exists := cache.ReadRecord(identifier, metricName)
	assert.True(t, exists)
	assert.True(t, storedRecord.IsActive)
	assert.Nil(t, storedRecord.ScalerError)
	assert.Equal(t, 1, len(storedRecord.Metric))
	assert.Equal(t, int64(100), storedRecord.Metric[0].Value.Value())
}

func TestStoreRecords_UpdateWithNewMetrics(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metricName := "test-metric"

	// Store initial record
	initialMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(100, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{initialMetric},
			ScalerError: nil,
		},
	})

	// Update with new metrics
	newMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(200, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    false,
			Metric:      []external_metrics.ExternalMetricValue{newMetric},
			ScalerError: nil,
		},
	})

	// Verify new metrics were stored
	storedRecord, exists := cache.ReadRecord(identifier, metricName)
	assert.True(t, exists)
	assert.False(t, storedRecord.IsActive)
	assert.Nil(t, storedRecord.ScalerError)
	assert.Equal(t, 1, len(storedRecord.Metric))
	assert.Equal(t, int64(200), storedRecord.Metric[0].Value.Value())
}

func TestStoreRecords_PreserveMetricsOnError(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metricName := "test-metric"

	// Store initial good record
	goodMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(100, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{goodMetric},
			ScalerError: nil,
		},
	})

	// Update with error - should preserve old metrics
	scalerError := errors.New("scaler connection failed")
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    false,
			Metric:      []external_metrics.ExternalMetricValue{},
			ScalerError: scalerError,
		},
	})

	// Verify old metrics are preserved but error and activity are updated
	storedRecord, exists := cache.ReadRecord(identifier, metricName)
	assert.True(t, exists)
	assert.False(t, storedRecord.IsActive, "IsActive should be updated")
	assert.NotNil(t, storedRecord.ScalerError, "ScalerError should be updated")
	assert.Equal(t, "scaler connection failed", storedRecord.ScalerError.Error())
	assert.Equal(t, 1, len(storedRecord.Metric), "Old metrics should be preserved")
	assert.Equal(t, int64(100), storedRecord.Metric[0].Value.Value(), "Old metric value should be preserved")
}

func TestStoreRecords_PreserveMetricsOnEmptyMetrics(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metricName := "test-metric"

	// Store initial good record
	goodMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(150, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{goodMetric},
			ScalerError: nil,
		},
	})

	// Update with empty metrics (no error) - should preserve old metrics
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    false,
			Metric:      []external_metrics.ExternalMetricValue{},
			ScalerError: nil,
		},
	})

	// Verify old metrics are preserved but activity is updated
	storedRecord, exists := cache.ReadRecord(identifier, metricName)
	assert.True(t, exists)
	assert.False(t, storedRecord.IsActive, "IsActive should be updated")
	assert.Nil(t, storedRecord.ScalerError, "ScalerError should be nil")
	assert.Equal(t, 1, len(storedRecord.Metric), "Old metrics should be preserved")
	assert.Equal(t, int64(150), storedRecord.Metric[0].Value.Value(), "Old metric value should be preserved")
}

func TestStoreRecords_RecoverFromError(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metricName := "test-metric"

	// Store initial good record
	goodMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(100, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{goodMetric},
			ScalerError: nil,
		},
	})

	// Update with error
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    false,
			Metric:      []external_metrics.ExternalMetricValue{},
			ScalerError: errors.New("temporary error"),
		},
	})

	// Recover with new good metrics
	recoveredMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(200, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{recoveredMetric},
			ScalerError: nil,
		},
	})

	// Verify new metrics replaced old ones
	storedRecord, exists := cache.ReadRecord(identifier, metricName)
	assert.True(t, exists)
	assert.True(t, storedRecord.IsActive)
	assert.Nil(t, storedRecord.ScalerError)
	assert.Equal(t, 1, len(storedRecord.Metric))
	assert.Equal(t, int64(200), storedRecord.Metric[0].Value.Value())
}

func TestStoreRecords_EmptyInput(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metricName := "test-metric"

	// Store initial record
	initialMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(100, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{initialMetric},
			ScalerError: nil,
		},
	})

	// Try to store empty map - should not affect existing records
	cache.StoreRecords(identifier, map[string]MetricsRecord{})

	// Verify original record still exists
	storedRecord, exists := cache.ReadRecord(identifier, metricName)
	assert.True(t, exists)
	assert.True(t, storedRecord.IsActive)
	assert.Equal(t, 1, len(storedRecord.Metric))
	assert.Equal(t, int64(100), storedRecord.Metric[0].Value.Value())
}

func TestStoreRecords_MultipleMetrics(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metric1Name := "metric-1"
	metric2Name := "metric-2"

	// Store multiple metrics
	metric1 := external_metrics.ExternalMetricValue{
		MetricName: metric1Name,
		Value:      *resource.NewQuantity(100, resource.DecimalSI),
	}
	metric2 := external_metrics.ExternalMetricValue{
		MetricName: metric2Name,
		Value:      *resource.NewQuantity(200, resource.DecimalSI),
	}

	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metric1Name: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{metric1},
			ScalerError: nil,
		},
		metric2Name: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{metric2},
			ScalerError: nil,
		},
	})

	// Update only metric1 with error - metric2 should remain unchanged
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metric1Name: {
			IsActive:    false,
			Metric:      []external_metrics.ExternalMetricValue{},
			ScalerError: errors.New("metric1 error"),
		},
	})

	// Verify metric1 preserved old values
	record1, exists := cache.ReadRecord(identifier, metric1Name)
	assert.True(t, exists)
	assert.False(t, record1.IsActive)
	assert.NotNil(t, record1.ScalerError)
	assert.Equal(t, 1, len(record1.Metric))
	assert.Equal(t, int64(100), record1.Metric[0].Value.Value())

	// Verify metric2 remains unchanged
	record2, exists := cache.ReadRecord(identifier, metric2Name)
	assert.True(t, exists)
	assert.True(t, record2.IsActive)
	assert.Nil(t, record2.ScalerError)
	assert.Equal(t, 1, len(record2.Metric))
	assert.Equal(t, int64(200), record2.Metric[0].Value.Value())
}

func TestDelete(t *testing.T) {
	cache := NewMetricsCache()
	identifier := "test-namespace.test-scaledobject"
	metricName := "test-metric"

	// Store a record
	metricValue := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(100, resource.DecimalSI),
	}
	cache.StoreRecords(identifier, map[string]MetricsRecord{
		metricName: {
			IsActive:    true,
			Metric:      []external_metrics.ExternalMetricValue{metricValue},
			ScalerError: nil,
		},
	})

	// Verify it exists
	_, exists := cache.ReadRecord(identifier, metricName)
	assert.True(t, exists)

	// Delete it
	cache.Delete(identifier)

	// Verify it's gone
	_, exists = cache.ReadRecord(identifier, metricName)
	assert.False(t, exists)
}

func TestReadRecord_NonExistent(t *testing.T) {
	cache := NewMetricsCache()

	// Try to read non-existent record
	_, exists := cache.ReadRecord("non-existent-identifier", "non-existent-metric")
	assert.False(t, exists)
}
