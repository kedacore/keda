/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This scaler is based on sarama library.
// It lacks support for AWS MSK. For AWS MSK please see: apache-kafka scaler.

package scalers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/kafka"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type kafkaScaler struct {
	metricType      v2.MetricTargetType
	metadata        kafkaMetadata
	client          sarama.Client
	admin           sarama.ClusterAdmin
	logger          logr.Logger
	previousOffsets map[string]map[int32]int64
}

const (
	stringEnable     = "enable"
	stringDisable    = "disable"
	defaultUnsafeSsl = false
)

type kafkaMetadata struct {
	bootstrapServers       []string
	group                  string
	topic                  string
	partitionLimitation    []int32
	lagThreshold           int64
	activationLagThreshold int64
	offsetResetPolicy      offsetResetPolicy
	allowIdleConsumers     bool
	excludePersistentLag   bool
	version                sarama.KafkaVersion

	// If an invalid offset is found, whether to scale to 1 (false - the default) so consumption can
	// occur or scale to 0 (true). See discussion in https://github.com/kedacore/keda/issues/2612
	scaleToZeroOnInvalidOffset         bool
	limitToPartitionsWithLag           bool
	ensureEvenDistributionOfPartitions bool

	// SASL
	saslType kafkaSaslType
	username string
	password string

	// GSSAPI
	keytabPath          string
	realm               string
	kerberosConfigPath  string
	kerberosServiceName string
	kerberosDisableFAST bool

	// OAUTHBEARER
	tokenProvider         kafkaSaslOAuthTokenProvider
	scopes                []string
	oauthTokenEndpointURI string
	oauthExtensions       map[string]string

	// MSK
	awsRegion        string
	awsAuthorization awsutils.AuthorizationMetadata

	// TLS
	enableTLS   bool
	cert        string
	key         string
	keyPassword string
	ca          string
	unsafeSsl   bool

	triggerIndex int
}

type offsetResetPolicy string

const (
	latest   offsetResetPolicy = "latest"
	earliest offsetResetPolicy = "earliest"
)

type kafkaSaslType string

// supported SASL types
const (
	KafkaSASLTypeNone        kafkaSaslType = "none"
	KafkaSASLTypePlaintext   kafkaSaslType = "plaintext"
	KafkaSASLTypeSCRAMSHA256 kafkaSaslType = "scram_sha256"
	KafkaSASLTypeSCRAMSHA512 kafkaSaslType = "scram_sha512"
	KafkaSASLTypeOAuthbearer kafkaSaslType = "oauthbearer"
	KafkaSASLTypeGSSAPI      kafkaSaslType = "gssapi"
)

type kafkaSaslOAuthTokenProvider string

// supported SASL OAuth token provider types
const (
	KafkaSASLOAuthTokenProviderBearer    kafkaSaslOAuthTokenProvider = "bearer"
	KafkaSASLOAuthTokenProviderAWSMSKIAM kafkaSaslOAuthTokenProvider = "aws_msk_iam"
)

const (
	lagThresholdMetricName             = "lagThreshold"
	activationLagThresholdMetricName   = "activationLagThreshold"
	kafkaMetricType                    = "External"
	defaultKafkaLagThreshold           = 10
	defaultKafkaActivationLagThreshold = 0
	defaultOffsetResetPolicy           = latest
	invalidOffset                      = -1
)

// NewKafkaScaler creates a new kafkaScaler
func NewKafkaScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "kafka_scaler")

	kafkaMetadata, err := parseKafkaMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing kafka metadata: %w", err)
	}

	client, admin, err := getKafkaClients(ctx, kafkaMetadata)
	if err != nil {
		return nil, err
	}

	previousOffsets := make(map[string]map[int32]int64)

	return &kafkaScaler{
		client:          client,
		admin:           admin,
		metricType:      metricType,
		metadata:        kafkaMetadata,
		logger:          logger,
		previousOffsets: previousOffsets,
	}, nil
}

