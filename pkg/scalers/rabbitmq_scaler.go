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
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
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
	rabbitModeDeliverGetRate               = "DeliverGetRate"
	rabbitModePublishedToDeliveredRatio    = "PublishedToDeliveredRatio"
	rabbitModeExpectedQueueConsumptionTime = "ExpectedQueueConsumptionTime"
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

	// OAuth2 authentication fields (parsed from authParams via scale resolver)
	OAuthTokenURI       string `keda:"name=oauthTokenURI,    order=authParams, optional"`
	OAuthClientID       string `keda:"name=clientID,         order=authParams, optional"`
	OAuthClientSecret   string `keda:"name=clientSecret,     order=authParams, optional"`
	OAuthScopes         string `keda:"name=scopes,           order=authParams, optional"`
	OAuthEndpointParams string `keda:"name=endpointParams,   order=authParams, optional"`

	// Auth method detection flags (computed, not parsed)
	hasOAuth2           bool
	hasBasicAuth        bool
	hasWorkloadIdentity bool
}

func (r *rabbitMQMetadata) Validate() error {
	if r.Protocol != amqpProtocol && r.Protocol != httpProtocol && r.Protocol != autoProtocol {
		return fmt.Errorf("the protocol has to be either `%s`, `%s`, or `%s` but is `%s`",
			amqpProtocol, httpProtocol, autoProtocol, r.Protocol)
	}

	if r.EnableTLS != rmqTLSEnable && r.EnableTLS != rmqTLSDisable {
		return fmt.Errorf("incorrect value for TLS given: %s", r.EnableTLS)
	}

	certGiven := r.Cert != ""
	keyGiven := r.Key != ""
	if certGiven != keyGiven {
		return fmt.Errorf("both key and certificate must be provided")
	}

	if r.PageSize < 1 {
		return fmt.Errorf("pageSize should be 1 or greater")
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
			return fmt.Errorf("unknown host URL scheme: `%s`", parsedURL.Scheme)
		}
	}

	if r.Protocol == amqpProtocol && r.WorkloadIdentityResource != "" {
		return fmt.Errorf("workload identity is not supported for the AMQP protocol at the moment")
	}

	if err := r.validateOAuth2(); err != nil {
		return err
	}

	if r.UseRegex && r.Protocol != httpProtocol {
		return fmt.Errorf("configure useRegex=true only with HTTP protocol")
	}

	if r.ExcludeUnacknowledged && r.Protocol != httpProtocol {
		return fmt.Errorf("configure excludeUnacknowledged=true only with HTTP protocol")
	}

	if err := r.validateTrigger(); err != nil {
		return err
	}

	return nil
}

func (r *rabbitMQMetadata) validateOAuth2() error {
	// OAuth2 validation - check if any OAuth2 parameter is present
	hasAnyOAuth2Param := r.OAuthTokenURI != "" || r.OAuthClientID != "" || r.OAuthClientSecret != ""
	if !hasAnyOAuth2Param {
		return nil
	}

	// Validate required OAuth2 parameters are present
	if r.OAuthTokenURI == "" || r.OAuthClientID == "" || r.OAuthClientSecret == "" {
		return fmt.Errorf("OAuth2 requires tokenUrl clientId and clientSecret to be configured in TriggerAuthentication")
	}

	// Phase 1: OAuth2 only works with HTTP protocol
	if r.Protocol == amqpProtocol {
		return fmt.Errorf("OAuth2 authentication is only supported with HTTP protocol, not AMQP")
	}

	// OAuth2 is exclusive - don't mix with other auth methods
	if r.hasBasicAuth || r.hasWorkloadIdentity {
		return fmt.Errorf("OAuth2 authentication cannot be combined with username/password authentication or Azure Workload Identity")
	}

	return nil
}

