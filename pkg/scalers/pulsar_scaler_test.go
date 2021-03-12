package scalers

import (
	"fmt"
	"testing"
)

type parsePulsarMetadataTestData struct {
	metadata     map[string]string
	isError      bool
	isActive     bool
	statsUrl     string
	tenant       string
	namespace    string
	topic        string
	subscription string
}

type parsePulsarAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  bool
	cert       string
	key        string
	ca         string
}

type pulsarMetricIdentifier struct {
	metadataTestData *parsePulsarMetadataTestData
	name             string
}

// A complete valid metadata example for reference
var validPulsarMetadata = map[string]string{
	"statsUrl":     "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats",
	"tenant":       "apache",
	"namespace":    "pulsar",
	"topic":        "my-topic",
	"subscription": "sub1",
}

// A complete valid authParams example for sasl, with username and passwd
var validPulsarWithAuthParams = map[string]string{
	"tls":  "enable",
	"cert": "admin.cert.pem",
	"key":  "admin-pk8.pem",
	"ca":   "ca.cert.pem",
}

// A complete valid authParams example for sasl, without username and passwd
var validPulsarWithoutAuthParams = map[string]string{}

var parsePulsarMetadataTestDataset = []parsePulsarMetadataTestData{
	// failure, no statsUrl
	{map[string]string{}, true, false, "", "", "", "", ""},
	{map[string]string{"statsUrl": "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats"}, true, false, "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "", "", "", ""},
	{map[string]string{"statsUrl": "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "tenant": "apache"}, true, false, "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "", "", ""},
	{map[string]string{"statsUrl": "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "tenant": "apache", "namespace": "pulsar"}, true, false, "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "pulsar", "", ""},
	{map[string]string{"statsUrl": "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "tenant": "apache", "namespace": "pulsar", "topic": "my-topic"}, true, false, "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "pulsar", "my-topic", ""},
	{map[string]string{"statsUrl": "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "tenant": "apache", "namespace": "pulsar", "topic": "my-topic", "subscription": "sub1"}, false, true, "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "pulsar", "my-topic", "sub1"},
	{map[string]string{"statsUrl": "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "tenant": "apache", "namespace": "pulsar", "topic": "my-topic", "subscription": "sub2"}, false, true, "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "pulsar", "my-topic", "sub2"},
	{map[string]string{"statsUrl": "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "tenant": "apache", "namespace": "pulsar", "topic": "my-topic", "subscription": "sub3"}, false, false, "http://172.20.0.151:80/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "pulsar", "my-topic", "sub3"},
	{map[string]string{"statsUrl": "http://127.0.0.1:8080/admin/v2/persistent/apache/pulsar/my-topic/stats", "tenant": "apache", "namespace": "pulsar", "topic": "my-topic", "subscription": "sub1"}, false, false, "http://127.0.0.1:8080/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "pulsar", "my-topic", "sub1"},

	// tls
	{map[string]string{"statsUrl": "https://localhost:8443/admin/v2/persistent/apache/pulsar/my-topic/stats", "tls": "enable", "cert": "admin.cert.pem", "key": "admin-pk8.pem", "ca": "ca.cert.pem", "tenant": "apache", "namespace": "pulsar", "topic": "my-topic", "subscription": "sub1"}, false, true, "https://localhost:8443/admin/v2/persistent/apache/pulsar/my-topic/stats", "apache", "pulsar", "my-topic", "sub1"},
}

var parsePulsarMetadataTestAuthTlsDataset = []parsePulsarAuthParamsTestData{
	// failure, no statsUrl
	{map[string]string{"tls": "enable", "cert": "admin.cert.pem", "key": "admin-pk8.pem", "ca": "ca.cert.pem"}, false, true, "admin.cert.pem", "admin-pk8.pem", "ca.cert.pem"},
}

var pulsarMetricIdentifiers = []pulsarMetricIdentifier{
	{&parsePulsarMetadataTestDataset[5], "pulsar-apache-pulsar-my-topic-sub1"},
}

func TestParsePulsarMetadata(t *testing.T) {
	for _, testData := range parsePulsarMetadataTestDataset {
		meta, err := parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validPulsarWithAuthParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if meta.statsUrl != testData.statsUrl {
			t.Errorf("Expected statsUrl %s but got %s\n", testData.statsUrl, meta.statsUrl)
		}

		if meta.tenant != testData.tenant {
			t.Errorf("Expected tenant %s but got %s\n", testData.tenant, meta.tenant)
		}

		if meta.namespace != testData.namespace {
			t.Errorf("Expected namespace %s but got %s\n", testData.namespace, meta.namespace)
		}

		if meta.topic != testData.topic {
			t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.topic)
		}

		if meta.subscription != testData.subscription {
			t.Errorf("Expected subscription %s but got %s\n", testData.subscription, meta.subscription)
		}

		meta, err = parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validPulsarWithoutAuthParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if meta.statsUrl != testData.statsUrl {
			t.Errorf("Expected statsUrl %s but got %s\n", testData.statsUrl, meta.statsUrl)
		}

		if meta.tenant != testData.tenant {
			t.Errorf("Expected tenant %s but got %s\n", testData.tenant, meta.tenant)
		}

		if meta.namespace != testData.namespace {
			t.Errorf("Expected namespace %s but got %s\n", testData.namespace, meta.namespace)
		}

		if meta.topic != testData.topic {
			t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.topic)
		}

		if meta.subscription != testData.subscription {
			t.Errorf("Expected subscription %s but got %s\n", testData.subscription, meta.subscription)
		}

	}
}

func TestPulsarAuthParams(t *testing.T) {
	for _, testData := range parsePulsarMetadataTestAuthTlsDataset {
		meta, err := parsePulsarMetadata(&ScalerConfig{TriggerMetadata: validPulsarMetadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta.enableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, meta.enableTLS)
		}

		if meta.ca != testData.ca {
			t.Errorf("Expected ca to be set to %s but got %s\n", testData.ca, meta.ca)
		}

		if meta.cert != testData.cert {
			t.Errorf("Expected cert to be set to %s but got %s\n", testData.cert, meta.cert)
		}

		if meta.key != testData.key {
			t.Errorf("Expected key to be set to %s but got %s\n", testData.key, meta.key)
		}
	}
}

func TestPulsarGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range pulsarMetricIdentifiers {
		meta, err := parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validWithAuthParams})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockPulsarScaler := pulsarScaler{meta, nil}

		metricSpec := mockPulsarScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestPulsarIsActive(t *testing.T) {
	for _, testData := range pulsarMetricIdentifiers {

		mockPulsarScaler, err := NewPulsarScaler(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validPulsarWithoutAuthParams})
		if err != nil {
			t.Fatal("Failed:", err)
		}

		active, err := mockPulsarScaler.IsActive(nil)
		if err != nil {
			t.Fatal("Failed:", err)
		}

		if active != testData.metadataTestData.isActive {
			t.Errorf("Expected %t got %t", testData.metadataTestData.isActive, active)
		}

	}
}

func TestPulsarIsActiveWithAuthParams(t *testing.T) {
	for _, testData := range pulsarMetricIdentifiers {

		mockPulsarScaler, err := NewPulsarScaler(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validPulsarWithAuthParams})
		if err != nil {
			t.Fatal("Failed:", err)
		}

		active, err := mockPulsarScaler.IsActive(nil)
		if err != nil {
			t.Fatal("Failed:", err)
		}

		if active != testData.metadataTestData.isActive {
			t.Errorf("Expected %t got %t", testData.metadataTestData.isActive, active)
		}

	}
}

func TestPulsarGetMetric(t *testing.T) {
	for _, testData := range pulsarMetricIdentifiers {

		mockPulsarScaler, err := NewPulsarScaler(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validPulsarWithoutAuthParams})
		if err != nil {
			t.Fatal("Failed:", err)
		}

		metricSpec := mockPulsarScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name

		metric, err := mockPulsarScaler.GetMetrics(nil, metricName, nil)
		if err != nil {
			t.Fatal("Failed:", err)
		}

		fmt.Printf("%+v\n", metric)
	}
}