func parseKafkaAuthParams(config *scalersconfig.ScalerConfig, meta *kafkaMetadata) error {
	meta.enableTLS = false
	enableTLS := false
	if val, ok := config.TriggerMetadata["tls"]; ok {
		switch val {
		case stringEnable:
			enableTLS = true
		case stringDisable:
			enableTLS = false
		default:
			return fmt.Errorf("error incorrect TLS value given, got %s", val)
		}
	}

	if val, ok := config.AuthParams["tls"]; ok {
		val = strings.TrimSpace(val)
		if enableTLS {
			return errors.New("unable to set `tls` in both ScaledObject and TriggerAuthentication together")
		}
		switch val {
		case stringEnable:
			enableTLS = true
		case stringDisable:
			enableTLS = false
		default:
			return fmt.Errorf("error incorrect TLS value given, got %s", val)
		}
	}

	if enableTLS {
		if err := parseTLS(config, meta); err != nil {
			return err
		}
	}

	meta.saslType = KafkaSASLTypeNone
	var saslAuthType string
	switch {
	case config.TriggerMetadata["sasl"] != "":
		saslAuthType = config.TriggerMetadata["sasl"]
	default:
		saslAuthType = ""
	}
	if val, ok := config.AuthParams["sasl"]; ok {
		if saslAuthType != "" {
			return errors.New("unable to set `sasl` in both ScaledObject and TriggerAuthentication together")
		}
		saslAuthType = val
	}

	saslAuthType = strings.TrimSpace(saslAuthType)
	mode := kafkaSaslType(saslAuthType)
	if saslAuthType != "" && mode != KafkaSASLTypeNone {
		switch mode {
		case KafkaSASLTypePlaintext, KafkaSASLTypeSCRAMSHA256, KafkaSASLTypeSCRAMSHA512:
			err := parseSaslParams(config, meta, mode)
			if err != nil {
				return err
			}
		case KafkaSASLTypeOAuthbearer:
			err := parseSaslOAuthParams(config, meta, mode)
			if err != nil {
				return err
			}
		case KafkaSASLTypeGSSAPI:
			err := parseKerberosParams(config, meta, mode)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("err SASL mode %s given", mode)
		}
	}

	return nil
}

func parseSaslOAuthParams(config *scalersconfig.ScalerConfig, meta *kafkaMetadata, mode kafkaSaslType) error {
	var tokenProviderTypeValue string
	if val, ok := config.TriggerMetadata["saslTokenProvider"]; ok {
		tokenProviderTypeValue = val
	}

	if val, ok := config.AuthParams["saslTokenProvider"]; ok {
		if tokenProviderTypeValue != "" {
			return errors.New("unable to set `saslTokenProvider` in both ScaledObject and TriggerAuthentication together")
		}
		tokenProviderTypeValue = val
	}

	tokenProviderType := KafkaSASLOAuthTokenProviderBearer
	if tokenProviderTypeValue != "" {
		tokenProviderType = kafkaSaslOAuthTokenProvider(strings.TrimSpace(tokenProviderTypeValue))
	}

	var tokenProviderErr error
	switch tokenProviderType {
	case KafkaSASLOAuthTokenProviderBearer:
		tokenProviderErr = parseSaslOAuthBearerParams(config, meta)
	case KafkaSASLOAuthTokenProviderAWSMSKIAM:
		tokenProviderErr = parseSaslOAuthAWSMSKIAMParams(config, meta)
	default:
		return fmt.Errorf("err SASL OAuth token provider %s given", tokenProviderType)
	}

	if tokenProviderErr != nil {
		return fmt.Errorf("error parsing OAuth token provider configuration: %w", tokenProviderErr)
	}

	meta.saslType = mode
	meta.tokenProvider = tokenProviderType

	return nil
}

func parseSaslOAuthBearerParams(config *scalersconfig.ScalerConfig, meta *kafkaMetadata) error {
	if config.AuthParams["username"] == "" {
		return errors.New("no username given")
	}
	meta.username = strings.TrimSpace(config.AuthParams["username"])

	if config.AuthParams["password"] == "" {
		return errors.New("no password given")
	}
	meta.password = strings.TrimSpace(config.AuthParams["password"])

	meta.scopes = strings.Split(config.AuthParams["scopes"], ",")

	if config.AuthParams["oauthTokenEndpointUri"] == "" {
		return errors.New("no oauth token endpoint uri given")
	}
	meta.oauthTokenEndpointURI = strings.TrimSpace(config.AuthParams["oauthTokenEndpointUri"])

	meta.oauthExtensions = make(map[string]string)
	oauthExtensionsRaw := config.AuthParams["oauthExtensions"]
	if oauthExtensionsRaw != "" {
		for _, extension := range strings.Split(oauthExtensionsRaw, ",") {
			splittedExtension := strings.Split(extension, "=")
			if len(splittedExtension) != 2 {
				return errors.New("invalid OAuthBearer extension, must be of format key=value")
			}
			meta.oauthExtensions[splittedExtension[0]] = splittedExtension[1]
		}
	}

	return nil
}

func parseSaslOAuthAWSMSKIAMParams(config *scalersconfig.ScalerConfig, meta *kafkaMetadata) error {
	if !meta.enableTLS {
		return errors.New("TLS is required for AWS MSK authentication")
	}

	if config.TriggerMetadata["awsRegion"] == "" {
		return errors.New("no awsRegion given")
	}

	meta.awsRegion = config.TriggerMetadata["awsRegion"]

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, meta.awsRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return fmt.Errorf("error getting AWS authorization: %w", err)
	}

	meta.awsAuthorization = auth
	return nil
}

