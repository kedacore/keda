package scalers

import (
	"context"
	"fmt"
	"strconv"
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
	triggerMetadata map[string]string
	authParams      map[string]string
	isError         bool
	enableTLS       bool
	cert            string
	key             string
	ca              string
	bearerToken     string
	username        string
	password        string
}

type pulsarMetricIdentifier struct {
	metadataTestData *parsePulsarMetadataTestData
	name             string
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
	// test metric msgBacklogThreshold
	{map[string]string{"adminURL": "http://127.0.0.1:8080", "topic": "persistent://public/default/my-topic", "isPartitionedTopic": "true", "subscription": "sub1", "msgBacklogThreshold": "5"}, false, false, true, "http://127.0.0.1:8080", "persistent://public/default/my-topic", "sub1"},
	// FIXME: msgBacklog support DEPRECATED to be removed in v2.13
	// test metric msgBacklog
	{map[string]string{"adminURL": "http://127.0.0.1:8080", "topic": "persistent://public/default/my-topic", "isPartitionedTopic": "true", "subscription": "sub1", "msgBacklog": "5"}, false, false, true, "http://127.0.0.1:8080", "persistent://public/default/my-topic", "sub1"},
	// END FIXME

	// tls
	{map[string]string{"adminURL": "https://localhost:8443", "tls": "enable", "cert": "certdata", "key": "keydata", "ca": "cadata", "topic": "persistent://public/default/my-topic", "subscription": "sub1"}, false, true, false, "https://localhost:8443", "persistent://public/default/my-topic", "sub1"},
}

var parsePulsarMetadataTestAuthTLSDataset = []parsePulsarAuthParamsTestData{
	// Passes, mutual TLS, no other auth (legacy "tls: enable")
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "tls": "enable"}, map[string]string{"cert": "certdata", "key": "keydata", "ca": "cadata"}, false, true, "certdata", "keydata", "cadata", "", "", ""},
	// Passes, mutual TLS, no other auth (uses new way to enable tls)
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "tls"}, map[string]string{"cert": "certdata", "key": "keydata", "ca": "cadata"}, false, true, "certdata", "keydata", "cadata", "", "", ""},
	// Fails, mutual TLS (legacy "tls: enable") without cert
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "tls": "enable"}, map[string]string{"cert": "", "key": "keydata", "ca": "cadata"}, true, true, "certdata", "keydata", "cadata", "", "", ""},
	// Fails, mutual TLS, (uses new way to enable tls) without cert
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "tls"}, map[string]string{"cert": "certdata", "key": "", "ca": "cadata"}, true, true, "certdata", "keydata", "cadata", "", "", ""},
	// Passes, server side TLS with bearer token. Note that EnableTLS is expected to be false because it is not mTLS.
	// The legacy behavior required tls: enable in order to configure a custom root ca. Now, all that is required is configuring a root ca.
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "tls": "enable", "authModes": "bearer"}, map[string]string{"ca": "cadata", "bearerToken": "my-special-token"}, false, false, "", "", "cadata", "my-special-token", "", ""},
	// Passes, server side TLS with basic auth. Note that EnableTLS is expected to be false because it is not mTLS.
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "basic"}, map[string]string{"ca": "cadata", "username": "admin", "password": "password123"}, false, false, "", "", "cadata", "", "admin", "password123"},
}

var pulsarMetricIdentifiers = []pulsarMetricIdentifier{
	{&parsePulsarMetadataTestDataset[0], "pulsar-apache-pulsar-my-topic-sub1"},
}

