package scalers

import (
	"reflect"
	"testing"
)

type parseKafkaMetadataTestData struct {
	metadata          map[string]string
	isError           bool
	numBrokers        int
	brokers           []string
	group             string
	topic             string
	offsetResetPolicy offsetResetPolicy
}

type parseKafkaAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableSASL bool
	enableTLS  bool
}

type kafkaMetricIdentifier struct {
	metadataTestData *parseKafkaMetadataTestData
	name             string
}

// A complete valid metadata example for reference
var validKafkaMetadata = map[string]string{
	"bootstrapServers": "broker1:9092,broker2:9092",
	"consumerGroup":    "my-group",
	"topic":            "my-topic",
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
	{map[string]string{}, true, 0, nil, "", "", ""},
	// failure, no consumer group
	{map[string]string{"bootstrapServers": "foobar:9092"}, true, 1, []string{"foobar:9092"}, "", "", "latest"},
	// failure, no topic
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group"}, true, 1, []string{"foobar:9092"}, "my-group", "", offsetResetPolicy("latest")},
	// success
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest")},
	// success, more brokers
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest")},
	// success, offsetResetPolicy policy latest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "latest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", offsetResetPolicy("latest")},
	// failure, offsetResetPolicy policy wrong
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "foo"}, true, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", ""},
	// success, offsetResetPolicy policy earliest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "earliest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", offsetResetPolicy("earliest")},
}

var parseKafkaAuthParamsTestDataset = []parseKafkaAuthParamsTestData{
	// success, SASL only
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin"}, false, true, false},
	// success, SASL only
	{map[string]string{"sasl": "scram_sha256", "username": "admin", "password": "admin"}, false, true, false},
	// success, SASL only
	{map[string]string{"sasl": "scram_sha512", "username": "admin", "password": "admin"}, false, true, false},
	// success, TLS only
	{map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, false, true},
	// success, SASL + TLS
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true, true},
	// failure, SASL incorrect type
	{map[string]string{"sasl": "foo", "username": "admin", "password": "admin"}, true, false, false},
	// failure, SASL missing username
	{map[string]string{"sasl": "plaintext", "password": "admin"}, true, false, false},
	// failure, SASL missing password
	{map[string]string{"sasl": "plaintext", "username": "admin"}, true, false, false},
	// failure, TLS incorrect
	{map[string]string{"tls": "yes", "cert": "ceert", "key": "keey"}, true, false, false},
	// failure, TLS missing ca
	{map[string]string{"tls": "yes", "ca": "caaa", "key": "keey"}, true, false, false},
	// failure, TLS missing cert
	{map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false, false},
	// failure, TLS missing key
	{map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert"}, true, false, false},
	// failure, SASL + TLS, incorrect sasl
	{map[string]string{"sasl": "foo", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false, false},
	// failure, SASL + TLS, incorrect tls
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "foo", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true, false},
	// failure, SASL + TLS, missing username
	{map[string]string{"sasl": "plaintext", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false, false},
	// failure, SASL + TLS, missing password
	{map[string]string{"sasl": "plaintext", "username": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false, false},
	// failure, SASL + TLS, missing ca
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "cert": "ceert", "key": "keey"}, true, true, false},
	// failure, SASL + TLS, missing cert
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "key": "keey"}, true, true, false},
	// failure, SASL + TLS, missing key
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert"}, true, true, false},
}

var kafkaMetricIdentifiers = []kafkaMetricIdentifier{
	{&parseKafkaMetadataTestDataset[4], "kafka-my-topic-my-group"},
}

func TestGetBrokers(t *testing.T) {
	for _, testData := range parseKafkaMetadataTestDataset {
		meta, err := parseKafkaMetadata(nil, testData.metadata, validWithAuthParams)

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

		meta, err = parseKafkaMetadata(nil, testData.metadata, validWithoutAuthParams)

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
	}
}

func TestKafkaAuthParams(t *testing.T) {
	for _, testData := range parseKafkaAuthParamsTestDataset {
		meta, err := parseKafkaMetadata(nil, validKafkaMetadata, testData.authParams)

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta.enableSASL != testData.enableSASL {
			t.Errorf("Expected enableSASL to be set to %v but got %v\n", testData.enableSASL, meta.enableSASL)
		}
		if meta.enableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, meta.enableTLS)
		}
	}
}
func TestKafkaGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range kafkaMetricIdentifiers {
		meta, err := parseKafkaMetadata(nil, testData.metadataTestData.metadata, validWithAuthParams)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockKafkaScaler := kafkaScaler{meta, nil, nil}

		metricSpec := mockKafkaScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