func parseTLS(config *scalersconfig.ScalerConfig, meta *kafkaMetadata) error {
	certGiven := config.AuthParams["cert"] != ""
	keyGiven := config.AuthParams["key"] != ""
	if certGiven && !keyGiven {
		return errors.New("key must be provided with cert")
	}
	if keyGiven && !certGiven {
		return errors.New("cert must be provided with key")
	}
	meta.ca = config.AuthParams["ca"]
	meta.cert = config.AuthParams["cert"]
	meta.key = config.AuthParams["key"]
	meta.unsafeSsl = defaultUnsafeSsl

	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		unsafeSsl, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		meta.unsafeSsl = unsafeSsl
	}

	if value, found := config.AuthParams["keyPassword"]; found {
		meta.keyPassword = value
	} else {
		meta.keyPassword = ""
	}
	meta.enableTLS = true
	return nil
}

func parseKerberosParams(config *scalersconfig.ScalerConfig, meta *kafkaMetadata, mode kafkaSaslType) error {
	if config.AuthParams["username"] == "" {
		return errors.New("no username given")
	}
	meta.username = strings.TrimSpace(config.AuthParams["username"])

	if (config.AuthParams["password"] == "" && config.AuthParams["keytab"] == "") ||
		(config.AuthParams["password"] != "" && config.AuthParams["keytab"] != "") {
		return errors.New("exactly one of 'password' or 'keytab' must be provided for GSSAPI authentication")
	}
	if config.AuthParams["password"] != "" {
		meta.password = strings.TrimSpace(config.AuthParams["password"])
	} else {
		path, err := saveToFile(config.AuthParams["keytab"])
		if err != nil {
			return fmt.Errorf("error saving keytab to file: %w", err)
		}
		meta.keytabPath = path
	}

	if config.AuthParams["realm"] == "" {
		return errors.New("no realm given")
	}
	meta.realm = strings.TrimSpace(config.AuthParams["realm"])

	if config.AuthParams["kerberosConfig"] == "" {
		return errors.New("no Kerberos configuration file (kerberosConfig) given")
	}
	path, err := saveToFile(config.AuthParams["kerberosConfig"])
	if err != nil {
		return fmt.Errorf("error saving kerberosConfig to file: %w", err)
	}
	meta.kerberosConfigPath = path

	if config.AuthParams["kerberosServiceName"] != "" {
		meta.kerberosServiceName = strings.TrimSpace(config.AuthParams["kerberosServiceName"])
	}

	meta.kerberosDisableFAST = false
	if val, ok := config.AuthParams["kerberosDisableFAST"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing kerberosDisableFAST: %w", err)
		}
		meta.kerberosDisableFAST = t
	}

	meta.saslType = mode
	return nil
}

func parseSaslParams(config *scalersconfig.ScalerConfig, meta *kafkaMetadata, mode kafkaSaslType) error {
	if config.AuthParams["username"] == "" {
		return errors.New("no username given")
	}
	meta.username = strings.TrimSpace(config.AuthParams["username"])

	if config.AuthParams["password"] == "" {
		return errors.New("no password given")
	}
	meta.password = strings.TrimSpace(config.AuthParams["password"])
	meta.saslType = mode

	return nil
}

func saveToFile(content string) (string, error) {
	data := []byte(content)

	tempKrbDir := fmt.Sprintf("%s%c%s", os.TempDir(), os.PathSeparator, "kerberos")
	err := os.MkdirAll(tempKrbDir, 0700)
	if err != nil {
		return "", fmt.Errorf(`error creating temporary directory: %s.  Error: %w
		Note, when running in a container a writable /tmp/kerberos emptyDir must be mounted.  Refer to documentation`, tempKrbDir, err)
	}

	tempFile, err := os.CreateTemp(tempKrbDir, "krb_*")
	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %w", err)
	}
	defer tempFile.Close()

	_, err = tempFile.Write(data)
	if err != nil {
		return "", fmt.Errorf("error writing to temporary file: %w", err)
	}

	// Get the temporary file's name
	tempFilename := tempFile.Name()

	return tempFilename, nil
}

func parseKafkaMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (kafkaMetadata, error) {
	meta := kafkaMetadata{}
	switch {
	case config.TriggerMetadata["bootstrapServersFromEnv"] != "":
		meta.bootstrapServers = strings.Split(config.ResolvedEnv[config.TriggerMetadata["bootstrapServersFromEnv"]], ",")
	case config.TriggerMetadata["bootstrapServers"] != "":
		meta.bootstrapServers = strings.Split(config.TriggerMetadata["bootstrapServers"], ",")
	default:
		return meta, errors.New("no bootstrapServers given")
	}

	switch {
	case config.TriggerMetadata["consumerGroupFromEnv"] != "":
		meta.group = config.ResolvedEnv[config.TriggerMetadata["consumerGroupFromEnv"]]
	case config.TriggerMetadata["consumerGroup"] != "":
		meta.group = config.TriggerMetadata["consumerGroup"]
	default:
		return meta, errors.New("no consumer group given")
	}

	switch {
	case config.TriggerMetadata["topicFromEnv"] != "":
		meta.topic = config.ResolvedEnv[config.TriggerMetadata["topicFromEnv"]]
	case config.TriggerMetadata["topic"] != "":
		meta.topic = config.TriggerMetadata["topic"]
	default:
		meta.topic = ""
		logger.V(1).Info(fmt.Sprintf("consumer group %q has no topic specified, "+
			"will use all topics subscribed by the consumer group for scaling", meta.group))
	}

	meta.partitionLimitation = nil
	partitionLimitationMetadata := strings.TrimSpace(config.TriggerMetadata["partitionLimitation"])
	if partitionLimitationMetadata != "" {
		if meta.topic == "" {
			logger.V(1).Info("no specific topic set, ignoring partitionLimitation setting")
		} else {
			pattern := config.TriggerMetadata["partitionLimitation"]
			parsed, err := kedautil.ParseInt32List(pattern)
			if err != nil {
				return meta, fmt.Errorf("error parsing in partitionLimitation '%s': %w", pattern, err)
			}
			meta.partitionLimitation = parsed
			logger.V(0).Info(fmt.Sprintf("partition limit active '%s'", pattern))
		}
	}

	meta.offsetResetPolicy = defaultOffsetResetPolicy

	if config.TriggerMetadata["offsetResetPolicy"] != "" {
		policy := offsetResetPolicy(config.TriggerMetadata["offsetResetPolicy"])
		if policy != earliest && policy != latest {
			return meta, fmt.Errorf("err offsetResetPolicy policy %q given", policy)
		}
		meta.offsetResetPolicy = policy
	}

	meta.lagThreshold = defaultKafkaLagThreshold

	if val, ok := config.TriggerMetadata[lagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %q: %w", lagThresholdMetricName, err)
		}
		if t <= 0 {
			return meta, fmt.Errorf("%q must be positive number", lagThresholdMetricName)
		}
		meta.lagThreshold = t
	}

	meta.activationLagThreshold = defaultKafkaActivationLagThreshold

	if val, ok := config.TriggerMetadata[activationLagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %q: %w", activationLagThresholdMetricName, err)
		}
		if t < 0 {
			return meta, fmt.Errorf("%q must be positive number", activationLagThresholdMetricName)
		}
		meta.activationLagThreshold = t
	}

	if err := parseKafkaAuthParams(config, &meta); err != nil {
		return meta, err
	}

	meta.allowIdleConsumers = false
	if val, ok := config.TriggerMetadata["allowIdleConsumers"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing allowIdleConsumers: %w", err)
		}
		meta.allowIdleConsumers = t
	}

	meta.excludePersistentLag = false
	if val, ok := config.TriggerMetadata["excludePersistentLag"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing excludePersistentLag: %w", err)
		}
		meta.excludePersistentLag = t
	}

	meta.scaleToZeroOnInvalidOffset = false
	if val, ok := config.TriggerMetadata["scaleToZeroOnInvalidOffset"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing scaleToZeroOnInvalidOffset: %w", err)
		}
		meta.scaleToZeroOnInvalidOffset = t
	}

	meta.limitToPartitionsWithLag = false
	if val, ok := config.TriggerMetadata["limitToPartitionsWithLag"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing limitToPartitionsWithLag: %w", err)
		}
		meta.limitToPartitionsWithLag = t

		if meta.allowIdleConsumers && meta.limitToPartitionsWithLag {
			return meta, fmt.Errorf("allowIdleConsumers and limitToPartitionsWithLag cannot be set simultaneously")
		}
		if len(meta.topic) == 0 && meta.limitToPartitionsWithLag {
			return meta, fmt.Errorf("topic must be specified when using limitToPartitionsWithLag")
		}
	}

	meta.ensureEvenDistributionOfPartitions = false
	if val, ok := config.TriggerMetadata["ensureEvenDistributionOfPartitions"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing ensureEvenDistributionOfPartitions: %w", err)
		}
		meta.ensureEvenDistributionOfPartitions = t

		if meta.limitToPartitionsWithLag && meta.ensureEvenDistributionOfPartitions {
			return meta, fmt.Errorf("limitToPartitionsWithLag and ensureEvenDistributionOfPartitions cannot be set simultaneously")
		}
		if len(meta.topic) == 0 && meta.ensureEvenDistributionOfPartitions {
			return meta, fmt.Errorf("topic must be specified when using ensureEvenDistributionOfPartitions")
		}
	}

	meta.version = sarama.V1_0_0_0
	if val, ok := config.TriggerMetadata["version"]; ok {
		val = strings.TrimSpace(val)
		version, err := sarama.ParseKafkaVersion(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing kafka version: %w", err)
		}
		meta.version = version
	}
	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}

