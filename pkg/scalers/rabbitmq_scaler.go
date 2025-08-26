package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	amqp "github.com/rabbitmq/amqp091-go"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var rabbitMQAnonymizePattern *regexp.Regexp

func init() {
	rabbitMQAnonymizePattern = regexp.MustCompile(`([^ \/:]+):([^\/:]+)\@`)
}

const (
	rabbitQueueLengthMetricName            = "queueLength"
	rabbitModeTriggerConfigName            = "mode"
	rabbitValueTriggerConfigName           = "value"
	rabbitActivationValueTriggerConfigName = "activationValue"
	rabbitModeUnknown                      = "Unknown"
	rabbitModeQueueLength                  = "QueueLength"
	rabbitModeMessageRate                  = "MessageRate"
	defaultRabbitMQQueueLength             = 20
	rabbitMetricType                       = "External"
	rabbitRootVhostPath                    = "/%2F"
	rmqTLSEnable                           = "enable"
	rmqTLSDisable                          = "disable"
)

const (
	httpProtocol    = "http"
	amqpProtocol    = "amqp"
	autoProtocol    = "auto"
	defaultProtocol = autoProtocol
)

const (
	sumOperation     = "sum"
	avgOperation     = "avg"
	maxOperation     = "max"
	defaultOperation = sumOperation
)

type rabbitMQScaler struct {
	metricType v2.MetricTargetType
	metadata   *rabbitMQMetadata
	connection *amqp.Connection
	channel    *amqp.Channel
	httpClient *http.Client
	azureOAuth *azure.ADWorkloadIdentityTokenProvider
	logger     logr.Logger
}

type rabbitMQMetadata struct {
	connectionName string // name used for the AMQP connection
	triggerIndex   int    // scaler index

	QueueName string `keda:"name=queueName,                       order=triggerMetadata"`
	// QueueLength or MessageRate
	Mode string `keda:"name=mode,                                 order=triggerMetadata, optional, default=Unknown"`
	//
	QueueLength float64 `keda:"name=queueLength,                  order=triggerMetadata, optional"`
	// trigger value (queue length or publish/sec. rate)
	Value float64 `keda:"name=value,                              order=triggerMetadata, optional"`
	// activation value
	ActivationValue float64 `keda:"name=activationValue,          order=triggerMetadata, optional"`
	// connection string for either HTTP or AMQP protocol
	Host string `keda:"name=host,                                 order=triggerMetadata;authParams;resolvedEnv"`
	// either http or amqp protocol
	Protocol string `keda:"name=protocol,                         order=triggerMetadata;authParams, default=auto"`
	// override the vhost from the connection info
	VhostName string `keda:"name=vhostName,                       order=triggerMetadata;authParams, optional"`
	// specify if the queueName contains a rexeg
	UseRegex bool `keda:"name=useRegex,                           order=triggerMetadata, optional"`
	// specify if the QueueLength value should exclude Unacknowledged messages (Ready messages only)
	ExcludeUnacknowledged bool `keda:"name=excludeUnacknowledged, order=triggerMetadata, optional"`
	// specify the page size if useRegex is enabled
	PageSize int64 `keda:"name=pageSize,                          order=triggerMetadata, default=100"`
	// specify the operation to apply in case of multiples queues
	Operation string `keda:"name=operation,                       order=triggerMetadata, default=sum"`
	// custom http timeout for a specific trigger
	Timeout time.Duration `keda:"name=timeout,                  order=triggerMetadata, optional"`

	Username string `keda:"name=username, order=authParams;resolvedEnv, optional"`
	Password string `keda:"name=password, order=authParams;resolvedEnv, optional"`

	// TLS
	Ca          string `keda:"name=ca,          order=authParams, optional"`
	Cert        string `keda:"name=cert,        order=authParams, optional"`
	Key         string `keda:"name=key,         order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword, order=authParams, optional"`
	EnableTLS   string `keda:"name=tls,         order=authParams, default=disable"`
	UnsafeSsl   bool   `keda:"name=unsafeSsl,   order=triggerMetadata, optional"`

	// token provider for azure AD
	WorkloadIdentityResource      string `keda:"name=workloadIdentityResource, order=authParams, optional"`
	workloadIdentityClientID      string
	workloadIdentityTenantID      string
	workloadIdentityAuthorityHost string
}

