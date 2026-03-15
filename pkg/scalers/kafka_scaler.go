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
	stringEnable  = "enable"
	stringDisable = "disable"
)

type kafkaMetadata struct {
	BootstrapServers []string `keda:"name=bootstrapServers,order=triggerMetadata;resolvedEnv"`
	Group            string   `keda:"name=consumerGroup,order=triggerMetadata;resolvedEnv"`
	Topic            string   `keda:"name=topic,order=triggerMetadata;resolvedEnv,optional"`

	PartitionLimitationStr string  `keda:"name=partitionLimitation,order=triggerMetadata,optional"`
	PartitionLimitation    []int32 // computed in Validate

	LagThreshold           int64 `keda:"name=lagThreshold,order=triggerMetadata,default=10"`
	ActivationLagThreshold int64 `keda:"name=activationLagThreshold,order=triggerMetadata,default=0"`

	OffsetResetPolicy                  offsetResetPolicy `keda:"name=offsetResetPolicy,order=triggerMetadata,default=latest,enum=latest;earliest"`
	AllowIdleConsumers                 bool              `keda:"name=allowIdleConsumers,order=triggerMetadata,default=false"`
	ExcludePersistentLag               bool              `keda:"name=excludePersistentLag,order=triggerMetadata,default=false"`
	ScaleToZeroOnInvalidOffset         bool              `keda:"name=scaleToZeroOnInvalidOffset,order=triggerMetadata,default=false"`
	LimitToPartitionsWithLag           bool              `keda:"name=limitToPartitionsWithLag,order=triggerMetadata,default=false"`
	EnsureEvenDistributionOfPartitions bool              `keda:"name=ensureEvenDistributionOfPartitions,order=triggerMetadata,default=false"`

	VersionStr string `keda:"name=version,order=triggerMetadata,optional"`

	TLS         string `keda:"name=tls,order=triggerMetadata;authParams,default=disable,enum=enable;disable"`
	UnsafeSsl   bool   `keda:"name=unsafeSsl,order=triggerMetadata,default=false"`
	CA          string `keda:"name=ca,order=authParams,optional"`
	Cert        string `keda:"name=cert,order=authParams,optional"`
	Key         string `keda:"name=key,order=authParams,optional"`
	KeyPassword string `keda:"name=keyPassword,order=authParams,optional"`

	Sasl     string `keda:"name=sasl,order=triggerMetadata;authParams,optional,enum=none;plaintext;scram_sha256;scram_sha512;oauthbearer;gssapi"`
	Username string `keda:"name=username,order=authParams,optional"`
	Password string `keda:"name=password,order=authParams,optional"`

	SaslTokenProvider     string `keda:"name=saslTokenProvider,order=triggerMetadata;authParams,optional,enum=bearer;aws_msk_iam"`
	ScopesStr             string `keda:"name=scopes,order=authParams,optional"`
	OAuthTokenEndpointURI string `keda:"name=oauthTokenEndpointUri,order=authParams,optional"`
	OAuthExtensionsStr    string `keda:"name=oauthExtensions,order=authParams,optional"`

	Keytab              string `keda:"name=keytab,order=authParams,optional"`
	Realm               string `keda:"name=realm,order=authParams,optional"`
	KerberosConfigRaw   string `keda:"name=kerberosConfig,order=authParams,optional"`
	KerberosServiceName string `keda:"name=kerberosServiceName,order=authParams,optional"`
	KerberosDisableFAST bool   `keda:"name=kerberosDisableFAST,order=authParams,default=false"`

	AWSRegion string `keda:"name=awsRegion,order=triggerMetadata,optional"`

	version            sarama.KafkaVersion
	saslType           kafkaSaslType
	tokenProvider      kafkaSaslOAuthTokenProvider
	enableTLS          bool
	keytabPath         string
	kerberosConfigPath string
	awsAuthorization   awsutils.AuthorizationMetadata
	scopes             []string
	oauthExtensions    map[string]string

	triggerIndex int
}