func (r *rabbitMQMetadata) validateTrigger() error {
	modes := map[string][]string{
		"all": {
			rabbitModeQueueLength,
			rabbitModeMessageRate,
			rabbitModeDeliverGetRate,
			rabbitModePublishedToDeliveredRatio,
			rabbitModeExpectedQueueConsumptionTime,
		},
		"httpOnly": {
			rabbitModeMessageRate,
			rabbitModeDeliverGetRate,
			rabbitModePublishedToDeliveredRatio,
			rabbitModeExpectedQueueConsumptionTime,
		},
	}

	// If nothing is specified for the trigger then return the default
	if r.QueueLength == 0 && r.Mode == rabbitModeUnknown && r.Value == 0 {
		r.Mode = rabbitModeQueueLength
		r.Value = defaultRabbitMQQueueLength
		return nil
	}

	if r.QueueLength != 0 && (r.Mode != rabbitModeUnknown || r.Value != 0) {
		return fmt.Errorf("queueLength is deprecated; use %s and %s", rabbitModeTriggerConfigName, rabbitValueTriggerConfigName)
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

	if !slices.Contains(modes["all"], r.Mode) {
		return fmt.Errorf("trigger mode %s must be one of %s, %s, %s, %s or %s", r.Mode,
			rabbitModeQueueLength,
			rabbitModeMessageRate,
			rabbitModeDeliverGetRate,
			rabbitModePublishedToDeliveredRatio,
			rabbitModeExpectedQueueConsumptionTime)
	}

	if slices.Contains(modes["httpOnly"], r.Mode) && r.Protocol != httpProtocol {
		return fmt.Errorf("protocol %s not supported; must be HTTP to use trigger mode %s", r.Protocol, r.Mode)
	}

	if r.Protocol == amqpProtocol && r.Timeout != 0 {
		return fmt.Errorf("AMQP protocol doesn't support custom timeouts: %d", r.Timeout)
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
	PublishDetail    publishDetail    `json:"publish_details"`
	DeliverGetDetail deliverGetDetail `json:"deliver_get_details"`
}

type publishDetail struct {
	Rate float64 `json:"rate"`
}

type deliverGetDetail struct {
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
		return nil, fmt.Errorf("error parsing RabbitMQ metadata: %w", err)
	}

	s.metadata = meta

	timeout := config.GlobalHTTPTimeout
	if s.metadata.Timeout != 0 {
		timeout = s.metadata.Timeout
	}

	// Create HTTP client based on auth method
	if meta.hasOAuth2 {
		// OAuth2 client with automatic token management
		s.httpClient = s.createOAuth2HTTPClient(timeout, meta)
	} else {
		// Standard HTTP client for basic auth or workload identity
		s.httpClient = kedautil.CreateHTTPClient(timeout, meta.UnsafeSsl)
		if meta.EnableTLS == rmqTLSEnable {
			tlsConfig, tlsErr := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.Ca, meta.UnsafeSsl)
			if tlsErr != nil {
				return nil, tlsErr
			}
			s.httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(tlsConfig)
		}
	}

	if meta.Protocol == amqpProtocol {
		// Override vhost if requested.
		host := meta.Host
		if meta.VhostName != "" || (meta.Username != "" && meta.Password != "") {
			hostURI, err := amqp.ParseURI(host)
			if err != nil {
				return nil, fmt.Errorf("error parsing RabbitMQ connection string: %w", err)
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
			return nil, fmt.Errorf("error establishing connection to RabbitMQ: %w", err)
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
		return nil, fmt.Errorf("error parsing RabbitMQ metadata: %w", err)
	}

	if config.PodIdentity.Provider == v1alpha1.PodIdentityProviderAzureWorkload {
		if meta.WorkloadIdentityResource != "" {
			meta.workloadIdentityClientID = config.PodIdentity.GetIdentityID()
			meta.workloadIdentityTenantID = config.PodIdentity.GetIdentityTenantID()
		}
	}

	meta.triggerIndex = config.TriggerIndex
	meta.determineAuthMethod()

	return meta, nil
}

// determineAuthMethod sets boolean flags for auth method detection
func (r *rabbitMQMetadata) determineAuthMethod() {
	r.hasOAuth2 = r.OAuthTokenURI != "" && r.OAuthClientID != "" && r.OAuthClientSecret != ""
	r.hasBasicAuth = r.Username != "" && r.Password != ""
	r.hasWorkloadIdentity = r.WorkloadIdentityResource != ""
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
			s.logger.Error(err, "Error closing RabbitMQ connection")
			return err
		}
	}
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

// createOAuth2HTTPClient creates an HTTP client with OAuth2 client credentials flow
// The client automatically handles token acquisition, caching, and refresh
func (s *rabbitMQScaler) createOAuth2HTTPClient(timeout time.Duration, meta *rabbitMQMetadata) *http.Client {
	config := clientcredentials.Config{
		ClientID:     meta.OAuthClientID,
		ClientSecret: meta.OAuthClientSecret,
		TokenURL:     meta.OAuthTokenURI,
	}

	// Parse scopes from comma-separated string
	if meta.OAuthScopes != "" {
		config.Scopes = strings.Split(meta.OAuthScopes, ",")
	}

	// Parse additional endpoint parameters (URL-encoded string from scale resolver)
	if meta.OAuthEndpointParams != "" {
		params, err := url.ParseQuery(meta.OAuthEndpointParams)
		if err == nil {
			config.EndpointParams = params
		} else {
			s.logger.Error(err, "Failed to parse OAuth2 endpoint parameters, ignoring")
		}
	}

	// Create OAuth2 client with automatic token management
	oauth2Client := config.Client(context.Background())

	// Apply timeout and optional TLS config
	if transport, ok := oauth2Client.Transport.(*oauth2.Transport); ok {
		baseTransport := &http.Transport{
			IdleConnTimeout: timeout,
		}

		// Apply TLS configuration if enabled
		if meta.EnableTLS == rmqTLSEnable {
			tlsConfig, err := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.Ca, meta.UnsafeSsl)
			if err != nil {
				s.logger.Error(err, "Failed to configure TLS for OAuth2 client, proceeding without TLS")
			} else {
				baseTransport.TLSClientConfig = tlsConfig
			}
		}

		transport.Base = baseTransport
	}

	oauth2Client.Timeout = timeout
	return oauth2Client
}

