package scalers

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
)

type parseEtcdMetadataTestData struct {
	metadata  map[string]string
	endpoints []string
	isError   bool
}

type parseEtcdAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

type etcdMetricIdentifier struct {
	metadataTestData *parseEtcdMetadataTestData
	triggerIndex     int
	name             string
}

// A complete valid metadata example for reference
var validEtcdMetadata = map[string]string{
	"endpoints":                   "172.0.0.1:2379,172.0.0.2:2379,172.0.0.3:2379",
	"watchKey":                    "length",
	"value":                       "5.5",
	"activationValue":             "0.5",
	"watchProgressNotifyInterval": "600",
}

var parseEtcdMetadataTestDataset = []parseEtcdMetadataTestData{
	// success
	{map[string]string{"endpoints": "172.0.0.1:2379,172.0.0.2:2379,172.0.0.3:2379", "watchKey": "length", "value": "5.5", "activationValue": "0.5", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379", "172.0.0.2:2379", "172.0.0.3:2379"}, false},
	// success
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "var", "value": "5.5", "activationValue": "0.5", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, false},
	// failure, endpoints missed
	{map[string]string{"endpoints": "", "watchKey": "length", "value": "5", "activationValue": "0", "watchProgressNotifyInterval": "600"}, []string{""}, true},
	// failure, watchKey missed
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "", "value": "5", "activationValue": "0", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, true},
	// failure, value invalid
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "length", "value": "a", "activationValue": "0", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, true},
	// failure, activationValue invalid
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "length", "value": "5", "activationValue": "b", "watchProgressNotifyInterval": "600"}, []string{"172.0.0.1:2379"}, true},
	// failure, watchProgressNotifyInterval invalid
	{map[string]string{"endpoints": "172.0.0.1:2379", "watchKey": "length", "value": "5", "activationValue": "0", "watchProgressNotifyInterval": "0"}, []string{"172.0.0.1:2379"}, true},
}

var parseEtcdAuthParamsTestDataset = []parseEtcdAuthParamsTestData{
	// success, TLS only
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, true},
	// success, TLS CA only
	{map[string]string{"tls": "enable", "ca": "caaa"}, false, true},
	// failure, TLS missing cert
	{map[string]string{"tls": "enable", "ca": "caaa", "key": "keey"}, true, false},
	// failure, TLS missing key
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert"}, true, false},
	// failure, TLS invalid
	{map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
}

var etcdMetricIdentifiers = []etcdMetricIdentifier{
	{&parseEtcdMetadataTestDataset[0], 0, "s0-etcd-length"},
	{&parseEtcdMetadataTestDataset[1], 1, "s1-etcd-var"},
}

func TestParseEtcdMetadata(t *testing.T) {
	for _, testData := range parseEtcdMetadataTestDataset {
		meta, err := parseEtcdMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if err == nil && !reflect.DeepEqual(meta.endpoints, testData.endpoints) {
			t.Errorf("Expected  %v but got %v\n", testData.endpoints, meta.endpoints)
		}
	}
}

func TestParseEtcdAuthParams(t *testing.T) {
	for _, testData := range parseEtcdAuthParamsTestDataset {
		meta, err := parseEtcdMetadata(&ScalerConfig{TriggerMetadata: validEtcdMetadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta.enableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, meta.enableTLS)
		}
		if meta.enableTLS {
			if meta.ca != testData.authParams["ca"] {
				t.Errorf("Expected ca to be set to %v but got %v\n", testData.authParams["ca"], meta.enableTLS)
			}
			if meta.cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %v but got %v\n", testData.authParams["cert"], meta.cert)
			}
			if meta.key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["key"], meta.key)
			}
			if meta.keyPassword != testData.authParams["keyPassword"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["keyPassword"], meta.key)
			}
		}
	}
}

func TestEtcdGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range etcdMetricIdentifiers {
		meta, err := parseEtcdMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockEtcdScaler := etcdScaler{"", meta, nil, logr.Logger{}}

		metricSpec := mockEtcdScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
