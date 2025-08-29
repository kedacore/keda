package scalers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	triggerMetadata        map[string]string
	authParams             map[string]string
	isError                bool
	enableTLS              bool
	cert                   string
	key                    string
	ca                     string
	bearerToken            string
	username               string
	password               string
	enableOAuth            bool
	oauthTokenURI          string
	scope                  string
	clientID               string
	clientSecret           string
	endpointParams         string
	expectedEndpointParams map[string][]string
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
	// tls
	{map[string]string{"adminURL": "https://localhost:8443", "tls": "enable", "cert": "certdata", "key": "keydata", "ca": "cadata", "topic": "persistent://public/default/my-topic", "subscription": "sub1"}, false, true, false, "https://localhost:8443", "persistent://public/default/my-topic", "sub1"},
}

var parsePulsarMetadataTestAuthTLSDataset = []parsePulsarAuthParamsTestData{
	// Passes, mutual TLS, no other auth (legacy "tls: enable")
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "tls": "enable"}, map[string]string{"cert": "certdata", "key": "keydata", "ca": "cadata"}, false, true, "certdata", "keydata", "cadata", "", "", "", false, "", "", "", "", "", nil},
	// Passes, mutual TLS, no other auth (uses new way to enable tls)
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "tls"}, map[string]string{"cert": "certdata", "key": "keydata", "ca": "cadata"}, false, true, "certdata", "keydata", "cadata", "", "", "", false, "", "", "", "", "", nil},
	// Fails, mutual TLS (legacy "tls: enable") without cert
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "tls": "enable"}, map[string]string{"cert": "", "key": "keydata", "ca": "cadata"}, true, true, "certdata", "keydata", "cadata", "", "", "", false, "", "", "", "", "", nil},
	// Fails, mutual TLS, (uses new way to enable tls) without cert
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "tls"}, map[string]string{"cert": "certdata", "key": "", "ca": "cadata"}, true, true, "certdata", "keydata", "cadata", "", "", "", false, "", "", "", "", "", nil},
	// Passes, server side TLS with bearer token. Note that EnableTLS is expected to be false because it is not mTLS.
	// The legacy behavior required tls: enable in order to configure a custom root ca. Now, all that is required is configuring a root ca.
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "tls": "enable", "authModes": "bearer"}, map[string]string{"ca": "cadata", "bearerToken": "my-special-token"}, false, false, "", "", "cadata", "my-special-token", "", "", false, "", "", "", "", "", nil},
	// Passes, server side TLS with basic auth. Note that EnableTLS is expected to be false because it is not mTLS.
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "basic"}, map[string]string{"ca": "cadata", "username": "admin", "password": "password123"}, false, false, "", "", "cadata", "", "admin", "password123", false, "", "", "", "", "", nil},

	// Passes, server side TLS with oauth. Note that EnableTLS is expected to be false because it is not mTLS.
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth"}, map[string]string{"ca": "cadata", "oauthTokenURI": "https1", "scope": "scope1", "clientID": "id1", "clientSecret": "secret123"}, false, false, "", "", "cadata", "", "", "", false, "https1", "scope1", "id1", "secret123", "", nil},
	// Passes, oauth config data is set from metadata only
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth", "oauthTokenURI": "https2", "scope": "scope2", "clientID": "id2"}, map[string]string{"ca": "cadata", "oauthTokenURI": "", "scope": "", "clientID": "", "clientSecret": ""}, false, false, "", "", "cadata", "", "", "", false, "https2", "scope2", "id2", "", "", nil},
	// Passes, oauth config data is set from TriggerAuth if both provided
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth", "oauthTokenURI": "https1", "scope": "scope1", "clientID": "id1"}, map[string]string{"ca": "cadata", "oauthTokenURI": "https3", "scope": "scope3", "clientID": "id3", "clientSecret": "secret123"}, false, false, "", "", "cadata", "", "", "", false, "https3", "scope3", "id3", "secret123", "", nil},
	// Passes, with multiple scopes from metadata
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth", "oauthTokenURI": "https4", "scope": "  sc:scope2, \tsc:scope1 ", "clientID": "id4"}, map[string]string{"ca": "cadata", "oauthTokenURI": "", "scope": "", "clientID": "", "clientSecret": ""}, false, false, "", "", "cadata", "", "", "", false, "https4", "sc:scope1 sc:scope2", "id4", "", "", nil},
	// Passes, with multiple scopes from TriggerAuth
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth"}, map[string]string{"ca": "cadata", "oauthTokenURI": "https5", "scope": " sc:scope2, \tsc:scope1 \n", "clientID": "id5", "clientSecret": "secret123"}, false, false, "", "", "cadata", "", "", "", false, "https5", "sc:scope1 sc:scope2", "id5", "secret123", "", nil},
	// Passes, no scope provided
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth"}, map[string]string{"ca": "cadata", "oauthTokenURI": "https5", "clientID": "id5", "clientSecret": "secret123"}, false, false, "", "", "cadata", "", "", "", false, "https5", "", "id5", "secret123", "", nil},
	// Passes, invalid scopes provided
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth", "scope": "   "}, map[string]string{"ca": "cadata", "oauthTokenURI": "https5", "scope": " , \n", "clientID": "id5", "clientSecret": "secret123"}, false, false, "", "", "cadata", "", "", "", false, "https5", "", "id5", "secret123", "", nil},
	// Passes, with audience provided in endpointParams
	{map[string]string{"adminURL": "http://172.20.0.151:80", "topic": "persistent://public/default/my-topic", "subscription": "sub1", "authModes": "oauth"}, map[string]string{"ca": "cadata", "oauthTokenURI": "https5", "clientID": "id5", "clientSecret": "secret123"}, false, false, "", "", "cadata", "", "", "", false, "https5", "", "id5", "secret123", "audience=abc", map[string][]string{"audience": {"abc"}}},
}