func (s *rabbitMQScaler) getQueueStatus(ctx context.Context) (int64, float64, float64, error) {
	if s.metadata.Protocol == httpProtocol {
		info, err := s.getQueueInfoViaHTTP(ctx)
		if err != nil {
			return -1, -1, -1, err
		}

		if s.metadata.ExcludeUnacknowledged {
			// messages count includes only ready
			return int64(info.MessagesReady), info.MessageStat.PublishDetail.Rate, info.MessageStat.DeliverGetDetail.Rate, nil
		}
		// messages count includes count of ready and unack-ed
		return int64(info.Messages), info.MessageStat.PublishDetail.Rate, info.MessageStat.DeliverGetDetail.Rate, nil
	}

	// QueueDeclarePassive assumes that the queue exists and fails if it doesn't
	items, err := s.channel.QueueDeclarePassive(s.metadata.QueueName, false, false, false, false, amqp.Table{})
	if err != nil {
		return -1, -1, -1, err
	}

	return int64(items.Messages), 0, 0, nil
}

func getJSON(ctx context.Context, s *rabbitMQScaler, url string) (queueInfo, error) {
	var result queueInfo

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return result, err
	}

	// OAuth2: Authorization header automatically added by oauth2.Transport
	// Azure Workload Identity: Manually add bearer token
	// Basic auth: Handled automatically via URL userinfo (see getQueueInfoViaHTTP)
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
	return result, fmt.Errorf("error requesting RabbitMQ API status: %s, response: %s, from: %s", r.Status, body, url)
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
	messages, publishRate, deliverGetRate, err := s.getQueueStatus(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, s.anonymizeRabbitMQError(err)
	}

	var metric external_metrics.ExternalMetricValue
	var isActive bool

	switch s.metadata.Mode {
	case rabbitModeQueueLength:
		metric = GenerateMetricInMili(metricName, float64(messages))
		isActive = float64(messages) > s.metadata.ActivationValue
	case rabbitModeMessageRate:
		metric = GenerateMetricInMili(metricName, publishRate)
		isActive = (publishRate > s.metadata.ActivationValue) || (float64(messages) > s.metadata.ActivationValue)
	case rabbitModeDeliverGetRate:
		metric = GenerateMetricInMili(metricName, deliverGetRate)
		isActive = deliverGetRate > s.metadata.ActivationValue
	case rabbitModePublishedToDeliveredRatio:
		ratio := float64(0)
		if (publishRate > 0) && (deliverGetRate == 0) {
			ratio = float64(s.metadata.ActivationValue)
		} else if (publishRate > 0) && (deliverGetRate > 0) {
			ratio = float64(publishRate / deliverGetRate)
		}
		metric = GenerateMetricInMili(metricName, ratio)
		isActive = (ratio > s.metadata.ActivationValue) || ((publishRate > 0) && (deliverGetRate == 0))
	case rabbitModeExpectedQueueConsumptionTime:
		eta := float64(0)
		if deliverGetRate == 0 {
			eta = float64(s.metadata.ActivationValue)
		} else {
			eta = ((publishRate - deliverGetRate) / deliverGetRate) + (float64(messages) / deliverGetRate)
		}
		metric = GenerateMetricInMili(metricName, eta)
		isActive = (eta > s.metadata.ActivationValue) || (deliverGetRate == 0)
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
			sumMessages, sumReady, sumPublishRate, sumDeliverGetRate := getSum(q)
			queue.Messages = sumMessages
			queue.MessagesReady = sumReady
			queue.MessageStat.PublishDetail.Rate = sumPublishRate
			queue.MessageStat.DeliverGetDetail.Rate = sumDeliverGetRate
		case avgOperation:
			avgMessages, avgReady, avgPublishRate, avgDeliverGetRate := getAverage(q)
			queue.Messages = avgMessages
			queue.MessagesReady = avgReady
			queue.MessageStat.PublishDetail.Rate = avgPublishRate
			queue.MessageStat.DeliverGetDetail.Rate = avgDeliverGetRate
		case maxOperation:
			maxMessages, maxReady, maxPublishRate, maxDeliverGetRate := getMaximum(q)
			queue.Messages = maxMessages
			queue.MessagesReady = maxReady
			queue.MessageStat.PublishDetail.Rate = maxPublishRate
			queue.MessageStat.DeliverGetDetail.Rate = maxDeliverGetRate
		default:
			return queue, fmt.Errorf("operation mode %s must be one of %s, %s, %s", s.metadata.Operation,
				sumOperation, avgOperation, maxOperation)
		}
	} else {
		queue.Messages = 0
		queue.MessagesReady = 0
		queue.MessageStat.PublishDetail.Rate = 0
		queue.MessageStat.DeliverGetDetail.Rate = 0
	}

	return queue, nil
}

