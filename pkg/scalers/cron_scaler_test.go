package scalers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type parseCronMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type cronMetricIdentifier struct {
	metadataTestData *parseCronMetadataTestData
	name             string
}

// A complete valid metadata example for reference
var validCronMetadata = map[string]string{
	"timezone":        "Etc/UTC",
	"start":           "0 0 * * Thu",
	"end":             "59 23 * * Thu",
	"desiredReplicas": "10",
}

var testCronMetadata = []parseCronMetadataTestData{
	{map[string]string{}, true},
	{validCronMetadata, false},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "45 * * * *"}, true},
	{map[string]string{"start": "30 * * * *", "end": "45 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30-33 * * * *", "end": "45 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "45-50 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "-30 * * * *", "end": "45 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "-50 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "50 * * -3 *", "desiredReplicas": "10"}, true},
}

var cronMetricIdentifiers = []cronMetricIdentifier{
	{&testCronMetadata[1], "cron-Etc-UTC-00xxThu-5923xxThu"},
}

var tz, _ = time.LoadLocation(validCronMetadata["timezone"])
var currentDay = time.Now().In(tz).Weekday().String()

func TestCronParseMetadata(t *testing.T) {
	for _, testData := range testCronMetadata {
		_, err := parseCronMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestIsActive(t *testing.T) {
	scaler, _ := NewCronScaler(&ScalerConfig{TriggerMetadata: validCronMetadata})
	isActive, _ := scaler.IsActive(context.TODO())
	if currentDay == "Thursday" {
		assert.Equal(t, isActive, true)
	} else {
		assert.Equal(t, isActive, false)
	}
}

func TestGetMetrics(t *testing.T) {
	scaler, _ := NewCronScaler(&ScalerConfig{TriggerMetadata: validCronMetadata})
	metrics, _ := scaler.GetMetrics(context.TODO(), "ReplicaCount", nil)
	assert.Equal(t, metrics[0].MetricName, "ReplicaCount")
	if currentDay == "Thursday" {
		assert.Equal(t, metrics[0].Value.Value(), int64(10))
	} else {
		assert.Equal(t, metrics[0].Value.Value(), int64(1))
	}
}

func TestCronGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range cronMetricIdentifiers {
		meta, err := parseCronMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockCronScaler := cronScaler{meta}

		metricSpec := mockCronScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