var pulsarMetricIdentifiers = []pulsarMetricIdentifier{
	{&parsePulsarMetadataTestDataset[0], "pulsar-apache-pulsar-my-topic-sub1"},
}

func TestParsePulsarMetadata(t *testing.T) {
	for _, testData := range parsePulsarMetadataTestDataset {
		meta, err := parsePulsarMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validPulsarWithAuthParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if !testData.isError && meta != nil {
			if meta.AdminURL != testData.adminURL {
				t.Errorf("Expected adminURL %s but got %s\n", testData.adminURL, meta.AdminURL)
			}

			if testData.isPartitionedTopic {
				if !strings.HasSuffix(meta.statsURL, "/partitioned-stats") {
					t.Errorf("Expected statsURL to end with /partitioned-stats but got %s\n", meta.statsURL)
				}
			} else if meta.statsURL != "" {
				if !strings.HasSuffix(meta.statsURL, "/stats") {
					t.Errorf("Expected statsURL to end with /stats but got %s\n", meta.statsURL)
				}
			}

			if meta.Topic != testData.topic {
				t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.Topic)
			}

			if meta.Subscription != testData.subscription {
				t.Errorf("Expected subscription %s but got %s\n", testData.subscription, meta.Subscription)
			}

			var testDataMsgBacklogThreshold int64 = defaultMsgBacklogThreshold
			if val, ok := testData.metadata["msgBacklogThreshold"]; ok {
				testDataMsgBacklogThreshold, err = strconv.ParseInt(val, 10, 64)
				if err != nil {
					t.Errorf("error parsing msgBacklogThreshold: %v", err)
				}
			}
			if meta.MsgBacklogThreshold != testDataMsgBacklogThreshold {
				t.Errorf("Expected msgBacklogThreshold %d but got %d\n", testDataMsgBacklogThreshold, meta.MsgBacklogThreshold)
			}
		}

		authParams := validPulsarWithoutAuthParams
		if k, ok := testData.metadata["tls"]; ok && k == "enable" {
			authParams = validPulsarWithAuthParams
		}

		meta, err = parsePulsarMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if !testData.isError && meta != nil {
			if meta.AdminURL != testData.adminURL {
				t.Errorf("Expected adminURL %s but got %s\n", testData.adminURL, meta.AdminURL)
			}

			if meta.Topic != testData.topic {
				t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.Topic)
			}

			if meta.Subscription != testData.subscription {
				t.Errorf("Expected subscription %s but got %s\n", testData.subscription, meta.Subscription)
			}
		}
	}
}

