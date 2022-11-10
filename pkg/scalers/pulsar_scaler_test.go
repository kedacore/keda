package scalers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
)

type parsePulsarMetadataTestData struct {
	metadata           map[string]string
	isError            bool
	isActive           bool
	isPartitionedTopic bool
	adminURL           string
	topic              string
	subscription       string
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
	"adminURL":     "http://172.20.0.151:80",
	"topic":        "persistent://public/default/my-topic",
	"subscription": "sub1",
	"tls":          "enable",
}

// A complete valid authParams example for sasl, with username and passwd
var validPulsarWithAuthParams = map[string]string{
	"cert": "certdata",
	"key":  "keydata",
	"ca":   "cadata",
}

// A complete valid authParams example for sasl, without username and passwd
var validPulsarWithoutAuthParams = map[string]string{}

var parsePulsarMetadataTestDataset = []parsePulsarMetadataTestData{
	// failure, no adminURL
	{map[string]string{}, true, false, false, "", "", ""},
	{map[string]string{"adminURL": "http://172.20.0.151:80"}, true, false, false, "http://172.20.0.151:80", "", ""},
	{map[string]string{"adminURL": "http://172.20.0.151:80"}, true, false, false, "http://172.20.0.151:80", "", ""},
	{map[string]string{"adminURL": "http://172.20.0.151:80"}, true, false, false, "http://172.20.0.151:80", "", ""},
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic"}, true, false, false, "http://172.20.0.151:80", "persistent://public/default/my-topic", ""},
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1"}, false, true, false, "http://172.20.0.151:80", "persistent://public/default/my-topic", "sub1"},
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub2"}, false, true, false, "http://172.20.0.151:80", "persistent://public/default/my-topic", "sub2"},
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub3"}, false, false, false, "http://172.20.0.151:80", "persistent://public/default/my-topic", "sub3"},
	{map[string]string{"adminURL": "http://127.0.0.1:8080", "topic": "persistent://public/default/my-topic", "subscription": "sub1"}, false, false, false, "http://127.0.0.1:8080", "persistent://public/default/my-topic", "sub1"},
	{map[string]string{"adminURL": "http://127.0.0.1:8080", "topic": "persistent://public/default/my-topic", "subscription": "sub1"}, false, false, false, "http://127.0.0.1:8080", "persistent://public/default/my-topic", "sub1"},
	{map[string]string{"adminURL": "http://127.0.0.1:8080", "topic": "persistent://public/default/my-topic", "isPartitionedTopic": "true", "subscription": "sub1"}, false, false, true, "http://127.0.0.1:8080", "persistent://public/default/my-topic", "sub1"},

	// tls
	{map[string]string{"adminURL": "https://localhost:8443", "tls": "enable", "cert": "certdata", "key": "keydata", "ca": "cadata", "topic": "persistent://public/default/my-topic", "subscription": "sub1"}, false, true, false, "https://localhost:8443", "persistent://public/default/my-topic", "sub1"},
}

var parsePulsarMetadataTestAuthTLSDataset = []parsePulsarAuthParamsTestData{
	// failure, no adminURL
	{map[string]string{"cert": "certdata", "key": "keydata", "ca": "cadata"}, false, true, "certdata", "keydata", "cadata"},
}

var pulsarMetricIdentifiers = []pulsarMetricIdentifier{
	{&parsePulsarMetadataTestDataset[0], "pulsar-apache-pulsar-my-topic-sub1"},
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

		if meta.adminURL != testData.adminURL {
			t.Errorf("Expected adminURL %s but got %s\n", testData.adminURL, meta.adminURL)
		}

		if !testData.isError {
			if testData.isPartitionedTopic {
				if !strings.HasSuffix(meta.statsURL, "/partitioned-stats") {
					t.Errorf("Expected statsURL to end with /partitioned-stats but got %s\n", meta.statsURL)
				}
			} else {
				if !strings.HasSuffix(meta.statsURL, "/stats") {
					t.Errorf("Expected statsURL to end with /stats but got %s\n", meta.statsURL)
				}
			}
		}

		if meta.topic != testData.topic {
			t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.topic)
		}

		if meta.subscription != testData.subscription {
			t.Errorf("Expected subscription %s but got %s\n", testData.subscription, meta.subscription)
		}

		authParams := validPulsarWithoutAuthParams
		if k, ok := testData.metadata["tls"]; ok && k == "enable" {
			authParams = validPulsarWithAuthParams
		}

		meta, err = parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if meta.adminURL != testData.adminURL {
			t.Errorf("Expected adminURL %s but got %s\n", testData.adminURL, meta.adminURL)
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
	for _, testData := range parsePulsarMetadataTestAuthTLSDataset {
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
			if testData.metadataTestData.isError {
				continue
			}
			t.Fatal("Could not parse metadata:", err)
		}
		mockPulsarScaler := pulsarScaler{meta, nil, logr.Discard()}

		metricSpec := mockPulsarScaler.GetMetricSpecForScaling(context.TODO())
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
			if testData.metadataTestData.isError {
				continue
			}
			t.Fatal("Failed:", err)
		}

		active, err := mockPulsarScaler.IsActive(context.TODO())
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
			if testData.metadataTestData.isError {
				continue
			}
			t.Fatal("Failed:", err)
		}

		active, err := mockPulsarScaler.IsActive(context.TODO())
		if err != nil && !testData.metadataTestData.isError {
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
			if testData.metadataTestData.isError {
				continue
			}
			t.Fatal("Failed:", err)
		}

		metricSpec := mockPulsarScaler.GetMetricSpecForScaling(context.TODO())
		metricName := metricSpec[0].External.Metric.Name

		metric, err := mockPulsarScaler.GetMetrics(context.TODO(), metricName, nil)
		if err != nil {
			t.Fatal("Failed:", err)
		}

		fmt.Printf("%+v\n", metric)
	}
}