func getKafkaClients(ctx context.Context, metadata kafkaMetadata) (sarama.Client, sarama.ClusterAdmin, error) {
	config, err := getKafkaClientConfig(ctx, metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting kafka client config: %w", err)
	}

	client, err := sarama.NewClient(metadata.bootstrapServers, config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating kafka client: %w", err)
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		if !client.Closed() {
			client.Close()
		}
		return nil, nil, fmt.Errorf("error creating kafka admin: %w", err)
	}

	return client, admin, nil
}

func getKafkaClientConfig(ctx context.Context, metadata kafkaMetadata) (*sarama.Config, error) {
	config := sarama.NewConfig()
	config.Version = metadata.version

	if metadata.saslType != KafkaSASLTypeNone && metadata.saslType != KafkaSASLTypeGSSAPI {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = metadata.username
		config.Net.SASL.Password = metadata.password
	}

	if metadata.enableTLS {
		config.Net.TLS.Enable = true
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(metadata.cert, metadata.key, metadata.keyPassword, metadata.ca, metadata.unsafeSsl)
		if err != nil {
			return nil, err
		}
		config.Net.TLS.Config = tlsConfig
	}

	if metadata.saslType == KafkaSASLTypePlaintext {
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if metadata.saslType == KafkaSASLTypeSCRAMSHA256 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &kafka.XDGSCRAMClient{HashGeneratorFcn: kafka.SHA256} }
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	}

	if metadata.saslType == KafkaSASLTypeSCRAMSHA512 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &kafka.XDGSCRAMClient{HashGeneratorFcn: kafka.SHA512} }
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	}

	if metadata.saslType == KafkaSASLTypeOAuthbearer {
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		switch metadata.tokenProvider {
		case KafkaSASLOAuthTokenProviderBearer:
			config.Net.SASL.TokenProvider = kafka.OAuthBearerTokenProvider(metadata.username, metadata.password, metadata.oauthTokenEndpointURI, metadata.scopes, metadata.oauthExtensions)
		case KafkaSASLOAuthTokenProviderAWSMSKIAM:
			awsAuth, err := awsutils.GetAwsConfig(ctx, metadata.awsAuthorization)
			if err != nil {
				return nil, fmt.Errorf("error getting AWS config: %w", err)
			}

			config.Net.SASL.TokenProvider = kafka.OAuthMSKTokenProvider(awsAuth)
		default:
			return nil, fmt.Errorf("err SASL OAuth token provider %s given but not supported", metadata.tokenProvider)
		}
	}

	if metadata.saslType == KafkaSASLTypeGSSAPI {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		if metadata.kerberosServiceName != "" {
			config.Net.SASL.GSSAPI.ServiceName = metadata.kerberosServiceName
		} else {
			config.Net.SASL.GSSAPI.ServiceName = "kafka"
		}
		config.Net.SASL.GSSAPI.Username = metadata.username
		config.Net.SASL.GSSAPI.Realm = metadata.realm
		config.Net.SASL.GSSAPI.KerberosConfigPath = metadata.kerberosConfigPath
		if metadata.keytabPath != "" {
			config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
			config.Net.SASL.GSSAPI.KeyTabPath = metadata.keytabPath
		} else {
			config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_USER_AUTH
			config.Net.SASL.GSSAPI.Password = metadata.password
		}

		if metadata.kerberosDisableFAST {
			config.Net.SASL.GSSAPI.DisablePAFXFAST = true
		}
	}

	return config, nil
}

