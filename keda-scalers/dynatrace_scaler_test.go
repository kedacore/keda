package scalers

import (
	"context"
	"fmt"
	"testing"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type dynatraceMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	errorCase  bool
}

type dynatraceMetricIdentifier struct {
	metadataTestData *dynatraceMetadataTestData
	triggerIndex     int
	name             string
}

var testDynatraceMetadata = []dynatraceMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{map[string]string{"threshold": "100", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, false},
	// malformed threshold
	{map[string]string{"threshold": "abc", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// malformed activationThreshold
	{map[string]string{"activationThreshold": "abc", "threshold": "100", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// missing threshold
	{map[string]string{"metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// missing metricsSelector
	{map[string]string{"threshold": "100"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// missing token (must come from auth params)
	{map[string]string{"token": "foo", "threshold": "100", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234"}, true},
}

var dynatraceMetricIdentifiers = []dynatraceMetricIdentifier{
	{&testDynatraceMetadata[1], 0, "s0-dynatrace"},
	{&testDynatraceMetadata[1], 1, "s1-dynatrace"},
}

func TestDynatraceParseMetadata(t *testing.T) {
	for _, testData := range testDynatraceMetadata {
		_, err := parseDynatraceMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.errorCase {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected success but got error", err)
		}
		if testData.errorCase && err == nil {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected error but got success")
		}
	}
}
func TestDynatraceGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range dynatraceMetricIdentifiers {
		meta, err := parseDynatraceMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockNewRelicScaler := dynatraceScaler{
			metadata:   meta,
			httpClient: nil,
		}

		metricSpec := mockNewRelicScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
