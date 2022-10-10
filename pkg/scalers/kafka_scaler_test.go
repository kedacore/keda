package scalers

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
)

type parseKafkaMetadataTestData struct {
	metadata           map[string]string
	isError            bool
	numBrokers         int
	brokers            []string
	group              string
	topic              string
	offsetResetPolicy  offsetResetPolicy
	allowIdleConsumers bool
}

type parseKafkaAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

type kafkaMetricIdentifier struct {
	metadataTestData *parseKafkaMetadataTestData
	scalerIndex      int
	name             string
}

// A complete valid metadata example for reference
var validKafkaMetadata = map[string]string{
	"bootstrapServers":   "broker1:9092,broker2:9092",
	"consumerGroup":      "my-group",
	"topic":              "my-topic",
	"allowIdleConsumers": "false",
}

// A complete valid authParams example for sasl, with username and passwd
var validWithAuthParams = map[string]string{
	"sasl":     "plaintext",
	"username": "admin",
	"password": "admin",
}

// A complete valid authParams example for sasl, without username and passwd
var validWithoutAuthParams = map[string]string{}

var parseKafkaMetadataTestDataset = []parseKafkaMetadataTestData{
	// failure, no bootstrapServers
	{map[string]string{}, true, 0, nil, "", "", "", false},
	// failure, no consumer group
	{map[string]string{"bootstrapServers": "foobar:9092"}, true, 1, []string{"foobar:9092"}, "", "", "latest", false},
	// success, no topic
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group"}, false, 1, []string{"foobar:9092"}, "my-group", "", offsetResetPolicy("latest"), false},
	// failure, version not supported
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "version": "1.2.3.4"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// failure, lagThreshold is negative value
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "-1"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// failure, lagThreshold is 0
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "0"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// failure, activationLagThreshold is not int
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "10", "activationLagThreshold": "AA"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// success
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// success, more brokers
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// success, offsetResetPolicy policy latest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "latest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// failure, offsetResetPolicy policy wrong
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "foo"}, true, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", "", false},
	// success, offsetResetPolicy policy earliest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "earliest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", offsetResetPolicy("earliest"), false},
	// failure, allowIdleConsumers malformed
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "notvalid"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), false},
	// success, allowIdleConsumers is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), true},
	// success, version supported
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest"), true},
}