func (s *kafkaScaler) getTopicPartitions() (map[string][]int32, error) {
	var topicsToDescribe = make([]string, 0)

	// when no topic is specified, query to cg group to fetch all subscribed topics
	if s.metadata.topic == "" {
		listCGOffsetResponse, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, nil)
		if err != nil {
			return nil, fmt.Errorf("error listing cg offset: %w", err)
		}

		if listCGOffsetResponse.Err > 0 {
			errMsg := fmt.Errorf("error listing cg offset: %w", listCGOffsetResponse.Err)
			s.logger.Error(errMsg, "")
			return nil, errMsg
		}

		for topicName := range listCGOffsetResponse.Blocks {
			topicsToDescribe = append(topicsToDescribe, topicName)
		}
	} else {
		topicsToDescribe = []string{s.metadata.topic}
	}

	topicsMetadata, err := s.admin.DescribeTopics(topicsToDescribe)
	if err != nil {
		return nil, fmt.Errorf("error describing topics: %w", err)
	}
	s.logger.V(1).Info(
		fmt.Sprintf("with topic name %s the list of topic metadata is %v", topicsToDescribe, topicsMetadata),
	)

	if s.metadata.topic != "" && len(topicsMetadata) != 1 {
		return nil, fmt.Errorf("expected only 1 topic metadata, got %d", len(topicsMetadata))
	}

	topicPartitions := make(map[string][]int32, len(topicsMetadata))
	for _, topicMetadata := range topicsMetadata {
		if topicMetadata.Err > 0 {
			errMsg := fmt.Errorf("error describing topics: %w", topicMetadata.Err)
			s.logger.Error(errMsg, "")
			return nil, errMsg
		}
		partitionMetadata := topicMetadata.Partitions
		var partitions []int32
		for _, p := range partitionMetadata {
			if s.isActivePartition(p.ID) {
				partitions = append(partitions, p.ID)
			}
		}
		if len(partitions) == 0 {
			return nil, fmt.Errorf("expected at least one active partition within the topic '%s'", topicMetadata.Name)
		}

		topicPartitions[topicMetadata.Name] = partitions
	}
	return topicPartitions, nil
}

func (s *kafkaScaler) isActivePartition(pID int32) bool {
	if s.metadata.partitionLimitation == nil {
		return true
	}
	for _, _pID := range s.metadata.partitionLimitation {
		if pID == _pID {
			return true
		}
	}
	return false
}

func (s *kafkaScaler) getConsumerOffsets(topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	offsets, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, topicPartitions)
	if err != nil {
		return nil, fmt.Errorf("error listing consumer group offsets: %w", err)
	}
	if offsets.Err > 0 {
		errMsg := fmt.Errorf("error listing consumer group offsets: %w", offsets.Err)
		s.logger.Error(errMsg, "")
		return nil, errMsg
	}
	return offsets, nil
}

// getLagForPartition returns (lag, lagWithPersistent, error)
// When excludePersistentLag is set to `false` (default), lag will always be equal to lagWithPersistent
// When excludePersistentLag is set to `true`, if partition is deemed to have persistent lag, lag will be set to 0 and lagWithPersistent will be latestOffset - consumerOffset
// These return values will allow proper scaling from 0 -> 1 replicas by the IsActive func.
func (s *kafkaScaler) getLagForPartition(topic string, partitionID int32, offsets *sarama.OffsetFetchResponse, topicPartitionOffsets map[string]map[int32]int64) (int64, int64, error) {
	block := offsets.GetBlock(topic, partitionID)
	if block == nil {
		errMsg := fmt.Errorf("error finding offset block for topic %s and partition %d from offset block: %v", topic, partitionID, offsets.Blocks)
		s.logger.Error(errMsg, "")
		return 0, 0, errMsg
	}
	if block.Err > 0 {
		errMsg := fmt.Errorf("error finding offset block for topic %s and partition %d: %w", topic, partitionID, offsets.Err)
		s.logger.Error(errMsg, "")
		return 0, 0, errMsg
	}

	consumerOffset := block.Offset

	// Handle invalid consumer offset for both latest and earliest policies
	// This must be done before getting latestOffset, so scaleToZeroOnInvalidOffset works
	// even when latestOffset cannot be retrieved (e.g., missing partition in response)
	if consumerOffset == invalidOffset {
		if s.metadata.offsetResetPolicy == latest {
			retVal := int64(1)
			if s.metadata.scaleToZeroOnInvalidOffset {
				retVal = 0
			}
			msg := fmt.Sprintf(
				"invalid offset found for topic %s in group %s and partition %d, probably no offset is committed yet. Returning with lag of %d",
				topic, s.metadata.group, partitionID, retVal)
			s.logger.V(1).Info(msg)
			return retVal, retVal, nil
		}
		// offsetResetPolicy == earliest
		// For earliest policy, we need latestOffset to return the full lag when scaleToZeroOnInvalidOffset is false
		// But if we can't get latestOffset, we should still respect scaleToZeroOnInvalidOffset
		if s.metadata.scaleToZeroOnInvalidOffset {
			return 0, 0, nil
		}
	}

	topicOffsets, found := topicPartitionOffsets[topic]
	if !found {
		s.logger.V(1).Info(fmt.Sprintf("Topic %s not found in latest offset response, treating partition %d as 0 lag", topic, partitionID))
		return 0, 0, nil
	}
	latestOffset, partitionFound := topicOffsets[partitionID]
	if !partitionFound {
		// Partition missing from latest offset response - treat as 0 lag and continue
		// This can happen with Azure Event Hub when partitions are intermittently not returned
		s.logger.V(1).Info(fmt.Sprintf("Partition %d in topic %s not found in latest offset response, treating as 0 lag", partitionID, topic))
		return 0, 0, nil
	}

	// If we got here with invalidOffset and earliest policy, scaleToZeroOnInvalidOffset must be false
	// Return the full lag (latestOffset) as per earliest policy behavior
	if consumerOffset == invalidOffset && s.metadata.offsetResetPolicy == earliest {
		return latestOffset, latestOffset, nil
	}

	// This code block tries to prevent KEDA Kafka trigger from scaling the scale target based on erroneous events
	if s.metadata.excludePersistentLag {
		switch previousOffset, found := s.previousOffsets[topic][partitionID]; {
		case !found:
			// No record of previous offset, so store current consumer offset
			// Allow this consumer lag to be considered in scaling
			if _, topicFound := s.previousOffsets[topic]; !topicFound {
				s.previousOffsets[topic] = map[int32]int64{partitionID: consumerOffset}
			} else {
				s.previousOffsets[topic][partitionID] = consumerOffset
			}
		case previousOffset == consumerOffset:
			// Indicates consumer is still on the same offset as the previous polling cycle, there may be some issue with consuming this offset.
			// return 0, so this consumer lag is not considered for scaling
			return 0, latestOffset - consumerOffset, nil
		default:
			// Successfully Consumed some messages, proceed to change the previous offset
			s.previousOffsets[topic][partitionID] = consumerOffset
		}
	}

	return latestOffset - consumerOffset, latestOffset - consumerOffset, nil
}

