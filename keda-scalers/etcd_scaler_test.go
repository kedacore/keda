package scalers

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type parseEtcdMetadataTestData struct {
	metadata  map[string]string
	endpoints []string
	isError   bool
}

type parseEtcdAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  string
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
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, etcdTLSEnable},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey"}, false, etcdTLSEnable},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, etcdTLSEnable},
	// success, TLS CA only
	{map[string]string{"tls": "enable", "ca": "caaa"}, false, etcdTLSEnable},
	// failure, TLS missing cert
	{map[string]string{"tls": "enable", "ca": "caaa", "key": "keey"}, true, etcdTLSDisable},
	// failure, TLS missing key
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert"}, true, etcdTLSDisable},
	// failure, TLS invalid
	{map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, etcdTLSDisable},
	// success, username and password
	{map[string]string{"username": "root", "password": "admin"}, false, etcdTLSDisable},
	// failure, missing password
	{map[string]string{"username": "root"}, true, etcdTLSDisable},
	// failure, missing username
	{map[string]string{"password": "admin"}, true, etcdTLSDisable},
}

var etcdMetricIdentifiers = []etcdMetricIdentifier{
	{&parseEtcdMetadataTestDataset[0], 0, "s0-etcd-length"},
	{&parseEtcdMetadataTestDataset[1], 1, "s1-etcd-var"},
}

func TestParseEtcdMetadata(t *testing.T) {
	for _, testData := range parseEtcdMetadataTestDataset {
		meta, err := parseEtcdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success %v", testData)
		}
		if err == nil && !reflect.DeepEqual(meta.Endpoints, testData.endpoints) {
			t.Errorf("Expected  %v but got %v\n", testData.endpoints, meta.Endpoints)
		}
	}
}

func TestParseEtcdAuthParams(t *testing.T) {
	for _, testData := range parseEtcdAuthParamsTestDataset {
		meta, err := parseEtcdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: validEtcdMetadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta != nil && meta.EnableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, meta.EnableTLS)
		}
		if meta != nil && meta.EnableTLS == etcdTLSEnable {
			if meta.Ca != testData.authParams["ca"] {
				t.Errorf("Expected ca to be set to %v but got %v\n", testData.authParams["ca"], meta.EnableTLS)
			}
			if meta.Cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %v but got %v\n", testData.authParams["cert"], meta.Cert)
			}
			if meta.Key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["key"], meta.Key)
			}
			if meta.KeyPassword != testData.authParams["keyPassword"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["keyPassword"], meta.Key)
			}
		}
	}
}

func TestEtcdGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range etcdMetricIdentifiers {
		meta, err := parseEtcdMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
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
