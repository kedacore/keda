package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseHoneycombMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type honeycombMetricIdentifier struct {
	metadataTestData *parseHoneycombMetadataTestData
	triggerIndex     int
	name             string
}

var testHoneycombMetadata = []parseHoneycombMetadataTestData{
	// missing everything
	{map[string]string{}, map[string]string{}, true},
	// minimal valid
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "10"}, map[string]string{}, false},
	// with calculation and timeRange
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "10", "calculation": "SUM", "timeRange": "120"}, map[string]string{}, false},
	// missing apiKey
	{map[string]string{"dataset": "ds", "threshold": "10"}, map[string]string{}, true},
	// missing dataset
	{map[string]string{"apiKey": "abc", "threshold": "10"}, map[string]string{}, true},
	// missing threshold
	{map[string]string{"apiKey": "abc", "dataset": "ds"}, map[string]string{}, true},
	// invalid threshold
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "notanumber"}, map[string]string{}, true},
	// minimal with queryRaw (valid JSON)
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "10", "queryRaw": `{"breakdowns":["ua"],"calculations":[{"op":"COUNT"}]}`}, map[string]string{}, false},
	// minimal with queryRaw (invalid JSON)
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "10", "queryRaw": `not-json`}, map[string]string{}, true},
}

var honeycombMetricIdentifiers = []honeycombMetricIdentifier{
	{&testHoneycombMetadata[1], 0, "s0-honeycomb"},
	{&testHoneycombMetadata[2], 1, "s1-honeycomb"},
}

func TestHoneycombParseMetadata(t *testing.T) {
	for i, testData := range testHoneycombMetadata {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			cfg := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
			}
			_, err := parseHoneycombMetadata(cfg)
			if err != nil && !testData.isError {
				t.Errorf("Test case %d: Expected success but got error: %v\nMetadata: %v\nAuthParams: %v", i, err, testData.metadata, testData.authParams)
			}
			if testData.isError && err == nil {
				t.Errorf("Test case %d: Expected error but got success\nMetadata: %v\nAuthParams: %v", i, testData.metadata, testData.authParams)
			}
		})
	}
}

func TestHoneycombGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range honeycombMetricIdentifiers {
		cfg := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		}
		meta, err := parseHoneycombMetadata(cfg)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockScaler := honeycombScaler{
			metadata:   meta,
			logger:     logr.Discard(),
			metricType: v2.AverageValueMetricType,
			httpClient: nil,
		}
		metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestExtractResultField(t *testing.T) {
	// 1. Use resultField, field present, numeric
	results := []map[string]interface{}{
		{"data": map[string]interface{}{"COUNT": float64(42), "foo": "bar"}},
	}
	val, err := extractResultField(results, "COUNT")
	if err != nil || val != 42 {
		t.Errorf("Expected 42, got %v, err: %v", val, err)
	}

	// 2. Use resultField, field not present
	_, err = extractResultField(results, "DOES_NOT_EXIST")
	if err == nil {
		t.Error("Expected error for missing result field, got nil")
	}

	// 3. Use resultField, field present but not numeric
	badResults := []map[string]interface{}{
		{"data": map[string]interface{}{"COUNT": "not-a-number"}},
	}
	_, err = extractResultField(badResults, "COUNT")
	if err == nil {
		t.Error("Expected error for non-numeric result field, got nil")
	}

	// 4. resultField empty, fallback to first numeric
	results = []map[string]interface{}{
		{"data": map[string]interface{}{"foo": "bar", "COUNT": float64(99)}},
	}
	val, err = extractResultField(results, "")
	if err != nil || val != 99 {
		t.Errorf("Expected 99, got %v, err: %v", val, err)
	}

	// 5. No numeric value at all
	results = []map[string]interface{}{
		{"data": map[string]interface{}{"foo": "bar"}},
	}
	_, err = extractResultField(results, "")
	if err == nil {
		t.Error("Expected error for no numeric value found, got nil")
	}

	// 6. Empty results
	results = []map[string]interface{}{}
	_, err = extractResultField(results, "")
	if err == nil {
		t.Error("Expected error for empty results, got nil")
	}

	// 7. No "data" key
	results = []map[string]interface{}{
		{"COUNT": float64(77)},
	}
	_, err = extractResultField(results, "COUNT")
	if err == nil {
		t.Error("Expected error for missing 'data' key, got nil")
	}

	// 8. Multiple result rows, should only consider the first
	results = []map[string]interface{}{
		{"data": map[string]interface{}{"COUNT": float64(3)}},
		{"data": map[string]interface{}{"COUNT": float64(999)}},
	}
	val, err = extractResultField(results, "COUNT")
	if err != nil || val != 3 {
		t.Errorf("Expected 3 from first row, got %v, err: %v", val, err)
	}
}

func TestParseHoneycombMetadata_QueryRawOverridesFields(t *testing.T) {
	rawQuery := `{"filters":[{"column":"foo","op":"=","value":"bar"}]}`
	metaMap := map[string]string{
		"apiKey":     "abc",
		"dataset":    "ds",
		"threshold":  "10",
		"queryRaw":   rawQuery,
		"breakdowns": "shouldBeIgnored",
	}
	cfg := &scalersconfig.ScalerConfig{
		TriggerMetadata: metaMap,
	}
	meta, err := parseHoneycombMetadata(cfg)
	if err != nil {
		t.Fatal("Expected success, got error:", err)
	}
	// The meta.Query should match what's in rawQuery (not necessarily byte-for-byte, but same structure)
	b, _ := json.Marshal(meta.Query)
	var rawQ map[string]interface{}
	_ = json.Unmarshal([]byte(rawQuery), &rawQ)
	var parsedQ map[string]interface{}
	_ = json.Unmarshal(b, &parsedQ)
	if fmt.Sprintf("%v", rawQ) != fmt.Sprintf("%v", parsedQ) {
		t.Errorf("Expected QueryRaw to override other fields. Got: %s", string(b))
	}
}