var parseKafkaAuthParamsTestDataset = []parseKafkaAuthParamsTestData{
	// success, SASL only
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin"}, false, false},
	// success, SASL only
	{map[string]string{"sasl": "scram_sha256", "username": "admin", "password": "admin"}, false, false},
	// success, SASL only
	{map[string]string{"sasl": "scram_sha512", "username": "admin", "password": "admin"}, false, false},
	// success, TLS only
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, true},
	// success, TLS CA only
	{map[string]string{"tls": "enable", "ca": "caaa"}, false, true},
	// success, SASL + TLS
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, SASL + TLS explicitly disabled
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "disable"}, false, false},
	// success, SASL OAUTHBEARER + TLS
	{map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// failure, SASL OAUTHBEARER + TLS bad sasl type
	{map[string]string{"sasl": "foo", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, true, false},
	// success, SASL OAUTHBEARER + TLS missing scope
	{map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// failure, SASL OAUTHBEARER + TLS missing oauthTokenEndpointUri
	{map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "", "tls": "disable"}, true, false},
	// failure, SASL incorrect type
	{map[string]string{"sasl": "foo", "username": "admin", "password": "admin"}, true, false},
	// failure, SASL missing username
	{map[string]string{"sasl": "plaintext", "password": "admin"}, true, false},
	// failure, SASL missing password
	{map[string]string{"sasl": "plaintext", "username": "admin"}, true, false},
	// failure, TLS missing cert
	{map[string]string{"tls": "enable", "ca": "caaa", "key": "keey"}, true, false},
	// failure, TLS missing key
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert"}, true, false},
	// failure, TLS invalid
	{map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, incorrect sasl
	{map[string]string{"sasl": "foo", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, incorrect tls
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "foo", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing username
	{map[string]string{"sasl": "plaintext", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing password
	{map[string]string{"sasl": "plaintext", "username": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing cert
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing key
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert"}, true, false},
}

var parseKafkaOAuthbreakerAuthParamsTestDataset = []parseKafkaAuthParamsTestData{
	// success, SASL OAUTHBEARER + TLS
	{map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// success, SASL OAUTHBEARER + TLS multiple scopes
	{map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope1, scope2", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// success, SASL OAUTHBEARER + TLS missing scope
	{map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// failure, SASL OAUTHBEARER + TLS bad sasl type
	{map[string]string{"sasl": "foo", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, true, false},
	// failure, SASL OAUTHBEARER + TLS missing oauthTokenEndpointUri
	{map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "", "tls": "disable"}, true, false},
}

var kafkaMetricIdentifiers = []kafkaMetricIdentifier{
	{&parseKafkaMetadataTestDataset[7], 0, "s0-kafka-my-topic"},
	{&parseKafkaMetadataTestDataset[7], 1, "s1-kafka-my-topic"},
	{&parseKafkaMetadataTestDataset[2], 1, "s1-kafka-my-group-topics"},
}

func TestGetBrokers(t *testing.T) {
	for _, testData := range parseKafkaMetadataTestDataset {
		meta, err := parseKafkaMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validWithAuthParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if len(meta.bootstrapServers) != testData.numBrokers {
			t.Errorf("Expected %d bootstrap servers but got %d\n", testData.numBrokers, len(meta.bootstrapServers))
		}
		if !reflect.DeepEqual(testData.brokers, meta.bootstrapServers) {
			t.Errorf("Expected %v but got %v\n", testData.brokers, meta.bootstrapServers)
		}
		if meta.group != testData.group {
			t.Errorf("Expected group %s but got %s\n", testData.group, meta.group)
		}
		if meta.topic != testData.topic {
			t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.topic)
		}
		if err == nil && meta.offsetResetPolicy != testData.offsetResetPolicy {
			t.Errorf("Expected offsetResetPolicy %s but got %s\n", testData.offsetResetPolicy, meta.offsetResetPolicy)
		}

		meta, err = parseKafkaMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validWithoutAuthParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if len(meta.bootstrapServers) != testData.numBrokers {
			t.Errorf("Expected %d bootstrap servers but got %d\n", testData.numBrokers, len(meta.bootstrapServers))
		}
		if !reflect.DeepEqual(testData.brokers, meta.bootstrapServers) {
			t.Errorf("Expected %v but got %v\n", testData.brokers, meta.bootstrapServers)
		}
		if meta.group != testData.group {
			t.Errorf("Expected group %s but got %s\n", testData.group, meta.group)
		}
		if meta.topic != testData.topic {
			t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.topic)
		}
		if err == nil && meta.offsetResetPolicy != testData.offsetResetPolicy {
			t.Errorf("Expected offsetResetPolicy %s but got %s\n", testData.offsetResetPolicy, meta.offsetResetPolicy)
		}
		if err == nil && meta.allowIdleConsumers != testData.allowIdleConsumers {
			t.Errorf("Expected allowIdleConsumers %t but got %t\n", testData.allowIdleConsumers, meta.allowIdleConsumers)
		}
	}
}

func TestKafkaAuthParams(t *testing.T) {
	for _, testData := range parseKafkaAuthParamsTestDataset {
		meta, err := parseKafkaMetadata(&ScalerConfig{TriggerMetadata: validKafkaMetadata, AuthParams: testData.authParams}, logr.Discard())

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

func TestKafkaOAuthbreakerAuthParams(t *testing.T) {
	for _, testData := range parseKafkaOAuthbreakerAuthParamsTestDataset {
		meta, err := parseKafkaMetadata(&ScalerConfig{TriggerMetadata: validKafkaMetadata, AuthParams: testData.authParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if testData.authParams["scopes"] == "" {
			if len(meta.scopes) != strings.Count(testData.authParams["scopes"], ",")+1 {
				t.Errorf("Expected scopes to be set to %v but got %v\n", strings.Count(testData.authParams["scopes"], ","), len(meta.scopes))
			}
		}
	}
}

func TestKafkaGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range kafkaMetricIdentifiers {
		meta, err := parseKafkaMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validWithAuthParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockKafkaScaler := kafkaScaler{"", meta, nil, nil, logr.Discard()}

		metricSpec := mockKafkaScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
