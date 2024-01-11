package scalers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
)

type parseNewRelicMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type newrelicMetricIdentifier struct {
	metadataTestData *parseNewRelicMetadataTestData
	triggerIndex     int
	name             string
}

var testNewRelicMetadata = []parseNewRelicMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, false},
	// all properly formed
	{map[string]string{"account": "0", "region": "EU", "threshold": "100", "activationThreshold": "20", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, false},
	// account passed via auth params
	{map[string]string{"region": "EU", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{"account": "0"}, false},
	// region passed via auth params
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{"region": "EU"}, false},
	// account as String
	{map[string]string{"account": "ABC", "threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, true},
	// missing account
	{map[string]string{"threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, true},
	// missing account
	{map[string]string{"threshold": "100", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, true},
	// malformed threshold
	{map[string]string{"account": "0", "threshold": "one", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, true},
	// malformed activationThreshold
	{map[string]string{"account": "0", "threshold": "100", "activationThreshold": "notanint", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, true},
	// missing threshold
	{map[string]string{"account": "0", "queryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, true},
	// missing query
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey"}, map[string]string{}, true},
	// noDataError invalid value
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "invalid", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, true},
	// noDataError valid value
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "true", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, false},
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "false", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, false},
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "0", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, false},
	{map[string]string{"account": "0", "threshold": "100", "queryKey": "somekey", "noDataError": "1", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, map[string]string{}, false},
}

var newrelicMetricIdentifiers = []newrelicMetricIdentifier{
	{&testNewRelicMetadata[1], 0, "s0-new-relic"},
	{&testNewRelicMetadata[1], 1, "s1-new-relic"},
}

func TestNewRelicParseMetadata(t *testing.T) {
	for _, testData := range testNewRelicMetadata {
		_, err := parseNewRelicMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams}, logr.Discard())
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
		meta, err := parseNewRelicMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex}, logr.Discard())
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
