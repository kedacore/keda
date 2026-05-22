package scalers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseCronMinReplicasMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type cronMinReplicasMetricIdentifier struct {
	metadataTestData *parseCronMinReplicasMetadataTestData
	triggerIndex     int
	name             string
}

var validCronMinReplicasMetadata = map[string]string{
	"timezone":    "Etc/UTC",
	"start":       "0 22 * * *",
	"end":         "0 6 * * *",
	"minReplicas": "2",
}

var validCronMinReplicasMetadataWithMax = map[string]string{
	"timezone":    "Etc/UTC",
	"start":       "0 22 * * *",
	"end":         "0 6 * * *",
	"minReplicas": "2",
	"maxReplicas": "5",
}

var validCronMinReplicasMetadataDay = map[string]string{
	"timezone":    "Etc/UTC",
	"start":       "0 6 * * *",
	"end":         "0 22 * * *",
	"minReplicas": "10",
	"maxReplicas": "50",
}

var testCronMinReplicasMetadata = []parseCronMinReplicasMetadataTestData{
	{map[string]string{}, true},
	{validCronMinReplicasMetadata, false},
	{validCronMinReplicasMetadataWithMax, false},
	{validCronMinReplicasMetadataDay, false},
	// missing timezone
	{map[string]string{"start": "0 22 * * *", "end": "0 6 * * *", "minReplicas": "2"}, true},
	// same start and end
	{map[string]string{"timezone": "Etc/UTC", "start": "0 22 * * *", "end": "0 22 * * *", "minReplicas": "2"}, true},
	// minReplicas = 0
	{map[string]string{"timezone": "Etc/UTC", "start": "0 22 * * *", "end": "0 6 * * *", "minReplicas": "0"}, true},
	// missing minReplicas
	{map[string]string{"timezone": "Etc/UTC", "start": "0 22 * * *", "end": "0 6 * * *"}, true},
	// maxReplicas < minReplicas
	{map[string]string{"timezone": "Etc/UTC", "start": "0 22 * * *", "end": "0 6 * * *", "minReplicas": "10", "maxReplicas": "5"}, true},
	// maxReplicas negative
	{map[string]string{"timezone": "Etc/UTC", "start": "0 22 * * *", "end": "0 6 * * *", "minReplicas": "2", "maxReplicas": "-1"}, true},
	// invalid start expression
	{map[string]string{"timezone": "Etc/UTC", "start": "-30 * * * *", "end": "0 6 * * *", "minReplicas": "2"}, true},
	// invalid end expression
	{map[string]string{"timezone": "Etc/UTC", "start": "0 22 * * *", "end": "-30 * * * *", "minReplicas": "2"}, true},
}

var cronMinReplicasMetricIdentifiers = []cronMinReplicasMetricIdentifier{
	{&testCronMinReplicasMetadata[1], 0, "s0-cron-min-replicas-Etc-UTC-022xxx-06xxx"},
	{&testCronMinReplicasMetadata[3], 1, "s1-cron-min-replicas-Etc-UTC-06xxx-022xxx"},
}

var tzMinReplicas, _ = time.LoadLocation("Etc/UTC")
var currentHourForMinReplicas = time.Now().In(tzMinReplicas).Hour()

func TestCronMinReplicasParseMetadata(t *testing.T) {
	for i, testData := range testCronMinReplicasMetadata {
		_, err := parseCronMinReplicasMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Errorf("test case %d: expected success but got error: %v", i, err)
		}
		if testData.isError && err == nil {
			t.Errorf("test case %d: expected error but got success", i)
		}
	}
}

func TestCronMinReplicasIsActiveNight(t *testing.T) {
	scaler, _ := NewCronMinReplicasScaler(nil, &scalersconfig.ScalerConfig{TriggerMetadata: validCronMinReplicasMetadata})
	_, isActive, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	// night window: 22:00-06:00
	if currentHourForMinReplicas >= 22 || currentHourForMinReplicas < 6 {
		assert.Equal(t, true, isActive)
	} else {
		assert.Equal(t, false, isActive)
	}
}

func TestCronMinReplicasIsActiveDay(t *testing.T) {
	scaler, _ := NewCronMinReplicasScaler(nil, &scalersconfig.ScalerConfig{TriggerMetadata: validCronMinReplicasMetadataDay})
	_, isActive, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	// day window: 06:00-22:00
	if currentHourForMinReplicas >= 6 && currentHourForMinReplicas < 22 {
		assert.Equal(t, true, isActive)
	} else {
		assert.Equal(t, false, isActive)
	}
}

func TestCronMinReplicasGetMetricsNight(t *testing.T) {
	scaler, _ := NewCronMinReplicasScaler(nil, &scalersconfig.ScalerConfig{TriggerMetadata: validCronMinReplicasMetadata})
	metrics, _, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	assert.Equal(t, "ReplicaCount", metrics[0].MetricName)
	if currentHourForMinReplicas >= 22 || currentHourForMinReplicas < 6 {
		assert.Equal(t, int64(2), metrics[0].Value.Value())
	} else {
		assert.Equal(t, int64(0), metrics[0].Value.Value())
	}
}

func TestCronMinReplicasGetMetricsDay(t *testing.T) {
	scaler, _ := NewCronMinReplicasScaler(nil, &scalersconfig.ScalerConfig{TriggerMetadata: validCronMinReplicasMetadataDay})
	metrics, _, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	assert.Equal(t, "ReplicaCount", metrics[0].MetricName)
	if currentHourForMinReplicas >= 6 && currentHourForMinReplicas < 22 {
		assert.Equal(t, int64(10), metrics[0].Value.Value())
	} else {
		assert.Equal(t, int64(0), metrics[0].Value.Value())
	}
}

func TestCronMinReplicasMaxReplicasMetadata(t *testing.T) {
	meta, err := parseCronMinReplicasMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: validCronMinReplicasMetadataWithMax})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), meta.MinReplicas)
	assert.Equal(t, int64(5), meta.MaxReplicas)
}

func TestCronMinReplicasMaxReplicasAbsent(t *testing.T) {
	meta, err := parseCronMinReplicasMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: validCronMinReplicasMetadata})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), meta.MaxReplicas)
}

func TestCronMinReplicasGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range cronMinReplicasMetricIdentifiers {
		meta, err := parseCronMinReplicasMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		startSchedule, _ := parser.Parse(meta.Start)
		endSchedule, _ := parser.Parse(meta.End)

		mockScaler := cronMinReplicasScaler{
			metricType:    "",
			metadata:      meta,
			logger:        logr.Discard(),
			startSchedule: startSchedule,
			endSchedule:   endSchedule,
		}

		metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