func (r *rabbitMQMetadata) Validate() error {
	if r.Protocol != amqpProtocol && r.Protocol != httpProtocol && r.Protocol != autoProtocol {
		return fmt.Errorf("the protocol has to be either `%s`, `%s`, or `%s` but is `%s`",
			amqpProtocol, httpProtocol, autoProtocol, r.Protocol)
	}

	if r.EnableTLS != rmqTLSEnable && r.EnableTLS != rmqTLSDisable {
		return fmt.Errorf("err incorrect value for TLS given: %s", r.EnableTLS)
	}

	certGiven := r.Cert != ""
	keyGiven := r.Key != ""
	if certGiven != keyGiven {
		return fmt.Errorf("both key and cert must be provided")
	}

	if r.PageSize < 1 {
		return fmt.Errorf("pageSize should be 1 or greater than 1")
	}

	if (r.Username != "" || r.Password != "") && (r.Username == "" || r.Password == "") {
		return fmt.Errorf("username and password must be given together")
	}

	// If the protocol is auto, check the host scheme.
	if r.Protocol == autoProtocol {
		parsedURL, err := url.Parse(r.Host)
		if err != nil {
			return fmt.Errorf("can't parse host to find protocol: %w", err)
		}
		switch parsedURL.Scheme {
		case "amqp", "amqps":
			r.Protocol = amqpProtocol
		case "http", "https":
			r.Protocol = httpProtocol
		default:
			return fmt.Errorf("unknown host URL scheme `%s`", parsedURL.Scheme)
		}
	}

	if r.Protocol == amqpProtocol && r.WorkloadIdentityResource != "" {
		return fmt.Errorf("workload identity is not supported for amqp protocol currently")
	}

	if r.UseRegex && r.Protocol != httpProtocol {
		return fmt.Errorf("configure only useRegex with http protocol")
	}

	if r.ExcludeUnacknowledged && r.Protocol != httpProtocol {
		return fmt.Errorf("configure excludeUnacknowledged=true with http protocol only")
	}

	if err := r.validateTrigger(); err != nil {
		return err
	}

	return nil
}

func (r *rabbitMQMetadata) validateTrigger() error {
	// If nothing is specified for the trigger then return the default
	if r.QueueLength == 0 && r.Mode == rabbitModeUnknown && r.Value == 0 {
		r.Mode = rabbitModeQueueLength
		r.Value = defaultRabbitMQQueueLength
		return nil
	}

	if r.QueueLength != 0 && (r.Mode != rabbitModeUnknown || r.Value != 0) {
		return fmt.Errorf("queueLength is deprecated; configure only %s and %s", rabbitModeTriggerConfigName, rabbitValueTriggerConfigName)
	}

	if r.QueueLength != 0 {
		r.Mode = rabbitModeQueueLength
		r.Value = r.QueueLength

		return nil
	}

	if r.Mode == rabbitModeUnknown {
		return fmt.Errorf("%s must be specified", rabbitModeTriggerConfigName)
	}

	if r.Value == 0 {
		return fmt.Errorf("%s must be specified", rabbitValueTriggerConfigName)
	}

	if r.Mode != rabbitModeQueueLength && r.Mode != rabbitModeMessageRate {
		return fmt.Errorf("trigger mode %s must be one of %s, %s", r.Mode, rabbitModeQueueLength, rabbitModeMessageRate)
	}

	if r.Mode == rabbitModeMessageRate && r.Protocol != httpProtocol {
		return fmt.Errorf("protocol %s not supported; must be http to use mode %s", r.Protocol, rabbitModeMessageRate)
	}

	if r.Protocol == amqpProtocol && r.Timeout != 0 {
		return fmt.Errorf("amqp protocol doesn't support custom timeouts: %d", r.Timeout)
	}

	return nil
}

