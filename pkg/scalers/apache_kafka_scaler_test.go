package scalers

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
)

type parseApacheKafkaMetadataTestData struct {
	metadata                 map[string]string
	isError                  bool
	numBrokers               int
	brokers                  []string
	group                    string
	topic                    []string
	partitionLimitation      []int32
	offsetResetPolicy        offsetResetPolicy
	allowIdleConsumers       bool
	excludePersistentLag     bool
	limitToPartitionsWithLag bool
}

type parseApacheKafkaAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

// Testing the case where `tls` and `sasl` are specified in ScaledObject
type parseApacheKafkaAuthParamsTestDataSecondAuthMethod struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

type apacheKafkaMetricIdentifier struct {
	metadataTestData *parseApacheKafkaMetadataTestData
	scalerIndex      int
	name             string
}

// A complete valid metadata example for reference
var validApacheKafkaMetadata = map[string]string{
	"bootstrapServers":   "broker1:9092,broker2:9092",
	"consumerGroup":      "my-group",
	"topic":              "my-topics",
	"allowIdleConsumers": "false",
}

// A complete valid authParams example for sasl, with username and passwd
var validApacheKafkaWithAuthParams = map[string]string{
	"sasl":     "plaintext",
	"username": "admin",
	"password": "admin",
}

// A complete valid authParams example for sasl, without username and passwd
var validApacheKafkaWithoutAuthParams = map[string]string{}

var parseApacheKafkaMetadataTestDataset = []parseApacheKafkaMetadataTestData{
	// failure, no consumer group
	{map[string]string{"bootstrapServers": "foobar:9092"}, true, 1, []string{"foobar:9092"}, "", nil, nil, "latest", false, false, false},
	// success, no topics
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group"}, false, 1, []string{"foobar:9092"}, "my-group", []string{}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, ignore partitionLimitation if no topics
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "partitionLimitation": "1,2,3,4,5,6"}, false, 1, []string{"foobar:9092"}, "my-group", []string{}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, no limitation with whitespaced limitation value
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "partitionLimitation": "           "}, false, 1, []string{"foobar:9092"}, "my-group", []string{}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, no limitation
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "partitionLimitation": ""}, false, 1, []string{"foobar:9092"}, "my-group", []string{}, nil, offsetResetPolicy("latest"), false, false, false},
	// failure, lagThreshold is negative value
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "lagThreshold": "-1"}, true, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// failure, lagThreshold is 0
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "lagThreshold": "0"}, true, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, activationLagThreshold is 0
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "lagThreshold": "10", "activationLagThreshold": "0"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// success
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, partitionLimitation as list
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "partitionLimitation": "1,2,3,4"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, []int32{1, 2, 3, 4}, offsetResetPolicy("latest"), false, false, false},
	// success, partitionLimitation as range
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "partitionLimitation": "1-4"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, []int32{1, 2, 3, 4}, offsetResetPolicy("latest"), false, false, false},
	// success, partitionLimitation mixed list + ranges
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "partitionLimitation": "1-4,8,10-12"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, []int32{1, 2, 3, 4, 8, 10, 11, 12}, offsetResetPolicy("latest"), false, false, false},
	// failure, partitionLimitation wrong data type
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "partitionLimitation": "a,b,c,d"}, true, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, more brokers
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topics"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, offsetResetPolicy policy latest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topics", "offsetResetPolicy": "latest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// failure, offsetResetPolicy policy wrong
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topics", "offsetResetPolicy": "foo"}, true, 2, []string{"foo:9092", "bar:9092"}, "my-group", []string{"my-topics"}, nil, "", false, false, false},
	// success, offsetResetPolicy policy earliest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topics", "offsetResetPolicy": "earliest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("earliest"), false, false, false},
	// failure, allowIdleConsumers malformed
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "notvalid"}, true, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, allowIdleConsumers is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), true, false, false},
	// failure, excludePersistentLag is malformed
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "excludePersistentLag": "notvalid"}, true, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// success, excludePersistentLag is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "excludePersistentLag": "true"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, true, false},
	// success, version supported
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), true, false, false},
	// success, limitToPartitionsWithLag is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "limitToPartitionsWithLag": "true"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, true},
	// failure, limitToPartitionsWithLag is malformed
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "limitToPartitionsWithLag": "notvalid"}, true, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), false, false, false},
	// failure, allowIdleConsumers and limitToPartitionsWithLag cannot be set to true simultaneously
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true", "limitToPartitionsWithLag": "true"}, true, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), true, false, true},
	// success, allowIdleConsumers can be set when limitToPartitionsWithLag is false
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true", "limitToPartitionsWithLag": "false"}, false, 1, []string{"foobar:9092"}, "my-group", []string{"my-topics"}, nil, offsetResetPolicy("latest"), true, false, false},
	// failure, topic must be specified when limitToPartitionsWithLag is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "limitToPartitionsWithLag": "true"}, true, 1, []string{"foobar:9092"}, "my-group", []string{}, nil, offsetResetPolicy("latest"), false, false, true},
}