func (m *kafkaMetadata) Validate() error {
	if m.LagThreshold <= 0 {
		return fmt.Errorf("%q must be positive number", lagThresholdMetricName)
	}
	if m.ActivationLagThreshold < 0 {
		return fmt.Errorf("%q must be a non-negative number", activationLagThresholdMetricName)
	}

	if m.PartitionLimitationStr != "" && strings.TrimSpace(m.PartitionLimitationStr) != "" {
		limitArray, err := parsePartitionLimitation(m.PartitionLimitationStr)
		if err != nil {
			return err
		}
		m.PartitionLimitation = limitArray
	}

	if m.Topic == "" {
		m.PartitionLimitation = nil
	}

	if m.AllowIdleConsumers && m.LimitToPartitionsWithLag {
		return fmt.Errorf("allowIdleConsumers and limitToPartitionsWithLag cannot be set simultaneously")
	}
	if len(m.Topic) == 0 && m.LimitToPartitionsWithLag {
		return fmt.Errorf("topic must be specified when using limitToPartitionsWithLag")
	}
	if m.LimitToPartitionsWithLag && m.EnsureEvenDistributionOfPartitions {
		return fmt.Errorf("limitToPartitionsWithLag and ensureEvenDistributionOfPartitions cannot be set simultaneously")
	}
	if len(m.Topic) == 0 && m.EnsureEvenDistributionOfPartitions {
		return fmt.Errorf("topic must be specified when using ensureEvenDistributionOfPartitions")
	}

	if err := m.parseTLS(); err != nil {
		return err
	}
	if err := m.parseSASL(); err != nil {
		return err
	}

	return nil
}

func parsePartitionLimitation(partitionLimitationStr string) ([]int32, error) {
	partitionLimitation := make([]int32, 0)
	for _, part := range strings.Split(partitionLimitationStr, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid partition limitation range: %s", part)
			}
			start, err := strconv.ParseInt(strings.TrimSpace(rangeParts[0]), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid partition limitation: %s", part)
			}
			end, err := strconv.ParseInt(strings.TrimSpace(rangeParts[1]), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid partition limitation: %s", part)
			}
			for i := start; i <= end; i++ {
				partitionLimitation = append(partitionLimitation, int32(i))
			}
		} else {
			val, err := strconv.ParseInt(part, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid partition limitation: %s", part)
			}
			partitionLimitation = append(partitionLimitation, int32(val))
		}
	}
	return partitionLimitation, nil
}

func (m *kafkaMetadata) parseTLS() error {
	switch m.TLS {
	case "", stringDisable:
		m.enableTLS = false
	case stringEnable:
		if m.Cert != "" && m.Key == "" {
			return errors.New("key must be provided with cert")
		}
		if m.Key != "" && m.Cert == "" {
			return errors.New("cert must be provided with key")
		}
		m.enableTLS = true
	}
	return nil
}

func (m *kafkaMetadata) parseSASL() error {
	m.saslType = KafkaSASLTypeNone
	if m.Sasl == "" {
		return nil
	}

	mode := kafkaSaslType(m.Sasl)
	if mode == KafkaSASLTypeNone {
		return nil
	}

	switch mode {
	case KafkaSASLTypePlaintext, KafkaSASLTypeSCRAMSHA256, KafkaSASLTypeSCRAMSHA512:
		if m.Username == "" {
			return errors.New("no username given")
		}
		if m.Password == "" {
			return errors.New("no password given")
		}
		m.saslType = mode

	case KafkaSASLTypeOAuthbearer:
		if err := m.parseOAuthParams(); err != nil {
			return fmt.Errorf("error parsing OAuth token provider configuration: %w", err)
		}
		m.saslType = mode

	case KafkaSASLTypeGSSAPI:
		if err := m.parseGSSAPIParams(); err != nil {
			return err
		}
		m.saslType = mode
	}
	return nil
}

func (m *kafkaMetadata) parseOAuthParams() error {
	tokenProvider := KafkaSASLOAuthTokenProviderBearer
	if m.SaslTokenProvider != "" {
		tokenProvider = kafkaSaslOAuthTokenProvider(m.SaslTokenProvider)
	}

	switch tokenProvider {
	case KafkaSASLOAuthTokenProviderBearer:
		if m.Username == "" {
			return errors.New("no username given")
		}
		if m.Password == "" {
			return errors.New("no password given")
		}
		if m.OAuthTokenEndpointURI == "" {
			return errors.New("no oauth token endpoint uri given")
		}
		m.scopes = strings.Split(m.ScopesStr, ",")
		m.oauthExtensions = make(map[string]string)
		if m.OAuthExtensionsStr != "" {
			for _, ext := range strings.Split(m.OAuthExtensionsStr, ",") {
				kv := strings.Split(ext, "=")
				if len(kv) != 2 {
					return errors.New("invalid OAuthBearer extension, must be of format key=value")
				}
				m.oauthExtensions[kv[0]] = kv[1]
			}
		}

	case KafkaSASLOAuthTokenProviderAWSMSKIAM:
		if !m.enableTLS {
			return errors.New("TLS is required for AWS MSK authentication")
		}
		if m.AWSRegion == "" {
			return errors.New("no awsRegion given")
		}
	}

	m.tokenProvider = tokenProvider
	return nil
}