func compareScope(scopes []string, scopeStr string) bool {
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[scope] = true
	}
	scopeList := strings.Fields(scopeStr)
	for _, scope := range scopeList {
		if !scopeMap[scope] {
			return false
		}
	}
	return true
}

func TestPulsarAuthParams(t *testing.T) {
	for _, testData := range parsePulsarMetadataTestAuthTLSDataset {
		meta, err := parsePulsarMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.triggerMetadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", testData.authParams, err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if meta == nil || meta.pulsarAuth == nil {
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
			if testData.username != "" {
				t.Errorf("Expected EnableBasicAuth to be true when username is %s\n", testData.username)
			}
			if testData.password != "" {
				t.Errorf("Expected EnableBasicAuth to be true when password is %s\n", testData.password)
			}
		}

		if meta.pulsarAuth.Username != testData.username {
			t.Errorf("Expected username to be set to %s but got %s\n", testData.username, meta.pulsarAuth.Username)
		}

		if meta.pulsarAuth.Password != testData.password {
			t.Errorf("Expected password to be set to %s but got %s\n", testData.password, meta.pulsarAuth.Password)
		}
	}
}

func TestPulsarOAuthParams(t *testing.T) {
	for _, testData := range parsePulsarMetadataTestAuthTLSDataset {
		meta, err := parsePulsarMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.triggerMetadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", testData.authParams, err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if meta == nil || meta.pulsarAuth == nil {
			t.Log("meta.pulsarAuth is nil, skipping rest of validation of", testData)
			continue
		}

		if meta.pulsarAuth.EnableOAuth != (testData.clientID != "" || testData.clientSecret != "") {
			if testData.clientID != "" {
				t.Errorf("Expected EnableOAuth to be true when clientID is %s\n", testData.clientID)
			}
			if testData.clientSecret != "" {
				t.Errorf("Expected EnableOAuth to be true when clientSecret is %s\n", testData.clientSecret)
			}
		}

		if meta.pulsarAuth.OauthTokenURI != testData.oauthTokenURI {
			t.Errorf("Expected oauthTokenURI to be set to %s but got %s\n", testData.oauthTokenURI, meta.pulsarAuth.OauthTokenURI)
		}

		if testData.scope != "" && !compareScope(meta.pulsarAuth.Scopes, testData.scope) {
			t.Errorf("Expected scopes %s but got %s\n", testData.scope, meta.pulsarAuth.Scopes)
		}
		if testData.scope == "" && meta.pulsarAuth.Scopes != nil {
			t.Errorf("Expected scopes to be null but got %s\n", meta.pulsarAuth.Scopes)
		}

		if meta.pulsarAuth.ClientID != testData.clientID {
			t.Errorf("Expected clientID to be set to %s but got %s\n", testData.clientID, meta.pulsarAuth.ClientID)
		}

		if meta.pulsarAuth.EnableOAuth && meta.pulsarAuth.ClientSecret == "" {
			t.Errorf("Expected clientSecret not to be empty.\n")
		}

		if testData.clientSecret != "" && strings.Compare(meta.pulsarAuth.ClientSecret, testData.clientSecret) != 0 {
			t.Errorf("Expected clientSecret to be set to %s but got %s\n", testData.clientSecret, meta.pulsarAuth.ClientSecret)
		}

		if reflect.DeepEqual(testData.expectedEndpointParams, meta.pulsarAuth.EndpointParams) {
			t.Errorf("Expected endpointParams %s but got %s\n", testData.expectedEndpointParams, meta.pulsarAuth.EndpointParams)
		}
	}
}

func TestPulsarGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range pulsarMetricIdentifiers {
		meta, err := parsePulsarMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validPulsarWithAuthParams})
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
		mockPulsarScaler, err := NewPulsarScaler(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validPulsarWithoutAuthParams})
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
		mockPulsarScaler, err := NewPulsarScaler(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validPulsarWithAuthParams})
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
		mockPulsarScaler, err := NewPulsarScaler(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validPulsarWithoutAuthParams})
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