type queueInfo struct {
	Messages               int         `json:"messages"`
	MessagesReady          int         `json:"messages_ready"`
	MessagesUnacknowledged int         `json:"messages_unacknowledged"`
	MessageStat            messageStat `json:"message_stats"`
	Name                   string      `json:"name"`
}

type regexQueueInfo struct {
	Queues     []queueInfo `json:"items"`
	TotalPages int         `json:"page_count"`
}

type messageStat struct {
	PublishDetail publishDetail `json:"publish_details"`
}

type publishDetail struct {
	Rate float64 `json:"rate"`
}

// NewRabbitMQScaler creates a new rabbitMQ scaler
func NewRabbitMQScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	s := &rabbitMQScaler{}

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}
	s.metricType = metricType

	s.logger = InitializeLogger(config, "rabbitmq_scaler")

	meta, err := parseRabbitMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing rabbitmq metadata: %w", err)
	}

	s.metadata = meta

	timeout := config.GlobalHTTPTimeout
	if s.metadata.Timeout != 0 {
		timeout = s.metadata.Timeout
	}

	s.httpClient = kedautil.CreateHTTPClient(timeout, meta.UnsafeSsl)
	if meta.EnableTLS == rmqTLSEnable {
		tlsConfig, tlsErr := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.Ca, meta.UnsafeSsl)
		if tlsErr != nil {
			return nil, tlsErr
		}
		s.httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(tlsConfig)
	}

	if meta.Protocol == amqpProtocol {
		// Override vhost if requested.
		host := meta.Host
		if meta.VhostName != "" || (meta.Username != "" && meta.Password != "") {
			hostURI, err := amqp.ParseURI(host)
			if err != nil {
				return nil, fmt.Errorf("error parsing rabbitmq connection string: %w", err)
			}
			if meta.VhostName != "" {
				hostURI.Vhost = meta.VhostName
			}

			if meta.Username != "" && meta.Password != "" {
				hostURI.Username = meta.Username
				hostURI.Password = meta.Password
			}

			host = hostURI.String()
		}

		conn, ch, err := getConnectionAndChannel(host, meta)
		if err != nil {
			return nil, fmt.Errorf("error establishing rabbitmq connection: %w", err)
		}
		s.connection = conn
		s.channel = ch
	}

	return s, nil
}

func parseRabbitMQMetadata(config *scalersconfig.ScalerConfig) (*rabbitMQMetadata, error) {
	meta := &rabbitMQMetadata{
		connectionName: connectionName(config),
	}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing rabbitmq metadata: %w", err)
	}

	if config.PodIdentity.Provider == v1alpha1.PodIdentityProviderAzureWorkload {
		if meta.WorkloadIdentityResource != "" {
			meta.workloadIdentityClientID = config.PodIdentity.GetIdentityID()
			meta.workloadIdentityTenantID = config.PodIdentity.GetIdentityTenantID()
		}
	}

	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

// getConnectionAndChannel returns an amqp connection. If enableTLS is true tls connection is made using
// the given ceClient cert, ceClient key,and CA certificate. If clientKeyPassword is not empty the provided password will be used to
// decrypt the given key. If enableTLS is disabled then amqp connection will be created without tls.
func getConnectionAndChannel(host string, meta *rabbitMQMetadata) (*amqp.Connection, *amqp.Channel, error) {
	amqpConfig := amqp.Config{
		Properties: amqp.Table{
			"connection_name": meta.connectionName,
		},
	}

	if meta.EnableTLS == rmqTLSEnable {
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.Ca, meta.UnsafeSsl)
		if err != nil {
			return nil, nil, err
		}

		amqpConfig.TLSClientConfig = tlsConfig
	}

	conn, err := amqp.DialConfig(host, amqpConfig)
	if err != nil {
		return nil, nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}

	return conn, channel, nil
}