func (m *kafkaMetadata) parseGSSAPIParams() error {
	if m.Username == "" {
		return errors.New("no username given")
	}
	if (m.Password == "" && m.Keytab == "") || (m.Password != "" && m.Keytab != "") {
		return errors.New("exactly one of 'password' or 'keytab' must be provided for GSSAPI authentication")
	}
	if m.Realm == "" {
		return errors.New("no realm given")
	}
	if m.KerberosConfigRaw == "" {
		return errors.New("no Kerberos configuration file (kerberosConfig) given")
	}
	return nil
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

	return &kafkaScaler{
		client:          client,
		admin:           admin,
		metricType:      metricType,
		metadata:        kafkaMetadata,
		logger:          logger,
		previousOffsets: make(map[string]map[int32]int64),
	}, nil
}

func parseKafkaMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (kafkaMetadata, error) {
	meta := kafkaMetadata{}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, fmt.Errorf("error parsing kafka metadata: %w", err)
	}

	if meta.Topic == "" {
		logger.V(1).Info(fmt.Sprintf("consumer group %q has no topic specified, "+
			"will use all topics subscribed by the consumer group for scaling", meta.Group))
	}

	meta.version = sarama.V1_0_0_0
	if meta.VersionStr != "" {
		version, err := sarama.ParseKafkaVersion(meta.VersionStr)
		if err != nil {
			return meta, fmt.Errorf("error parsing kafka version: %w", err)
		}
		meta.version = version
	}

	if meta.saslType == KafkaSASLTypeGSSAPI {
		if meta.Keytab != "" {
			path, err := saveToFile(meta.Keytab)
			if err != nil {
				return meta, fmt.Errorf("error saving keytab to file: %w", err)
			}
			meta.keytabPath = path
		}
		if meta.KerberosConfigRaw != "" {
			path, err := saveToFile(meta.KerberosConfigRaw)
			if err != nil {
				return meta, fmt.Errorf("error saving kerberosConfig to file: %w", err)
			}
			meta.kerberosConfigPath = path
		}
	}

	if meta.saslType == KafkaSASLTypeOAuthbearer && meta.tokenProvider == KafkaSASLOAuthTokenProviderAWSMSKIAM {
		auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, meta.AWSRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
		if err != nil {
			return meta, fmt.Errorf("error getting AWS authorization: %w", err)
		}
		meta.awsAuthorization = auth
	}

	meta.triggerIndex = config.TriggerIndex
	return meta, nil
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

	return tempFile.Name(), nil
}

func getKafkaClients(ctx context.Context, metadata kafkaMetadata) (sarama.Client, sarama.ClusterAdmin, error) {
	config, err := getKafkaClientConfig(ctx, metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting kafka client config: %w", err)
	}

	client, err := sarama.NewClient(metadata.BootstrapServers, config)
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
		config.Net.SASL.User = metadata.Username
		config.Net.SASL.Password = metadata.Password
	}

	if metadata.enableTLS {
		config.Net.TLS.Enable = true
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(metadata.Cert, metadata.Key, metadata.KeyPassword, metadata.CA, metadata.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		config.Net.TLS.Config = tlsConfig
	}

	if metadata.saslType == KafkaSASLTypePlaintext {
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if metadata.saslType == KafkaSASLTypeSCRAMSHA256 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &kafka.XDGSCRAMClient{HashGeneratorFcn: kafka.SHA256}
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	}

	if metadata.saslType == KafkaSASLTypeSCRAMSHA512 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &kafka.XDGSCRAMClient{HashGeneratorFcn: kafka.SHA512}
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	}

	if metadata.saslType == KafkaSASLTypeOAuthbearer {
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		switch metadata.tokenProvider {
		case KafkaSASLOAuthTokenProviderBearer:
			config.Net.SASL.TokenProvider = kafka.OAuthBearerTokenProvider(
				metadata.Username, metadata.Password,
				metadata.OAuthTokenEndpointURI, metadata.scopes, metadata.oauthExtensions,
			)
		case KafkaSASLOAuthTokenProviderAWSMSKIAM:
			awsAuth, err := awsutils.GetAwsConfig(ctx, metadata.awsAuthorization)
			if err != nil {
				return nil, fmt.Errorf("error getting AWS config: %w", err)
			}
			config.Net.SASL.TokenProvider = kafka.OAuthMSKTokenProvider(awsAuth)
		}
	}

	if metadata.saslType == KafkaSASLTypeGSSAPI {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		if metadata.KerberosServiceName != "" {
			config.Net.SASL.GSSAPI.ServiceName = metadata.KerberosServiceName
		} else {
			config.Net.SASL.GSSAPI.ServiceName = "kafka"
		}
		config.Net.SASL.GSSAPI.Username = metadata.Username
		config.Net.SASL.GSSAPI.Realm = metadata.Realm
		config.Net.SASL.GSSAPI.KerberosConfigPath = metadata.kerberosConfigPath
		if metadata.keytabPath != "" {
			config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
			config.Net.SASL.GSSAPI.KeyTabPath = metadata.keytabPath
		} else {
			config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_USER_AUTH
			config.Net.SASL.GSSAPI.Password = metadata.Password
		}
		if metadata.KerberosDisableFAST {
			config.Net.SASL.GSSAPI.DisablePAFXFAST = true
		}
	}

	return config, nil
}

