package scalers

import (
	"context"
	"fmt"
	"testing"
)

type parseNewRelicMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type newrelicMetricIdentifier struct {
	metadataTestData *parseNewRelicMetadataTestData
	scalerIndex      int
	name             string
}

var testNewRelicMetadata = []parseNewRelicMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	// all properly formed
	{map[string]string{"account": "0", "region": "EU", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	// account as String
	{map[string]string{"account": "ABC", "metricName": "results", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// missing account
	{map[string]string{"metricName": "results", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// missing metricName
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	// malformed threshold
	{map[string]string{"account": "0", "threshold": "one", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// missing threshold
	{map[string]string{"account": "0", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// missing query
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey"}, true},
	// noDataError invalid value
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "invalid", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// noDataError valid value
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "true", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "false", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "0", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "1", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
}

var newrelicMetricIdentifiers = []newrelicMetricIdentifier{
	{&testNewRelicMetadata[1], 0, "s0-new-relic"},
	{&testNewRelicMetadata[1], 1, "s1-new-relic"},
}

func TestNewRelicParseMetadata(t *testing.T) {
	for _, testData := range testNewRelicMetadata {
		_, err := parseNewRelicMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected error but got success")
		}
	}
}
func TestNewRelicGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range newrelicMetricIdentifiers {
		meta, err := parseNewRelicMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockNewRelicScaler := newrelicScaler{
			metadata: meta,
			nrClient: nil,
		}

		metricSpec := mockNewRelicScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