func TestParsePulsarMetadata(t *testing.T) {
	for _, testData := range parsePulsarMetadataTestDataset {
		logger := InitializeLogger(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validPulsarWithAuthParams}, "test_pulsar_scaler")
		meta, err := parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validPulsarWithAuthParams}, logger)

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

		var testDataMsgBacklogThreshold int64
		// FIXME: msgBacklog support DEPRECATED to be removed in v2.13
		if val, ok := testData.metadata["msgBacklog"]; ok {
			testDataMsgBacklogThreshold, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				t.Errorf("error parseing msgBacklog: %v", err)
			}
			// END FiXME
		} else if val, ok := testData.metadata["msgBacklogThreshold"]; ok {
			testDataMsgBacklogThreshold, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				t.Errorf("error parseing msgBacklogThreshold: %v", err)
			}
		} else {
			testDataMsgBacklogThreshold = defaultMsgBacklogThreshold
		}
		if meta.msgBacklogThreshold != testDataMsgBacklogThreshold && testDataMsgBacklogThreshold != defaultMsgBacklogThreshold {
			t.Errorf("Expected msgBacklogThreshold %s but got %d\n", testData.metadata["msgBacklogThreshold"], meta.msgBacklogThreshold)
		}

		authParams := validPulsarWithoutAuthParams
		if k, ok := testData.metadata["tls"]; ok && k == "enable" {
			authParams = validPulsarWithAuthParams
		}

		meta, err = parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: authParams}, logger)

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
		logger := InitializeLogger(&ScalerConfig{TriggerMetadata: testData.triggerMetadata, AuthParams: testData.authParams}, "test_pulsar_scaler")
		meta, err := parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.triggerMetadata, AuthParams: testData.authParams}, logger)

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", testData.authParams, err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if meta.pulsarAuth == nil {
			t.Log("meta.pulsarAuth is nil, skipping rest of validation of", testData)
			continue
		}

		if meta.pulsarAuth.EnableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, meta.pulsarAuth.EnableTLS)
		}

		if meta.pulsarAuth.CA != testData.ca {
			t.Errorf("Expected ca to be set to %s but got %s\n", testData.ca, meta.pulsarAuth.CA)
		}

		if meta.pulsarAuth.Cert != testData.cert {
			t.Errorf("Expected cert to be set to %s but got %s\n", testData.cert, meta.pulsarAuth.Cert)
		}

		if meta.pulsarAuth.Key != testData.key {
			t.Errorf("Expected key to be set to %s but got %s\n", testData.key, meta.pulsarAuth.Key)
		}

		if meta.pulsarAuth.EnableBearerAuth != (testData.bearerToken != "") {
			t.Errorf("Expected EnableBearerAuth to be true when bearerToken is %s\n", testData.bearerToken)
		}

		if meta.pulsarAuth.BearerToken != testData.bearerToken {
			t.Errorf("Expected bearer token to be set to %s but got %s\n", testData.bearerToken, meta.pulsarAuth.BearerToken)
		}

		if meta.pulsarAuth.EnableBasicAuth != (testData.username != "" || testData.password != "") {
			t.Errorf("Expected EnableBearerAuth to be true when bearerToken is %s\n", testData.bearerToken)
		}

		if meta.pulsarAuth.Username != testData.username {
			t.Errorf("Expected username to be set to %s but got %s\n", testData.username, meta.pulsarAuth.Username)
		}

		if meta.pulsarAuth.Password != testData.password {
			t.Errorf("Expected password to be set to %s but got %s\n", testData.password, meta.pulsarAuth.Password)
		}
	}
}

func TestPulsarGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range pulsarMetricIdentifiers {
		logger := InitializeLogger(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validWithAuthParams}, "test_pulsar_scaler")
		meta, err := parsePulsarMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validWithAuthParams}, logger)
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

		metricSpec := mockPulsarScaler.GetMetricSpecForScaling(context.TODO())
		metricName := metricSpec[0].External.Metric.Name

		_, active, err := mockPulsarScaler.GetMetricsAndActivity(context.TODO(), metricName)
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

		metricSpec := mockPulsarScaler.GetMetricSpecForScaling(context.TODO())
		metricName := metricSpec[0].External.Metric.Name

		_, active, err := mockPulsarScaler.GetMetricsAndActivity(context.TODO(), metricName)
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

		metric, _, err := mockPulsarScaler.GetMetricsAndActivity(context.TODO(), metricName)
		if err != nil {
			t.Fatal("Failed:", err)
		}

		fmt.Printf("%+v\n", metric)
	}
}