func (s *kafkaScaler) getTopicPartitions() (map[string][]int32, error) {
	var topicsToDescribe = make([]string, 0)

	if s.metadata.Topic == "" {
		listCGOffsetResponse, err := s.admin.ListConsumerGroupOffsets(s.metadata.Group, nil)
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
		topicsToDescribe = []string{s.metadata.Topic}
	}

	topicsMetadata, err := s.admin.DescribeTopics(topicsToDescribe)
	if err != nil {
		return nil, fmt.Errorf("error describing topics: %w", err)
	}
	s.logger.V(1).Info(
		fmt.Sprintf("with topic name %s the list of topic metadata is %v", topicsToDescribe, topicsMetadata),
	)

	if s.metadata.Topic != "" && len(topicsMetadata) != 1 {
		return nil, fmt.Errorf("expected only 1 topic metadata, got %d", len(topicsMetadata))
	}

	topicPartitions := make(map[string][]int32, len(topicsMetadata))
	for _, topicMetadata := range topicsMetadata {
		if topicMetadata.Err > 0 {
			errMsg := fmt.Errorf("error describing topics: %w", topicMetadata.Err)
			s.logger.Error(errMsg, "")
			return nil, errMsg
		}
		var partitions []int32
		for _, p := range topicMetadata.Partitions {
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
	if s.metadata.PartitionLimitation == nil {
		return true
	}
	for _, _pID := range s.metadata.PartitionLimitation {
		if pID == _pID {
			return true
		}
	}
	return false
}

func (s *kafkaScaler) getConsumerOffsets(topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	offsets, err := s.admin.ListConsumerGroupOffsets(s.metadata.Group, topicPartitions)
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
		if s.metadata.OffsetResetPolicy == latest {
			retVal := int64(1)
			if s.metadata.ScaleToZeroOnInvalidOffset {
				retVal = 0
			}
			msg := fmt.Sprintf(
				"invalid offset found for topic %s in group %s and partition %d, probably no offset is committed yet. Returning with lag of %d",
				topic, s.metadata.Group, partitionID, retVal)
			s.logger.V(1).Info(msg)
			return retVal, retVal, nil
		}
		// offsetResetPolicy == earliest
		// For earliest policy, we need latestOffset to return the full lag when scaleToZeroOnInvalidOffset is false
		// But if we can't get latestOffset, we should still respect scaleToZeroOnInvalidOffset
		if s.metadata.ScaleToZeroOnInvalidOffset {
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
	if consumerOffset == invalidOffset && s.metadata.OffsetResetPolicy == earliest {
		return latestOffset, latestOffset, nil
	}

	// This code block tries to prevent KEDA Kafka trigger from scaling the scale target based on erroneous events
	if s.metadata.ExcludePersistentLag {
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
	if s.metadata.Topic != "" {
		metricName = fmt.Sprintf("kafka-%s", s.metadata.Topic)
	} else {
		metricName = fmt.Sprintf("kafka-%s-topics", s.metadata.Group)
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(metricName)),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.LagThreshold),
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

	return []external_metrics.ExternalMetricValue{metric}, totalLagWithPersistent > s.metadata.ActivationLagThreshold, nil
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
	s.logger.V(1).Info(fmt.Sprintf("Kafka scaler: Providing metrics based on totalLag %v, topicPartitions %v, threshold %v", totalLag, len(topicPartitions), s.metadata.LagThreshold))

	if !s.metadata.AllowIdleConsumers || s.metadata.LimitToPartitionsWithLag || s.metadata.EnsureEvenDistributionOfPartitions {
		// don't scale out beyond the number of topicPartitions or partitionsWithLag depending on settings
		upperBound := totalTopicPartitions
		// Ensure that the number of partitions is evenly distributed across the number of consumers
		if s.metadata.EnsureEvenDistributionOfPartitions {
			nextFactor := getNextFactorThatBalancesConsumersToTopicPartitions(totalLag, totalTopicPartitions, s.metadata.LagThreshold)
			s.logger.V(1).Info(fmt.Sprintf("Kafka scaler: Providing metrics to ensure even distribution of partitions on totalLag %v, topicPartitions %v, evenPartitions %v", totalLag, totalTopicPartitions, nextFactor))
			totalLag = nextFactor * s.metadata.LagThreshold
		}
		if s.metadata.LimitToPartitionsWithLag {
			upperBound = partitionsWithLag
		}

		if (totalLag / s.metadata.LagThreshold) > upperBound {
			totalLag = upperBound * s.metadata.LagThreshold
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
