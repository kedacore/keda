package scalers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/go-logr/logr"

	kafka_oauth "github.com/kedacore/keda/v2/pkg/scalers/kafka"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseKafkaMetadataTestData struct {
	metadata                 map[string]string
	isError                  bool
	numBrokers               int
	brokers                  []string
	group                    string
	topic                    string
	partitionLimitation      []int32
	offsetResetPolicy        offsetResetPolicy
	allowIdleConsumers       bool
	excludePersistentLag     bool
	limitToPartitionsWithLag bool
}

type parseKafkaAuthParamsTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

type parseKafkaOAuthbearerAuthParamsTestData = struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

// Testing the case where `tls` and `sasl` are specified in ScaledObject
type parseAuthParamsTestDataSecondAuthMethod struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

type kafkaMetricIdentifier struct {
	metadataTestData *parseKafkaMetadataTestData
	triggerIndex     int
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
	{map[string]string{}, true, 0, nil, "", "", nil, "", false, false, false},
	// failure, no consumer group
	{map[string]string{"bootstrapServers": "foobar:9092"}, true, 1, []string{"foobar:9092"}, "", "", nil, "latest", false, false, false},
	// success, no topic
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group"}, false, 1, []string{"foobar:9092"}, "my-group", "", nil, offsetResetPolicy("latest"), false, false, false},
	// success, ignore partitionLimitation if no topic
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "partitionLimitation": "1,2,3,4,5,6"}, false, 1, []string{"foobar:9092"}, "my-group", "", nil, offsetResetPolicy("latest"), false, false, false},
	// success, no limitation with whitespaced limitation value
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "partitionLimitation": "           "}, false, 1, []string{"foobar:9092"}, "my-group", "", nil, offsetResetPolicy("latest"), false, false, false},
	// success, no limitation
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "partitionLimitation": ""}, false, 1, []string{"foobar:9092"}, "my-group", "", nil, offsetResetPolicy("latest"), false, false, false},
	// failure, version not supported
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "version": "1.2.3.4"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// failure, lagThreshold is negative value
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "-1"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// failure, lagThreshold is 0
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "0"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success, lagThreshold is 1000000
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "1000000", "activationLagThreshold": "0"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// failure, activationLagThreshold is not int
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "10", "activationLagThreshold": "AA"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success, activationLagThreshold is 0
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "lagThreshold": "10", "activationLagThreshold": "0"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success, partitionLimitation as list
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "1,2,3,4"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", []int32{1, 2, 3, 4}, offsetResetPolicy("latest"), false, false, false},
	// success, partitionLimitation as range
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "1-4"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", []int32{1, 2, 3, 4}, offsetResetPolicy("latest"), false, false, false},
	// success, partitionLimitation mixed list + ranges
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "1-4,8,10-12"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", []int32{1, 2, 3, 4, 8, 10, 11, 12}, offsetResetPolicy("latest"), false, false, false},
	// failure, partitionLimitation wrong data type
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "a,b,c,d"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success, more brokers
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success, offsetResetPolicy policy latest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "latest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// failure, offsetResetPolicy policy wrong
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "foo"}, true, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", nil, "", false, false, false},
	// success, offsetResetPolicy policy earliest
	{map[string]string{"bootstrapServers": "foo:9092,bar:9092", "consumerGroup": "my-group", "topic": "my-topic", "offsetResetPolicy": "earliest"}, false, 2, []string{"foo:9092", "bar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("earliest"), false, false, false},
	// failure, allowIdleConsumers malformed
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "notvalid"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success, allowIdleConsumers is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), true, false, false},
	// failure, excludePersistentLag is malformed
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "excludePersistentLag": "notvalid"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// success, excludePersistentLag is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "excludePersistentLag": "true"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, true, false},
	// success, version supported
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), true, false, false},
	// success, limitToPartitionsWithLag is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "limitToPartitionsWithLag": "true"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, true},
	// failure, limitToPartitionsWithLag is malformed
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "limitToPartitionsWithLag": "notvalid"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), false, false, false},
	// failure, allowIdleConsumers and limitToPartitionsWithLag cannot be set to true simultaneously
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "limitToPartitionsWithLag": "true"}, true, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), true, false, true},
	// success, allowIdleConsumers can be set when limitToPartitionsWithLag is false
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "limitToPartitionsWithLag": "false"}, false, 1, []string{"foobar:9092"}, "my-group", "my-topic", nil, offsetResetPolicy("latest"), true, false, false},
	// failure, topic must be specified when limitToPartitionsWithLag is true
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "limitToPartitionsWithLag": "true"}, true, 1, []string{"foobar:9092"}, "my-group", "", nil, offsetResetPolicy("latest"), false, false, true},
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
	// success, SASL GSSAPI/password
	{map[string]string{"sasl": "gssapi", "username": "admin", "password": "admin", "kerberosConfig": "<config>", "realm": "tst.com"}, false, false},
	// success, SASL GSSAPI/keytab
	{map[string]string{"sasl": "gssapi", "username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "tst.com"}, false, false},
	// success, SASL GSSAPI/password + TLS
	{map[string]string{"sasl": "gssapi", "username": "admin", "password": "admin", "kerberosConfig": "<config>", "realm": "tst.com", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, SASL GSSAPI/keytab + TLS
	{map[string]string{"sasl": "gssapi", "username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "tst.com", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, SASL GSSAPI, KerberosServiceName supported
	{map[string]string{"sasl": "gssapi", "username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "tst.com", "kerberosServiceName": "srckafka"}, false, false},
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
	// failure, SASL GSSAPI missing password and keytab
	{map[string]string{"sasl": "gssapi", "username": "admin", "kerberosConfig": "<config>", "realm": "tst.com"}, true, false},
	// failure, SASL GSSAPI provided both password and keytab
	{map[string]string{"sasl": "gssapi", "username": "admin", "password": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "tst.com"}, true, false},
	// failure, SASL GSSAPI/password + TLS missing realm
	{map[string]string{"sasl": "gssapi", "username": "admin", "password": "admin", "kerberosConfig": "<config>", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL GSSAPI/keytab + TLS missing username
	{map[string]string{"sasl": "gssapi", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "tst.com", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// success, SASL GSSAPI/disableFast
	{map[string]string{"sasl": "gssapi", "username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "tst.com", "kerberosDisableFAST": "true"}, false, false},
	// failure, SASL GSSAPI/disableFast incorrect
	{map[string]string{"sasl": "gssapi", "username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "tst.com", "kerberosDisableFAST": "notabool"}, true, false},
	// success, SASL none
	{map[string]string{"sasl": "none"}, false, false},
}
var parseAuthParamsTestDataset = []parseAuthParamsTestDataSecondAuthMethod{
	// success, SASL plaintext
	{map[string]string{"sasl": "plaintext", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// success, SASL scram_sha256
	{map[string]string{"sasl": "scram_sha256", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// success, SASL scram_sha512
	{map[string]string{"sasl": "scram_sha512", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// success, TLS only
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, true},
	// success, TLS CA only
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"ca": "caaa"}, false, true},
	// success, TLS CA only and unsafeSSL
	{map[string]string{"tls": "enable", "unsafeSsl": "true", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"ca": "caaa"}, false, true},
	// success, SASL + TLS
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, SASL + TLS explicitly disabled
	{map[string]string{"sasl": "plaintext", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin"}, false, false},
	// success, SASL OAUTHBEARER + TLS explicitly disabled
	{map[string]string{"sasl": "oauthbearer", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com"}, false, false},
	// success, SASL GSSAPI/password
	{map[string]string{"sasl": "gssapi", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "kerberosConfig": "<config>", "realm": "test.com"}, false, false},
	// success, SASL GSSAPI/password + TLS explicitly disabled
	{map[string]string{"sasl": "gssapi", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "kerberosConfig": "<config>", "realm": "test.com"}, false, false},
	// success, SASL GSSAPI/password + TLS
	{map[string]string{"sasl": "gssapi", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "kerberosConfig": "<config>", "realm": "test.com", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, false},
	// success, SASL GSSAPI/keytab
	{map[string]string{"sasl": "gssapi", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "test.com"}, false, false},
	// success, SASL GSSAPI/keytab + TLS explicitly disabled
	{map[string]string{"sasl": "gssapi", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "test.com"}, false, false},
	// success, SASL GSSAPI/keytab + TLS
	{map[string]string{"sasl": "gssapi", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "keytab": "/path/to/keytab", "kerberosConfig": "<config>", "realm": "test.com", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, false},
	// failure, SASL OAUTHBEARER + TLS explicitly disable +  bad SASL type
	{map[string]string{"sasl": "foo", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com"}, true, false},
	// success, SASL OAUTHBEARER + TLS missing scope
	{map[string]string{"sasl": "oauthbearer", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "oauthTokenEndpointUri": "https://website.com"}, false, false},
	// failure, SASL OAUTHBEARER + TLS missing oauthTokenEndpointUri
	{map[string]string{"sasl": "oauthbearer", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": ""}, true, false},
	// failure, SASL incorrect type
	{map[string]string{"sasl": "foo", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin"}, true, false},
	// failure, SASL missing username
	{map[string]string{"sasl": "plaintext", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"password": "admin"}, true, false},
	// failure, SASL missing password
	{map[string]string{"sasl": "plaintext", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin"}, true, false},
	// failure, TLS missing cert
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"ca": "caaa", "key": "keey"}, true, false},
	// failure, TLS missing key
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"ca": "caaa", "cert": "ceert"}, true, false},
	// failure, TLS invalid
	{map[string]string{"tls": "random", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, incorrect SASL type
	{map[string]string{"sasl": "foo", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, incorrect tls
	{map[string]string{"sasl": "plaintext", "tls": "foo", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing username
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"password": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing password
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing cert
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "ca": "caaa", "key": "keey"}, true, false},
	// failure, SASL + TLS, missing key
	{map[string]string{"sasl": "plaintext", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "plaintext", "username": "admin", "password": "admin", "ca": "caaa", "cert": "ceert"}, true, true},
	// failure, SASL GSSAPI missing keytab and password
	{map[string]string{"sasl": "gssapi", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "realm": "test.com"}, true, false},
	// failure, SASL GSSAPI values in both keytab and password
	{map[string]string{"sasl": "gssapi", "tls": "disable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "password": "admin", "keytab": "/path/to/keytab", "realm": "test.com"}, true, false},
	// failure, SASL GSSAPI + TLS missing realm
	{map[string]string{"sasl": "gssapi", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "keytab": "/path/to/keytab", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, SASL GSSAPI + TLS missing username
	{map[string]string{"sasl": "gssapi", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"realm": "test.com", "keytab": "/path/to/keytab", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, SASL GSSAPI + TLS missing kerberosConfig
	{map[string]string{"sasl": "gssapi", "tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"username": "admin", "realm": "test.com", "keytab": "/path/to/keytab", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// failure, setting SASL values in both places
	{map[string]string{"sasl": "scram_sha512", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "scram_sha512", "username": "admin", "password": "admin"}, true, false},
	// failure, setting TLS values in both places
	{map[string]string{"tls": "enable", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true, true},
	// success, setting SASL plaintext value with extra \n in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "plaintext\n", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL plaintext value with extra space in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "plaintext ", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha256 value with extra \n in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "scram_sha256\n", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha256 value with extra space in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "scram_sha256 ", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha512 value with extra \n in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "scram_sha512\n", "username": "admin", "password": "admin"}, false, true},
	// success, setting SASL scram_sha512 value with extra space in TriggerAuthentication
	{map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{"sasl": "scram_sha512 ", "username": "admin", "password": "admin"}, false, true},
	// success, SASL none
	{map[string]string{"sasl": "none", "bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "allowIdleConsumers": "true", "version": "1.0.0"}, map[string]string{}, false, false},
}

var parseKafkaOAuthbearerAuthParamsTestDataset = []parseKafkaOAuthbearerAuthParamsTestData{
	// success, SASL OAUTHBEARER + TLS
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// success, SASL OAUTHBEARER + tokenProvider + TLS
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "saslTokenProvider": "bearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// success, SASL OAUTHBEARER + TLS multiple scopes
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope1, scope2", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// success, SASL OAUTHBEARER + TLS missing scope
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, false, false},
	// failure, SASL OAUTHBEARER + TLS bad sasl type
	{map[string]string{}, map[string]string{"sasl": "foo", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable"}, true, false},
	// failure, SASL OAUTHBEARER + TLS missing oauthTokenEndpointUri
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "", "tls": "disable"}, true, false},
	// success, SASL OAUTHBEARER + extension
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable", "oauthExtensions": "extension_foo=bar"}, false, false},
	// success, SASL OAUTHBEARER + multiple extensions
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable", "oauthExtensions": "extension_foo=bar,extension_baz=baz"}, false, false},
	// failure, SASL OAUTHBEARER + bad extension
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "scopes": "scope", "oauthTokenEndpointUri": "https://website.com", "tls": "disable", "oauthExtensions": "extension_foo=bar,extension_bazbaz"}, true, false},
	// success, SASL OAUTHBEARER MSK + TLS + Credentials
	{map[string]string{"awsRegion": "eu-west-1"}, map[string]string{"sasl": "oauthbearer", "saslTokenProvider": "aws_msk_iam", "tls": "enable", "awsAccessKeyID": "none", "awsSecretAccessKey": "none"}, false, true},
	// success, SASL OAUTHBEARER MSK + TLS + Role
	{map[string]string{"awsRegion": "eu-west-1"}, map[string]string{"sasl": "oauthbearer", "saslTokenProvider": "aws_msk_iam", "tls": "enable", "awsRegion": "eu-west-1", "awsRoleArn": "none"}, false, true},
	// failure, SASL OAUTHBEARER MSK + no TLS
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "saslTokenProvider": "aws_msk_iam", "tls": "disable"}, true, false},
	// failure, SASL OAUTHBEARER MSK + TLS + no region
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "saslTokenProvider": "aws_msk_iam", "tls": "enable", "awsRegion": ""}, true, true},
	// failure, SASL OAUTHBEARER MSK + TLS + no credentials
	{map[string]string{}, map[string]string{"sasl": "oauthbearer", "saslTokenProvider": "aws_msk_iam", "tls": "enable", "awsRegion": "eu-west-1"}, true, true},
}

var kafkaMetricIdentifiers = []kafkaMetricIdentifier{
	{&parseKafkaMetadataTestDataset[11], 0, "s0-kafka-my-topic"},
	{&parseKafkaMetadataTestDataset[11], 1, "s1-kafka-my-topic"},
	{&parseKafkaMetadataTestDataset[2], 1, "s1-kafka-my-group-topics"},
}

func TestGetBrokers(t *testing.T) {
	for _, testData := range parseKafkaMetadataTestDataset {
		meta, err := parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validWithAuthParams}, logr.Discard())
		getBrokerTestBase(t, meta, testData, err)

		meta, err = parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: validWithoutAuthParams}, logr.Discard())
		getBrokerTestBase(t, meta, testData, err)
	}
}

func getBrokerTestBase(t *testing.T, meta kafkaMetadata, testData parseKafkaMetadataTestData, err error) {
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
	if !reflect.DeepEqual(testData.partitionLimitation, meta.partitionLimitation) {
		t.Errorf("Expected %v but got %v\n", testData.partitionLimitation, meta.partitionLimitation)
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
	expectedLagThreshold, er := parseExpectedLagThreshold(testData.metadata)
	if er != nil {
		t.Errorf("Unable to convert test data lagThreshold %s to string", testData.metadata["lagThreshold"])
	}

	if meta.lagThreshold != expectedLagThreshold && meta.lagThreshold != defaultKafkaLagThreshold {
		t.Errorf("Expected lagThreshold to be either %v or %v got %v ", meta.lagThreshold, defaultKafkaLagThreshold, expectedLagThreshold)
	}
}

func TestKafkaAuthParamsInTriggerAuthentication(t *testing.T) {
	for _, testData := range parseKafkaAuthParamsTestDataset {
		meta, err := parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: validKafkaMetadata, AuthParams: testData.authParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if !testData.isError && meta.enableTLS != testData.enableTLS {
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
		if meta.saslType == KafkaSASLTypeGSSAPI && !testData.isError {
			if testData.authParams["keytab"] != "" {
				err := testFileContents(testData, meta, "keytab")
				if err != nil {
					t.Error(err.Error())
				}
			}
			if !testData.isError {
				err := testFileContents(testData, meta, "kerberosConfig")
				if err != nil {
					t.Error(err.Error())
				}
			}
			if meta.kerberosServiceName != testData.authParams["kerberosServiceName"] {
				t.Errorf("Expected kerberos ServiceName to be set to %v but got %v\n", testData.authParams["kerberosServiceName"], meta.kerberosServiceName)
			}
		}
	}
}

func TestKafkaAuthParamsInScaledObject(t *testing.T) {
	for id, testData := range parseAuthParamsTestDataset {
		meta, err := parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Errorf("Test case: %v. Expected success but got error %v", id, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Test case: %v. Expected error but got success", id)
		}
		if !testData.isError {
			if testData.metadata["tls"] == "true" && !meta.enableTLS {
				t.Errorf("Test case: %v. Expected tls to be set to %v but got %v\n", id, testData.metadata["tls"], meta.enableTLS)
			}
			if meta.enableTLS {
				if meta.ca != testData.authParams["ca"] {
					t.Errorf("Test case: %v. Expected ca to be set to %v but got %v\n", id, testData.authParams["ca"], meta.ca)
				}
				if meta.cert != testData.authParams["cert"] {
					t.Errorf("Test case: %v. Expected cert to be set to %v but got %v\n", id, testData.authParams["cert"], meta.cert)
				}
				if meta.key != testData.authParams["key"] {
					t.Errorf("Test case: %v. Expected key to be set to %v but got %v\n", id, testData.authParams["key"], meta.key)
				}
				if meta.keyPassword != testData.authParams["keyPassword"] {
					t.Errorf("Test case: %v. Expected key to be set to %v but got %v\n", id, testData.authParams["keyPassword"], meta.keyPassword)
				}
				if val, ok := testData.authParams["unsafeSsl"]; ok && err == nil {
					boolVal, err := strconv.ParseBool(val)
					if err != nil && !testData.isError {
						t.Errorf("Expect error but got success in test case %s", meta.key)
					}
					if boolVal != meta.unsafeSsl {
						t.Errorf("Expected unsafeSsl key to be set to %v but got %v\n", boolVal, meta.unsafeSsl)
					}
				}
			}
		}
	}
}

func testFileContents(testData parseKafkaAuthParamsTestData, meta kafkaMetadata, prop string) error {
	if testData.authParams[prop] != "" {
		var path string
		switch prop {
		case "keytab":
			path = meta.keytabPath
		case "kerberosConfig":
			path = meta.kerberosConfigPath
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("expected to find '%v' file at %v", prop, path)
		}
		contents := string(data)
		if contents != testData.authParams[prop] {
			return fmt.Errorf("expected keytab value: '%v' but got '%v'", testData.authParams[prop], contents)
		}
	}
	return nil
}

func TestKafkaOAuthbearerAuthParams(t *testing.T) {
	for _, testData := range parseKafkaOAuthbearerAuthParamsTestDataset {
		for k, v := range validKafkaMetadata {
			testData.metadata[k] = v
		}

		meta, err := parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams}, logr.Discard())

		if err != nil && !testData.isError {
			t.Fatal("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Fatal("Expected error but got success")
		}

		if testData.authParams["saslTokenProvider"] == "" || testData.authParams["saslTokenProvider"] == "bearer" {
			if !testData.isError && meta.tokenProvider != KafkaSASLOAuthTokenProviderBearer {
				t.Errorf("Expected tokenProvider to be set to %v but got %v\n", KafkaSASLOAuthTokenProviderBearer, meta.tokenProvider)
			}

			if testData.authParams["scopes"] == "" {
				if len(meta.scopes) != strings.Count(testData.authParams["scopes"], ",")+1 {
					t.Errorf("Expected scopes to be set to %v but got %v\n", strings.Count(testData.authParams["scopes"], ","), len(meta.scopes))
				}
			}

			if err == nil && testData.authParams["oauthExtensions"] != "" {
				if len(meta.oauthExtensions) != strings.Count(testData.authParams["oauthExtensions"], ",")+1 {
					t.Errorf("Expected number of extensions to be set to %v but got %v\n", strings.Count(testData.authParams["oauthExtensions"], ",")+1, len(meta.oauthExtensions))
				}
			}
		} else if testData.authParams["saslTokenProvider"] == "aws_msk_iam" {
			if !testData.isError && meta.tokenProvider != KafkaSASLOAuthTokenProviderAWSMSKIAM {
				t.Errorf("Expected tokenProvider to be set to %v but got %v\n", KafkaSASLOAuthTokenProviderAWSMSKIAM, meta.tokenProvider)
			}

			if testData.metadata["awsRegion"] != "" && meta.awsRegion != testData.metadata["awsRegion"] {
				t.Errorf("Expected awsRegion to be set to %v but got %v\n", testData.metadata["awsRegion"], meta.awsRegion)
			}

			if testData.authParams["awsAccessKeyID"] != "" {
				if meta.awsAuthorization.AwsAccessKeyID != testData.authParams["awsAccessKeyID"] {
					t.Errorf("Expected awsAccessKeyID to be set to %v but got %v\n", testData.authParams["awsAccessKeyID"], meta.awsAuthorization.AwsAccessKeyID)
				}

				if meta.awsAuthorization.AwsSecretAccessKey != testData.authParams["awsSecretAccessKey"] {
					t.Errorf("Expected awsSecretAccessKey to be set to %v but got %v\n", testData.authParams["awsSecretAccessKey"], meta.awsAuthorization.AwsSecretAccessKey)
				}
			} else if testData.authParams["awsRoleArn"] != "" && meta.awsAuthorization.AwsRoleArn != testData.authParams["awsRoleArn"] {
				t.Errorf("Expected awsRoleArn to be set to %v but got %v\n", testData.authParams["awsRoleArn"], meta.awsAuthorization.AwsRoleArn)
			}
		}
	}
}

func TestKafkaClientsOAuthTokenProvider(t *testing.T) {
	testData := []struct {
		name                  string
		metadata              map[string]string
		authParams            map[string]string
		expectedTokenProvider string
	}{
		{"oauthbearer_bearer", map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "1,2"}, map[string]string{"sasl": "oauthbearer", "username": "admin", "password": "admin", "oauthTokenEndpointUri": "https://website.com"}, "OAuthBearer"},
		{"oauthbearer_aws_msk_iam", map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "1,2", "tls": "enable", "awsRegion": "eu-west-1"}, map[string]string{"sasl": "oauthbearer", "saslTokenProvider": "aws_msk_iam", "awsRegion": "eu-west-1", "awsAccessKeyID": "none", "awsSecretAccessKey": "none"}, "MSK"},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: tt.metadata, AuthParams: tt.authParams}, logr.Discard())
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}

			cfg, err := getKafkaClientConfig(context.TODO(), meta)
			if err != nil {
				t.Error("Expected success but got error", err)
			}

			if !cfg.Net.SASL.Enable {
				t.Error("Expected SASL to be enabled on client")
			}

			tokenProvider, ok := cfg.Net.SASL.TokenProvider.(kafka_oauth.TokenProvider)
			if !ok {
				t.Error("Expected token provider to be set on client")
			}

			if tokenProvider.String() != tt.expectedTokenProvider {
				t.Errorf("Expected token provider to be %v but got %v", tt.expectedTokenProvider, tokenProvider.String())
			}
		})
	}
}

func TestKafkaGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range kafkaMetricIdentifiers {
		meta, err := parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validWithAuthParams, TriggerIndex: testData.triggerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockKafkaScaler := kafkaScaler{"", meta, nil, nil, logr.Discard(), make(map[string]map[int32]int64)}

		metricSpec := mockKafkaScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGetTopicPartitions(t *testing.T) {
	testData := []struct {
		name         string
		metadata     map[string]string
		partitionIds []int32
		exp          map[string][]int32
	}{
		{"success_all_partitions_explicit", map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "1,2"}, []int32{1, 2}, map[string][]int32{"my-topic": {1, 2}}},
		{"success_partial_partitions_explicit", map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": "1,2,3"}, []int32{1, 2, 3, 4, 5, 6}, map[string][]int32{"my-topic": {1, 2, 3}}},
		{"success_all_partitions_implicit", map[string]string{"bootstrapServers": "foobar:9092", "consumerGroup": "my-group", "topic": "my-topic", "partitionLimitation": ""}, []int32{1, 2, 3, 4, 5, 6}, map[string][]int32{"my-topic": {1, 2, 3, 4, 5, 6}}},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := parseKafkaMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: tt.metadata, AuthParams: validWithAuthParams}, logr.Discard())
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			mockKafkaScaler := kafkaScaler{"", meta, nil, &MockClusterAdmin{partitionIds: tt.partitionIds}, logr.Discard(), make(map[string]map[int32]int64)}

			partitions, err := mockKafkaScaler.getTopicPartitions()

			if !reflect.DeepEqual(tt.exp, partitions) {
				t.Errorf("Expected %v but got %v\n", tt.exp, partitions)
			}

			if err != nil {
				t.Error("Expected success but got error", err)
			}
		})
	}
}

var _ sarama.ClusterAdmin = (*MockClusterAdmin)(nil)

type MockClusterAdmin struct {
	partitionIds []int32
}

func (m *MockClusterAdmin) CreateTopic(_ string, _ *sarama.TopicDetail, _ bool) error {
	return nil
}
func (m *MockClusterAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
	return nil, nil
}

func (m *MockClusterAdmin) DescribeTopics(topics []string) (metadata []*sarama.TopicMetadata, err error) {
	metadatas := make([]*sarama.TopicMetadata, len(topics))

	partitionMetadata := make([]*sarama.PartitionMetadata, len(m.partitionIds))
	for i, id := range m.partitionIds {
		partitionMetadata[i] = &sarama.PartitionMetadata{ID: id}
	}

	for i, name := range topics {
		metadatas[i] = &sarama.TopicMetadata{Name: name, Partitions: partitionMetadata}
	}
	return metadatas, nil
}

func (m *MockClusterAdmin) DeleteTopic(_ string) error {
	return nil
}

func (m *MockClusterAdmin) CreatePartitions(_ string, _ int32, _ [][]int32, _ bool) error {
	return nil
}

func (m *MockClusterAdmin) AlterPartitionReassignments(_ string, _ [][]int32) error {
	return nil
}

func (m *MockClusterAdmin) ListPartitionReassignments(_ string, _ []int32) (topicStatus map[string]map[int32]*sarama.PartitionReplicaReassignmentsStatus, err error) {
	return nil, nil
}

func (m *MockClusterAdmin) DeleteRecords(_ string, _ map[int32]int64) error {
	return nil
}

func (m *MockClusterAdmin) DescribeConfig(_ sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	return nil, nil
}

func (m *MockClusterAdmin) AlterConfig(_ sarama.ConfigResourceType, _ string, _ map[string]*string, _ bool) error {
	return nil
}

func (m *MockClusterAdmin) IncrementalAlterConfig(_ sarama.ConfigResourceType, _ string, _ map[string]sarama.IncrementalAlterConfigsEntry, _ bool) error {
	return nil
}

func (m *MockClusterAdmin) CreateACL(_ sarama.Resource, _ sarama.Acl) error {
	return nil
}

func (m *MockClusterAdmin) CreateACLs([]*sarama.ResourceAcls) error {
	return nil
}

func (m *MockClusterAdmin) ListAcls(_ sarama.AclFilter) ([]sarama.ResourceAcls, error) {
	return nil, nil
}

func (m *MockClusterAdmin) DeleteACL(_ sarama.AclFilter, _ bool) ([]sarama.MatchingAcl, error) {
	return nil, nil
}

func (m *MockClusterAdmin) ElectLeaders(sarama.ElectionType, map[string][]int32) (map[string]map[int32]*sarama.PartitionResult, error) {
	return nil, nil
}

func (m *MockClusterAdmin) ListConsumerGroups() (map[string]string, error) {
	return nil, nil
}

func (m *MockClusterAdmin) DescribeConsumerGroups(_ []string) ([]*sarama.GroupDescription, error) {
	return nil, nil
}

func (m *MockClusterAdmin) ListConsumerGroupOffsets(_ string, _ map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	return nil, nil
}

func (m *MockClusterAdmin) DeleteConsumerGroupOffset(_ string, _ string, _ int32) error {
	return nil
}

func (m *MockClusterAdmin) DeleteConsumerGroup(_ string) error {
	return nil
}

func (m *MockClusterAdmin) DescribeCluster() (brokers []*sarama.Broker, controllerID int32, err error) {
	return nil, 0, nil
}

func (m *MockClusterAdmin) DescribeLogDirs(_ []int32) (map[int32][]sarama.DescribeLogDirsResponseDirMetadata, error) {
	return nil, nil
}

func (m *MockClusterAdmin) DescribeUserScramCredentials(_ []string) ([]*sarama.DescribeUserScramCredentialsResult, error) {
	return nil, nil
}

func (m *MockClusterAdmin) DeleteUserScramCredentials(_ []sarama.AlterUserScramCredentialsDelete) ([]*sarama.AlterUserScramCredentialsResult, error) {
	return nil, nil
}

func (m *MockClusterAdmin) UpsertUserScramCredentials(_ []sarama.AlterUserScramCredentialsUpsert) ([]*sarama.AlterUserScramCredentialsResult, error) {
	return nil, nil
}

func (m *MockClusterAdmin) DescribeClientQuotas(_ []sarama.QuotaFilterComponent, _ bool) ([]sarama.DescribeClientQuotasEntry, error) {
	return nil, nil
}

func (m *MockClusterAdmin) AlterClientQuotas(_ []sarama.QuotaEntityComponent, _ sarama.ClientQuotasOp, _ bool) error {
	return nil
}

func (m *MockClusterAdmin) Controller() (*sarama.Broker, error) {
	return nil, nil
}

func (m *MockClusterAdmin) RemoveMemberFromConsumerGroup(_ string, _ []string) (*sarama.LeaveGroupResponse, error) {
	return nil, nil
}

func (m *MockClusterAdmin) Close() error {
	return nil
}

func (m *MockClusterAdmin) Coordinator(group string) (*sarama.Broker, error) {
	return nil, nil
}
