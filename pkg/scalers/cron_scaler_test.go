package scalers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseCronMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type cronMetricIdentifier struct {
	metadataTestData *parseCronMetadataTestData
	triggerIndex     int
	name             string
}

// A complete valid metadata example for reference
var validCronMetadata = map[string]string{
	"timezone":        "Etc/UTC",
	"start":           "0 0 * * Thu",
	"end":             "59 23 * * Thu",
	"desiredReplicas": "10",
}

// A complete valid metadata example which is enabled every even hours
var validCronMetadata2 = map[string]string{
	"timezone":        "Etc/UTC",
	"start":           "0 */2 * * *",    // Every 2 hours (even)
	"end":             "0 1-23/2 * * *", // Every 2 hours starting at 1 (odd)
	"desiredReplicas": "10",
}

// A valid metadata example using comma-separated values
var validCronMetadata3 = map[string]string{
	"timezone":        "Etc/UTC",
	"start":           "0,30 * * * *",  // Every hour at minute 0 and 30
	"end":             "15,45 * * * *", // Every hour at minute 15 and 45
	"desiredReplicas": "10",
}

var testCronMetadata = []parseCronMetadataTestData{
	{map[string]string{}, true},
	{validCronMetadata, false},
	{validCronMetadata2, false},
	{validCronMetadata3, false},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "45 * * * *"}, true},
	{map[string]string{"start": "30 * * * *", "end": "45 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30-33 * * * *", "end": "45 * * * *", "desiredReplicas": "10"}, false},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "45-50 * * * *", "desiredReplicas": "10"}, false},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "-30 * * * *", "end": "45 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "-50 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "50 * * -3 *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Asia/Kolkata", "start": "30 * * * *", "end": "30 * * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Etc/UTC", "start": "H 8 * * MON-FRI", "end": "H 17 * * MON-FRI", "desiredReplicas": "10"}, false},
	{map[string]string{"timezone": "Etc/UTC", "start": "H(0-29) H(6-7) * * *", "end": "H/15 18 * * *", "desiredReplicas": "10"}, false},
	{map[string]string{"timezone": "Etc/UTC", "start": "H(30-10) 8 * * *", "end": "0 17 * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Etc/UTC", "start": "H/0 8 * * *", "end": "0 17 * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Etc/UTC", "start": "H(0-99) 8 * * *", "end": "0 17 * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Etc/UTC", "start": "H/90 8 * * *", "end": "0 17 * * *", "desiredReplicas": "10"}, true},
	{map[string]string{"timezone": "Etc/UTC", "start": "CRON_TZ=Etc/UTC H 8 * * *", "end": "CRON_TZ=Etc/UTC H 17 * * *", "desiredReplicas": "10"}, false},
	{map[string]string{"timezone": "Etc/UTC", "start": "TZ=Etc/UTC H 8 * * *", "end": "TZ=Etc/UTC H 17 * * *", "desiredReplicas": "10"}, false},
}

var cronMetricIdentifiers = []cronMetricIdentifier{
	{&testCronMetadata[1], 0, "s0-cron-Etc-UTC-00xxThu-5923xxThu"},
	{&testCronMetadata[2], 1, "s1-cron-Etc-UTC-0xSl2xxx-01-23Sl2xxx"},
	{&testCronMetadata[3], 2, "s2-cron-Etc-UTC-0Cm30xxxx-15Cm45xxxx"},
	{&testCronMetadata[13], 0, "s0-cron-Etc-UTC-H-0-29-H-6-7-xxx-HSl1518xxx"},
}

var tz, _ = time.LoadLocation(validCronMetadata2["timezone"])
var currentDay = time.Now().In(tz).Weekday().String()
var currentHour = time.Now().In(tz).Hour()