// Close closes the kafka admin and client
func (s *kafkaScaler) Close(context.Context) error {
	// clean up any temporary files
	if strings.TrimSpace(s.metadata.kerberosConfigPath) != "" {
		if err := os.Remove(s.metadata.kerberosConfigPath); err != nil {
			return err
		}
	}
	if strings.TrimSpace(s.metadata.keytabPath) != "" {
		if err := os.Remove(s.metadata.keytabPath); err != nil {
			return err
		}
	}
	// underlying client will also be closed on admin's Close() call
	if s.admin == nil {
		return nil
	}

	return s.admin.Close()
}

func (s *kafkaScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricName string
	if s.metadata.topic != "" {
		metricName = fmt.Sprintf("kafka-%s", s.metadata.topic)
	} else {
		metricName = fmt.Sprintf("kafka-%s-topics", s.metadata.group)
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(metricName)),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.lagThreshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: kafkaMetricType}
	return []v2.MetricSpec{metricSpec}
}

type consumerOffsetResult struct {
	consumerOffsets *sarama.OffsetFetchResponse
	err             error
}

type producerOffsetResult struct {
	producerOffsets map[string]map[int32]int64
	err             error
}

func (s *kafkaScaler) getConsumerAndProducerOffsets(topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, map[string]map[int32]int64, error) {
	consumerChan := make(chan consumerOffsetResult, 1)
	go func() {
		consumerOffsets, err := s.getConsumerOffsets(topicPartitions)
		consumerChan <- consumerOffsetResult{consumerOffsets, err}
	}()

	producerChan := make(chan producerOffsetResult, 1)
	go func() {
		producerOffsets, err := s.getProducerOffsets(topicPartitions)
		producerChan <- producerOffsetResult{producerOffsets, err}
	}()

	consumerRes := <-consumerChan
	if consumerRes.err != nil {
		return nil, nil, consumerRes.err
	}

	producerRes := <-producerChan
	if producerRes.err != nil {
		return nil, nil, producerRes.err
	}

	return consumerRes.consumerOffsets, producerRes.producerOffsets, nil
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *kafkaScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	totalLag, totalLagWithPersistent, err := s.getTotalLag()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, totalLagWithPersistent > s.metadata.activationLagThreshold, nil
}

