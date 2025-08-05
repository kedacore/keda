package scalers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
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
	// all properly formed with region and activationThreshold
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
	// noDataError valid values
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
	for i, testData := range testNewRelicMetadata {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			_, err := parseNewRelicMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
			})
			if err != nil && !testData.isError {
				t.Errorf("Test case %d: Expected success but got error: %v\nMetadata: %v\nAuthParams: %v",
					i, err, testData.metadata, testData.authParams)
			}
			if testData.isError && err == nil {
				t.Errorf("Test case %d: Expected error but got success\nMetadata: %v\nAuthParams: %v",
					i, testData.metadata, testData.authParams)
			}
		})
	}
}

func TestNewRelicGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range newrelicMetricIdentifiers {
		meta, err := parseNewRelicMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockNewRelicScaler := newrelicScaler{
			metadata:   meta,
			nrClient:   nil,
			logger:     logr.Discard(),
			metricType: v2.AverageValueMetricType,
		}

		metricSpec := mockNewRelicScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