func TestCronParseMetadata(t *testing.T) {
	for _, testData := range testCronMetadata {
		_, err := parseCronMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestIsActive(t *testing.T) {
	scaler, _ := NewCronScaler(&scalersconfig.ScalerConfig{TriggerMetadata: validCronMetadata})
	_, isActive, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	if currentDay == "Thursday" {
		assert.Equal(t, isActive, true)
	} else {
		assert.Equal(t, isActive, false)
	}
}

func TestIsActiveRange(t *testing.T) {
	scaler, _ := NewCronScaler(&scalersconfig.ScalerConfig{TriggerMetadata: validCronMetadata2})
	_, isActive, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	if currentHour%2 == 0 {
		assert.Equal(t, isActive, true)
	} else {
		assert.Equal(t, isActive, false)
	}
}

func TestGetMetrics(t *testing.T) {
	scaler, _ := NewCronScaler(&scalersconfig.ScalerConfig{TriggerMetadata: validCronMetadata})
	metrics, _, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	assert.Equal(t, metrics[0].MetricName, "ReplicaCount")
	if currentDay == "Thursday" {
		assert.Equal(t, metrics[0].Value.Value(), int64(10))
	} else {
		assert.Equal(t, metrics[0].Value.Value(), int64(0))
	}
}

func TestGetMetricsRange(t *testing.T) {
	scaler, _ := NewCronScaler(&scalersconfig.ScalerConfig{TriggerMetadata: validCronMetadata2})
	metrics, _, _ := scaler.GetMetricsAndActivity(context.TODO(), "ReplicaCount")
	assert.Equal(t, metrics[0].MetricName, "ReplicaCount")
	if currentHour%2 == 0 {
		assert.Equal(t, metrics[0].Value.Value(), int64(10))
	} else {
		assert.Equal(t, metrics[0].Value.Value(), int64(0))
	}
}

func TestCronGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range cronMetricIdentifiers {
		meta, err := parseCronMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		startSchedule, _ := parser.Parse(meta.Start)
		endSchedule, _ := parser.Parse(meta.End)

		mockCronScaler := cronScaler{
			metricType:    "",
			metadata:      meta,
			logger:        logr.Discard(),
			startSchedule: startSchedule,
			endSchedule:   endSchedule,
		}

		metricSpec := mockCronScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestExpandCronHash(t *testing.T) {
	type testData struct {
		expr    string
		pattern string
	}
	tests := []testData{
		{"0 8 * * *", `^0 8 \* \* \*$`},
		{"H 8 * * MON-FRI", `^([0-9]|[1-5][0-9]) 8 \* \* MON-FRI$`},
		{"H(0-29) H(6-7) * * *", `^([0-9]|[12][0-9]) [67] \* \* \*$`},
		{"H/15 * * * *", `^([0-9]|1[0-4])-59/15 \* \* \* \*$`},
		{"H(10-40)/10 * * * *", `^[1-3][0-9]-40/10 \* \* \* \*$`},
		{"0 0 H * *", `^0 0 ([1-9]|1[0-9]|2[0-8]) \* \*$`},
		{"0,H 8 * * *", `^0,([0-9]|[1-5][0-9]) 8 \* \* \*$`},
		{"CRON_TZ=Etc/UTC H 8 * * *", `^CRON_TZ=Etc/UTC ([0-9]|[1-5][0-9]) 8 \* \* \*$`},
		{"TZ=Etc/UTC H 8 * * *", `^TZ=Etc/UTC ([0-9]|[1-5][0-9]) 8 \* \* \*$`},
		{"H/1 8 * * *", `^([0-9]|[1-5][0-9]) 8 \* \* \*$`},
		{"H(10-20)/1 8 * * *", `^(1[0-9]|20) 8 \* \* \*$`},
	}
	for _, tt := range tests {
		expanded, err := expandCronHash(tt.expr, "default/app/0")
		assert.NoError(t, err)
		assert.Regexp(t, tt.pattern, expanded, tt.expr)
	}
}

func TestExpandCronHashSeed(t *testing.T) {
	bare, _ := expandCronHash("H H(10-20) * * *", "default/app/0")
	stepOne, _ := expandCronHash("H/1 H(10-20)/1 * * *", "default/app/0")
	assert.Equal(t, bare, stepOne)

	start, _ := expandCronHash("H 8 * * *", "default/app/0")
	end, _ := expandCronHash("H 17 * * *", "default/app/0")
	assert.Equal(t, strings.Fields(start)[0], strings.Fields(end)[0])

	minutes := map[string]struct{}{}
	for i := 0; i < 20; i++ {
		expanded, _ := expandCronHash("H 8 * * *", fmt.Sprintf("default/app-%d/0", i))
		minutes[strings.Fields(expanded)[0]] = struct{}{}
	}
	assert.Greater(t, len(minutes), 1)
}

func TestCronHashSameSchedule(t *testing.T) {
	for _, end := range []string{"H(0-59) 8 * * *", "H/60 8 * * *"} {
		_, err := NewCronScaler(&scalersconfig.ScalerConfig{TriggerMetadata: map[string]string{
			"timezone":        "Etc/UTC",
			"start":           "H 8 * * *",
			"end":             end,
			"desiredReplicas": "10",
		}})
		assert.Error(t, err, end)
	}
}