// Close disposes of RabbitMQ connections
func (s *rabbitMQScaler) Close(context.Context) error {
	if s.connection != nil {
		err := s.connection.Close()
		if err != nil {
			s.logger.Error(err, "Error closing rabbitmq connection")
			return err
		}
	}
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *rabbitMQScaler) getQueueStatus(ctx context.Context) (int64, float64, error) {
	if s.metadata.Protocol == httpProtocol {
		info, err := s.getQueueInfoViaHTTP(ctx)
		if err != nil {
			return -1, -1, err
		}

		if s.metadata.ExcludeUnacknowledged {
			// messages count includes only ready
			return int64(info.MessagesReady), info.MessageStat.PublishDetail.Rate, nil
		}
		// messages count includes count of ready and unack-ed
		return int64(info.Messages), info.MessageStat.PublishDetail.Rate, nil
	}

	// QueueDeclarePassive assumes that the queue exists and fails if it doesn't
	items, err := s.channel.QueueDeclarePassive(s.metadata.QueueName, false, false, false, false, amqp.Table{})
	if err != nil {
		return -1, -1, err
	}

	return int64(items.Messages), 0, nil
}

func getJSON(ctx context.Context, s *rabbitMQScaler, url string) (queueInfo, error) {
	var result queueInfo

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return result, err
	}

	if s.metadata.WorkloadIdentityResource != "" {
		if s.azureOAuth == nil {
			s.azureOAuth = azure.NewAzureADWorkloadIdentityTokenProvider(ctx, s.metadata.workloadIdentityClientID, s.metadata.workloadIdentityTenantID, s.metadata.workloadIdentityAuthorityHost, s.metadata.WorkloadIdentityResource)
		}

		err = s.azureOAuth.Refresh()
		if err != nil {
			return result, err
		}

		request.Header.Set("Authorization", "Bearer "+s.azureOAuth.OAuthToken())
	}

	r, err := s.httpClient.Do(request)
	if err != nil {
		return result, err
	}

	defer r.Body.Close()

	if r.StatusCode == 200 {
		if s.metadata.UseRegex {
			var queues regexQueueInfo
			err = json.NewDecoder(r.Body).Decode(&queues)
			if err != nil {
				return queueInfo{}, err
			}
			if queues.TotalPages > 1 {
				return queueInfo{}, fmt.Errorf("regex matches more queues than can be recovered at once")
			}
			result, err := getComposedQueue(s, queues.Queues)
			return result, err
		}

		err = json.NewDecoder(r.Body).Decode(&result)
		return result, err
	}

	body, _ := io.ReadAll(r.Body)
	return result, fmt.Errorf("error requesting rabbitMQ API status: %s, response: %s, from: %s", r.Status, body, url)
}

func getVhostAndPathFromURL(rawPath, vhostName string) (resolvedVhostPath, resolvedPath string) {
	pathParts := strings.Split(rawPath, "/")
	resolvedVhostPath = "/" + pathParts[len(pathParts)-1]
	resolvedPath = path.Join(pathParts[:len(pathParts)-1]...)

	if len(resolvedPath) > 0 {
		resolvedPath = "/" + resolvedPath
	}
	if vhostName != "" {
		resolvedVhostPath = "/" + url.QueryEscape(vhostName)
	}
	if resolvedVhostPath == "" || resolvedVhostPath == "/" {
		resolvedVhostPath = rabbitRootVhostPath
	}

	return
}

