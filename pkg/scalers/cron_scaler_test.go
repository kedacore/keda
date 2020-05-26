package scalers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type parseCronMetadataTestData struct {
	metadata map[string]string
	isError  bool
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
}

var tz, _ = time.LoadLocation(validCronMetadata["timezone"])
var currentDay = time.Now().In(tz).Weekday().String()

func TestCronParseMetadata(t *testing.T) {
	for _, testData := range testCronMetadata {
		_, err := parseCronMetadata(testData.metadata, map[string]string{})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestIsActive(t *testing.T) {
	scaler, _ := NewCronScaler(map[string]string{}, validCronMetadata)
	isActive, _ := scaler.IsActive(context.TODO())
	if currentDay == "Thursday" {
		assert.Equal(t, isActive, true)
	} else {
		assert.Equal(t, isActive, false)
	}
}

func TestGetMetrics(t *testing.T) {
	scaler, _ := NewCronScaler(map[string]string{}, validCronMetadata)
	metrics, _ := scaler.GetMetrics(context.TODO(), "ReplicaCount", nil)
	assert.Equal(t, metrics[0].MetricName, "ReplicaCount")
	if currentDay == "Thursday" {
		assert.Equal(t, metrics[0].Value.Value(), int64(10))
	} else {
		assert.Equal(t, metrics[0].Value.Value(), int64(1))
	}
}