// getTotalLag returns totalLag, totalLagWithPersistent, error
// totalLag and totalLagWithPersistent are the summations of lag and lagWithPersistent returned by getLagForPartition function respectively.
// totalLag maybe less than totalLagWithPersistent when excludePersistentLag is set to `true` due to some partitions deemed as having persistent lag
func (s *kafkaScaler) getTotalLag() (int64, int64, error) {
	topicPartitions, err := s.getTopicPartitions()
	if err != nil {
		return 0, 0, err
	}

	consumerOffsets, producerOffsets, err := s.getConsumerAndProducerOffsets(topicPartitions)
	if err != nil {
		return 0, 0, err
	}

	totalLag := int64(0)
	totalLagWithPersistent := int64(0)
	totalTopicPartitions := int64(0)
	partitionsWithLag := int64(0)

	for topic, partitionsOffsets := range producerOffsets {
		for partition := range partitionsOffsets {
			lag, lagWithPersistent, err := s.getLagForPartition(topic, partition, consumerOffsets, producerOffsets)
			if err != nil {
				return 0, 0, err
			}
			totalLag += lag
			totalLagWithPersistent += lagWithPersistent

			if lag > 0 {
				partitionsWithLag++
			}
		}
		totalTopicPartitions += (int64)(len(partitionsOffsets))
	}
	s.logger.V(1).Info(fmt.Sprintf("Kafka scaler: Providing metrics based on totalLag %v, topicPartitions %v, threshold %v", totalLag, len(topicPartitions), s.metadata.lagThreshold))

	if !s.metadata.allowIdleConsumers || s.metadata.limitToPartitionsWithLag || s.metadata.ensureEvenDistributionOfPartitions {
		// don't scale out beyond the number of topicPartitions or partitionsWithLag depending on settings
		upperBound := totalTopicPartitions
		// Ensure that the number of partitions is evenly distributed across the number of consumers
		if s.metadata.ensureEvenDistributionOfPartitions {
			nextFactor := getNextFactorThatBalancesConsumersToTopicPartitions(totalLag, totalTopicPartitions, s.metadata.lagThreshold)
			s.logger.V(1).Info(fmt.Sprintf("Kafka scaler: Providing metrics to ensure even distribution of partitions on totalLag %v, topicPartitions %v, evenPartitions %v", totalLag, totalTopicPartitions, nextFactor))
			totalLag = nextFactor * s.metadata.lagThreshold
		}
		if s.metadata.limitToPartitionsWithLag {
			upperBound = partitionsWithLag
		}

		if (totalLag / s.metadata.lagThreshold) > upperBound {
			totalLag = upperBound * s.metadata.lagThreshold
		}
	}
	return totalLag, totalLagWithPersistent, nil
}

func getNextFactorThatBalancesConsumersToTopicPartitions(totalLag int64, totalTopicPartitions int64, lagThreshold int64) int64 {
	factors := FindFactors(totalTopicPartitions)
	for _, factor := range factors {
		if factor*lagThreshold >= totalLag {
			return factor
		}
	}
	return totalTopicPartitions
}

type brokerOffsetResult struct {
	offsetResp *sarama.OffsetResponse
	err        error
}

func (s *kafkaScaler) getProducerOffsets(topicPartitions map[string][]int32) (map[string]map[int32]int64, error) {
	version := int16(0)
	if s.client.Config().Version.IsAtLeast(sarama.V0_10_1_0) {
		version = 1
	}

	// Step 1: build one OffsetRequest instance per broker.
	requests := make(map[*sarama.Broker]*sarama.OffsetRequest)

	for topic, partitions := range topicPartitions {
		for _, partitionID := range partitions {
			broker, err := s.client.Leader(topic, partitionID)
			if err != nil {
				return nil, err
			}
			request, ok := requests[broker]
			if !ok {
				request = &sarama.OffsetRequest{Version: version}
				requests[broker] = request
			}
			request.AddBlock(topic, partitionID, sarama.OffsetNewest, 1)
		}
	}

	// Step 2: send requests, one per broker, and collect topicPartitionsOffsets
	resultCh := make(chan brokerOffsetResult, len(requests))
	var wg sync.WaitGroup
	wg.Add(len(requests))
	for broker, request := range requests {
		go func(brCopy *sarama.Broker, reqCopy *sarama.OffsetRequest) {
			defer wg.Done()
			response, err := brCopy.GetAvailableOffsets(reqCopy)
			resultCh <- brokerOffsetResult{response, err}
		}(broker, request)
	}

	wg.Wait()
	close(resultCh)

	topicPartitionsOffsets := make(map[string]map[int32]int64)
	for brokerOffsetRes := range resultCh {
		if brokerOffsetRes.err != nil {
			return nil, brokerOffsetRes.err
		}

		for topic, blocks := range brokerOffsetRes.offsetResp.Blocks {
			if _, found := topicPartitionsOffsets[topic]; !found {
				topicPartitionsOffsets[topic] = make(map[int32]int64)
			}
			for partitionID, block := range blocks {
				if block.Err != sarama.ErrNoError {
					return nil, block.Err
				}
				topicPartitionsOffsets[topic][partitionID] = block.Offset
			}
		}
	}

	return topicPartitionsOffsets, nil
}

// FindFactors returns all factors of a given number
func FindFactors(n int64) []int64 {
	if n < 1 {
		return nil
	}

	var factors []int64
	sqrtN := int64(math.Sqrt(float64(n)))

	for i := int64(1); i <= sqrtN; i++ {
		if n%i == 0 {
			factors = append(factors, i)
			if i != n/i {
				factors = append(factors, n/i)
			}
		}
	}

	sort.Slice(factors, func(i, j int) bool {
		return factors[i] < factors[j]
	})

	return factors
}