func (s *rabbitMQScaler) getQueueInfoViaHTTP(ctx context.Context) (*queueInfo, error) {
	parsedURL, err := url.Parse(s.metadata.Host)

	if err != nil {
		return nil, err
	}

	path := parsedURL.RawPath
	if path == "" {
		path = parsedURL.Path
	}

	vhost, subpaths := getVhostAndPathFromURL(path, s.metadata.VhostName)
	parsedURL.Path = subpaths

	if s.metadata.Username != "" && s.metadata.Password != "" {
		parsedURL.User = url.UserPassword(s.metadata.Username, s.metadata.Password)
	}

	var getQueueInfoManagementURI string
	if s.metadata.UseRegex {
		getQueueInfoManagementURI = fmt.Sprintf("%s/api/queues%s?page=1&use_regex=true&pagination=false&name=%s&page_size=%d", parsedURL.String(), vhost, url.QueryEscape(s.metadata.QueueName), s.metadata.PageSize)
	} else {
		getQueueInfoManagementURI = fmt.Sprintf("%s/api/queues%s/%s", parsedURL.String(), vhost, url.QueryEscape(s.metadata.QueueName))
	}

	var info queueInfo
	info, err = getJSON(ctx, s, getQueueInfoManagementURI)

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *rabbitMQScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("rabbitmq-%s", url.QueryEscape(s.metadata.QueueName)))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Value),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: rabbitMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *rabbitMQScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	messages, publishRate, err := s.getQueueStatus(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, s.anonymizeRabbitMQError(err)
	}

	var metric external_metrics.ExternalMetricValue
	var isActive bool
	if s.metadata.Mode == rabbitModeQueueLength {
		metric = GenerateMetricInMili(metricName, float64(messages))
		isActive = float64(messages) > s.metadata.ActivationValue
	} else {
		metric = GenerateMetricInMili(metricName, publishRate)
		isActive = publishRate > s.metadata.ActivationValue || float64(messages) > s.metadata.ActivationValue
	}

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

func getComposedQueue(s *rabbitMQScaler, q []queueInfo) (queueInfo, error) {
	var queue = queueInfo{}
	queue.Name = "composed-queue"
	queue.MessagesUnacknowledged = 0
	if len(q) > 0 {
		switch s.metadata.Operation {
		case sumOperation:
			sumMessages, sumReady, sumRate := getSum(q)
			queue.Messages = sumMessages
			queue.MessagesReady = sumReady
			queue.MessageStat.PublishDetail.Rate = sumRate
		case avgOperation:
			avgMessages, avgReady, avgRate := getAverage(q)
			queue.Messages = avgMessages
			queue.MessagesReady = avgReady
			queue.MessageStat.PublishDetail.Rate = avgRate
		case maxOperation:
			maxMessages, maxReady, maxRate := getMaximum(q)
			queue.Messages = maxMessages
			queue.MessagesReady = maxReady
			queue.MessageStat.PublishDetail.Rate = maxRate
		default:
			return queue, fmt.Errorf("operation mode %s must be one of %s, %s, %s", s.metadata.Operation, sumOperation, avgOperation, maxOperation)
		}
	} else {
		queue.Messages = 0
		queue.MessageStat.PublishDetail.Rate = 0
	}

	return queue, nil
}

func getSum(q []queueInfo) (int, int, float64) {
	var sumMessages int
	var sumMessagesReady int
	var sumRate float64
	for _, value := range q {
		sumMessages += value.Messages
		sumMessagesReady += value.MessagesReady
		sumRate += value.MessageStat.PublishDetail.Rate
	}
	return sumMessages, sumMessagesReady, sumRate
}

func getAverage(q []queueInfo) (int, int, float64) {
	sumMessages, sumReady, sumRate := getSum(q)
	length := len(q)
	return sumMessages / length, sumReady / length, sumRate / float64(length)
}

func getMaximum(q []queueInfo) (int, int, float64) {
	var maxMessages int
	var maxReady int
	var maxRate float64
	for _, value := range q {
		if value.Messages > maxMessages {
			maxMessages = value.Messages
		}
		if value.MessagesReady > maxReady {
			maxReady = value.MessagesReady
		}
		if value.MessageStat.PublishDetail.Rate > maxRate {
			maxRate = value.MessageStat.PublishDetail.Rate
		}
	}
	return maxMessages, maxReady, maxRate
}

// Mask host for log purposes
func (s *rabbitMQScaler) anonymizeRabbitMQError(err error) error {
	errorMessage := fmt.Sprintf("error inspecting rabbitMQ: %s", err)
	return fmt.Errorf("%s", rabbitMQAnonymizePattern.ReplaceAllString(errorMessage, "user:password@"))
}

// connectionName is used to provide a deterministic AMQP connection name when
// connecting to RabbitMQ
func connectionName(config *scalersconfig.ScalerConfig) string {
	return fmt.Sprintf("keda-%s-%s", config.ScalableObjectNamespace, config.ScalableObjectName)
}