func getSum(q []queueInfo) (int, int, float64, float64) {
	var sumMessages int
	var sumMessagesReady int
	var sumPublishRate, sumDeliverGetRate float64

	for _, value := range q {
		sumMessages += value.Messages
		sumMessagesReady += value.MessagesReady
		sumPublishRate += value.MessageStat.PublishDetail.Rate
		sumDeliverGetRate += value.MessageStat.DeliverGetDetail.Rate
	}

	return sumMessages, sumMessagesReady, sumPublishRate, sumDeliverGetRate
}

func getAverage(q []queueInfo) (int, int, float64, float64) {
	sumMessages, sumReady, sumPublishRate, sumDeliverGetRate := getSum(q)
	length := len(q)

	return sumMessages / length, sumReady / length, sumPublishRate / float64(length), sumDeliverGetRate / float64(length)
}

func getMaximum(q []queueInfo) (int, int, float64, float64) {
	var maxMessages int
	var maxReady int
	var maxPublishRate, maxDeliverGetRate float64

	for _, value := range q {
		if value.Messages > maxMessages {
			maxMessages = value.Messages
		}
		if value.MessagesReady > maxReady {
			maxReady = value.MessagesReady
		}
		if value.MessageStat.PublishDetail.Rate > maxPublishRate {
			maxPublishRate = value.MessageStat.PublishDetail.Rate
		}
		if value.MessageStat.DeliverGetDetail.Rate > maxDeliverGetRate {
			maxDeliverGetRate = value.MessageStat.DeliverGetDetail.Rate
		}
	}

	return maxMessages, maxReady, maxPublishRate, maxDeliverGetRate
}

// Mask host for log purposes
func (s *rabbitMQScaler) anonymizeRabbitMQError(err error) error {
	errorMessage := fmt.Sprintf("error inspecting RabbitMQ: %s", err)
	return fmt.Errorf("%s", rabbitMQAnonymizePattern.ReplaceAllString(errorMessage, "user:password@"))
}

// connectionName is used to provide a deterministic AMQP connection name when
// connecting to RabbitMQ
func connectionName(config *scalersconfig.ScalerConfig) string {
	return fmt.Sprintf("keda-%s-%s", config.ScalableObjectNamespace, config.ScalableObjectName)
}