var parseApacheKafkaAuthParamsTestDataset = []parseApacheKafkaAuthParamsTestData{
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
	{map[string]string{"sasl": "foo", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, SASL + TLS, incorrect tls
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "foo", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing username
	{map[string]string{"sasl": "plaintext", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, SASL + TLS, missing password
	{map[string]string{"sasl": "plaintext", "username": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, SASL + TLS, missing cert
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "key": "keey"}, true, true},
	// failure, SASL + TLS, missing key
	{map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "tls": "enable", "ca": "caaa", "cert": "ceert"}, true, true},
}
var parseApacheKafkaAuthParamsTestDataset2 = []parseApacheKafkaAuthParamsTestDataSecondAuthMethod{
	// success, SASL plaintext
	{map[string]string{"sasl": "plaintext", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// success, SASL scram_sha256
	{map[string]string{"sasl": "scram_sha256", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// success, SASL scram_sha512
	{map[string]string{"sasl": "scram_sha512", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// success, TLS only
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, true},
	// success, TLS CA only
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"ca": "caaa"}, false, true},
	// success, SASL + TLS
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, SASL + TLS explicitly disabled
	{map[string]string{"sasl": "plaintext", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// failure, SASL incorrect type
	{map[string]string{"sasl": "foo", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin"}, true, false},
	// failure, SASL missing username
	{map[string]string{"sasl": "plaintext", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"password": "admin"}, true, false},
	// failure, SASL missing password
	{map[string]string{"sasl": "plaintext", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin"}, true, false},
	// failure, TLS missing cert
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"ca": "caaa", "key": "keey"}, true, false},
	// failure, TLS missing key
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"ca": "caaa", "cert": "ceert"}, true, false},
	// failure, TLS invalid
	{map[string]string{"tls": "random", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, incorrect SASL type
	{map[string]string{"sasl": "foo", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, incorrect tls
	{map[string]string{"sasl": "plaintext", "tls": "foo", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing username
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, SASL + TLS, missing password
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, SASL + TLS, missing cert
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "key": "keey"}, true, true},
	// failure, SASL + TLS, missing key
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert"}, true, true},

	// failure, setting SASL values in both places
	{map[string]string{"sasl": "scram_sha512", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "scram_sha512", "username": "admin", "password": "admin"}, true, false},
	// failure, setting TLS values in both places
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// success, setting SASL plaintext value with extra \n in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "plaintext\n", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL plaintext value with extra space in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "plaintext ", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha256 value with extra \n in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "scram_sha256\n", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha256 value with extra space in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "scram_sha256 ", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha512 value with extra \n in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "scram_sha512\n", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha512 value with extra space in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"sasl": "scram_sha512 ", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL aws_msk_iam with tls enabled and passing credentials
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true", "awsRegion": "us-east-1"}, map[string]string{"tls": "enable", "sasl": "aws_msk_iam", "awsAccessKeyID": "none", "awsSecretAccessKey": "none"}, false, true},
	// failure, setting SASL aws_msk_iam with tls enabled and missing awsRegion
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true"}, map[string]string{"tls": "enable", "sasl": "aws_msk_iam", "awsAccessKeyID": "none", "awsSecretAccessKey": "none"}, true, true},
	// failure, setting SASL aws_msk_iam with tls disabled
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topics", "allowIdleConsumers": "true", "awsRegion": "us-east-1"}, map[string]string{"sasl": "aws_msk_iam", "awsAccessKeyID": "none", "awsSecretAccessKey": "none"}, true, false},
}

var apacheKafkaMetricIdentifiers = []apacheKafkaMetricIdentifier{
	{&parseApacheKafkaMetadataTestDataset[10], 0, "s0-kafka-my-topics"},
	{&parseApacheKafkaMetadataTestDataset[10], 1, "s1-kafka-my-topics"},
	{&parseApacheKafkaMetadataTestDataset[2], 1, "s1-kafka-my-group-topics"},
}

func TestApacheKafkaGetBrokers(t *testing.T) {
	for _, testData := range parseApacheKafkaMetadataTestDataset {
		meta, err := parseApacheKafkaMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validApacheKafkaWithAuthParams}, logr.Discard())

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
			t.Errorf("Expected %#v but got %#v\n", testData.brokers, meta.bootstrapServers)
		}
		if meta.group != testData.group {
			t.Errorf("Expected group %s but got %s\n", testData.group, meta.group)
		}
		if !reflect.DeepEqual(testData.topic, meta.topic) {
			t.Errorf("Expected topics %#v but got %#v\n", testData.topic, meta.topic)
		}
		if !reflect.DeepEqual(testData.partitionLimitation, meta.partitionLimitation) {
			t.Errorf("Expected %#v but got %#v\n", testData.partitionLimitation, meta.partitionLimitation)
		}
		if err == nil && meta.offsetResetPolicy != testData.offsetResetPolicy {
			t.Errorf("Expected offsetResetPolicy %s but got %s\n", testData.offsetResetPolicy, meta.offsetResetPolicy)
		}

		meta, err = parseApacheKafkaMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validApacheKafkaWithoutAuthParams}, logr.Discard())

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
			t.Errorf("Expected %#v but got %#v\n", testData.brokers, meta.bootstrapServers)
		}
		if meta.group != testData.group {
			t.Errorf("Expected group %s but got %s\n", testData.group, meta.group)
		}
		if !reflect.DeepEqual(testData.topic, meta.topic) {
			t.Errorf("Expected topics %#v but got %#v\n", testData.topic, meta.topic)
		}
		if !reflect.DeepEqual(testData.partitionLimitation, meta.partitionLimitation) {
			t.Errorf("Expected %#v but got %#v\n", testData.partitionLimitation, meta.partitionLimitation)
		}
		if err == nil && meta.offsetResetPolicy != testData.offsetResetPolicy {
			t.Errorf("Expected offsetResetPolicy %s but got %s\n", testData.offsetResetPolicy, meta.offsetResetPolicy)
		}
		if err == nil && meta.allowIdleConsumers != testData.allowIdleConsumers {
			t.Errorf("Expected allowIdleConsumers %t but got %t\n", testData.allowIdleConsumers, meta.allowIdleConsumers)
		}
		if err == nil && meta.excludePersistentLag != testData.excludePersistentLag {
			t.Errorf("Expected excludePersistentLag %t but got %t\n", testData.excludePersistentLag, meta.excludePersistentLag)
		}
		if err == nil && meta.limitToPartitionsWithLag != testData.limitToPartitionsWithLag {
			t.Errorf("Expected limitToPartitionsWithLag %t but got %t\n", testData.limitToPartitionsWithLag, meta.limitToPartitionsWithLag)
		}
	}
}

func TestApacheKafkaAuthParams(t *testing.T) {
	// Testing tls and sasl value in TriggerAuthentication
	for _, testData := range parseApacheKafkaAuthParamsTestDataset {
		meta, err := parseApacheKafkaMetadata(&ScalerConfig{TriggerMetadata: validApacheKafkaMetadata, AuthParams: testData.authParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		// we can ignore what tls is set if there is error
		if err == nil && meta.enableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %#v but got %#v\n", testData.enableTLS, meta.enableTLS)
		}
		if err == nil && meta.enableTLS {
			if meta.ca != testData.authParams["ca"] {
				t.Errorf("Expected ca to be set to %#v but got %#v\n", testData.authParams["ca"], meta.ca)
			}
			if meta.cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %#v but got %#v\n", testData.authParams["cert"], meta.cert)
			}
			if meta.key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %#v but got %#v\n", testData.authParams["key"], meta.key)
			}
			if meta.keyPassword != testData.authParams["keyPassword"] {
				t.Errorf("Expected key to be set to %#v but got %#v\n", testData.authParams["keyPassword"], meta.key)
			}
		}
	}

	// Testing tls and sasl value in scaledObject
	for id, testData := range parseApacheKafkaAuthParamsTestDataset2 {
		meta, err := parseApacheKafkaMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Errorf("Test case: %#v. Expected success but got error %#v", id, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Test case: %#v. Expected error but got success", id)
		}
		if !testData.isError {
			if testData.metadata["tls"] == stringTrue && !meta.enableTLS {
				t.Errorf("Test case: %#v. Expected tls to be set to %#v but got %#v\n", id, testData.metadata["tls"], meta.enableTLS)
			}
			if meta.enableTLS {
				if meta.ca != testData.authParams["ca"] {
					t.Errorf("Test case: %#v. Expected ca to be set to %#v but got %#v\n", id, testData.authParams["ca"], meta.ca)
				}
				if meta.cert != testData.authParams["cert"] {
					t.Errorf("Test case: %#v. Expected cert to be set to %#v but got %#v\n", id, testData.authParams["cert"], meta.cert)
				}
				if meta.key != testData.authParams["key"] {
					t.Errorf("Test case: %#v. Expected key to be set to %#v but got %#v\n", id, testData.authParams["key"], meta.key)
				}
				if meta.keyPassword != testData.authParams["keyPassword"] {
					t.Errorf("Test case: %#v. Expected key to be set to %#v but got %#v\n", id, testData.authParams["keyPassword"], meta.keyPassword)
				}
			}
		}
	}
}

func TestApacheKafkaGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range apacheKafkaMetricIdentifiers {
		meta, err := parseApacheKafkaMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validApacheKafkaWithAuthParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockKafkaScaler := apacheKafkaScaler{"", meta, nil, logr.Discard(), make(map[string]map[int]int64)}

		metricSpec := mockKafkaScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			str := fmt.Sprintf("Wrong External metric source name: %s, expected: %s for %#v\n", metricName, testData.name, testData)
			t.Error("Wrong External metric source name:", metricName, str)
		}
	}
}
