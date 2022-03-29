package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/streadway/amqp"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var rabbitMQAnonymizePattern *regexp.Regexp

func init() {
	rabbitMQAnonymizePattern = regexp.MustCompile(`([^ \/:]+):([^\/:]+)\@`)
}

const (
	rabbitQueueLengthMetricName  = "queueLength"
	rabbitModeTriggerConfigName  = "mode"
	rabbitValueTriggerConfigName = "value"
	rabbitModeQueueLength        = "QueueLength"
	rabbitModeMessageRate        = "MessageRate"
	defaultRabbitMQQueueLength   = 20
	rabbitMetricType             = "External"
	rabbitRootVhostPath          = "/%2F"
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
	metricType v2beta2.MetricTargetType
	metadata   *rabbitMQMetadata
	connection *amqp.Connection
	channel    *amqp.Channel
	httpClient *http.Client
}

type rabbitMQMetadata struct {
	queueName   string
	mode        string        // QueueLength or MessageRate
	value       int64         // trigger value (queue length or publish/sec. rate)
	host        string        // connection string for either HTTP or AMQP protocol
	protocol    string        // either http or amqp protocol
	vhostName   *string       // override the vhost from the connection info
	useRegex    bool          // specify if the queueName contains a rexeg
	pageSize    int64         // specify the page size if useRegex is enabled
	operation   string        // specify the operation to apply in case of multiples queues
	metricName  string        // custom metric name for trigger
	timeout     time.Duration // custom http timeout for a specific trigger
	scalerIndex int           // scaler index
}

type queueInfo struct {
	Messages               int         `json:"messages"`
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

var rabbitmqLog = logf.Log.WithName("rabbitmq_scaler")

// NewRabbitMQScaler creates a new rabbitMQ scaler
func NewRabbitMQScaler(config *ScalerConfig) (Scaler, error) {
	s := &rabbitMQScaler{}

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}
	s.metricType = metricType

	meta, err := parseRabbitMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing rabbitmq metadata: %s", err)
	}
	s.metadata = meta
	s.httpClient = kedautil.CreateHTTPClient(meta.timeout, false)

	if meta.protocol == amqpProtocol {
		// Override vhost if requested.
		host := meta.host
		if meta.vhostName != nil {
			hostURI, err := amqp.ParseURI(host)
			if err != nil {
				return nil, fmt.Errorf("error parsing rabbitmq connection string: %s", err)
			}
			hostURI.Vhost = *meta.vhostName
			host = hostURI.String()
		}

		conn, ch, err := getConnectionAndChannel(host)
		if err != nil {
			return nil, fmt.Errorf("error establishing rabbitmq connection: %s", err)
		}
		s.connection = conn
		s.channel = ch
	}

	return s, nil
}

func parseRabbitMQMetadata(config *ScalerConfig) (*rabbitMQMetadata, error) {
	meta := rabbitMQMetadata{}

	// Resolve protocol type
	meta.protocol = defaultProtocol
	if val, ok := config.AuthParams["protocol"]; ok {
		meta.protocol = val
	}
	if val, ok := config.TriggerMetadata["protocol"]; ok {
		meta.protocol = val
	}
	if meta.protocol != amqpProtocol && meta.protocol != httpProtocol && meta.protocol != autoProtocol {
		return nil, fmt.Errorf("the protocol has to be either `%s`, `%s`, or `%s` but is `%s`", amqpProtocol, httpProtocol, autoProtocol, meta.protocol)
	}

	// Resolve host value
	switch {
	case config.AuthParams["host"] != "":
		meta.host = config.AuthParams["host"]
	case config.TriggerMetadata["host"] != "":
		meta.host = config.TriggerMetadata["host"]
	case config.TriggerMetadata["hostFromEnv"] != "":
		meta.host = config.ResolvedEnv[config.TriggerMetadata["hostFromEnv"]]
	default:
		return nil, fmt.Errorf("no host setting given")
	}

	// If the protocol is auto, check the host scheme.
	if meta.protocol == autoProtocol {
		parsedURL, err := url.Parse(meta.host)
		if err != nil {
			return nil, fmt.Errorf("can't parse host to find protocol: %s", err)
		}
		switch parsedURL.Scheme {
		case "amqp", "amqps":
			meta.protocol = amqpProtocol
		case "http", "https":
			meta.protocol = httpProtocol
		default:
			return nil, fmt.Errorf("unknown host URL scheme `%s`", parsedURL.Scheme)
		}
	}

	// Resolve queueName
	if val, ok := config.TriggerMetadata["queueName"]; ok {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf("no queue name given")
	}

	// Resolve vhostName
	if val, ok := config.TriggerMetadata["vhostName"]; ok {
		meta.vhostName = &val
	}

	// Resolve useRegex
	if val, ok := config.TriggerMetadata["useRegex"]; ok {
		useRegex, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("useRegex has invalid value")
		}
		meta.useRegex = useRegex
	}

	// Resolve pageSize
	if val, ok := config.TriggerMetadata["pageSize"]; ok {
		pageSize, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("pageSize has invalid value")
		}
		meta.pageSize = pageSize
		if meta.pageSize < 1 {
			return nil, fmt.Errorf("pageSize should be 1 or greater than 1")
		}
	} else {
		meta.pageSize = 100
	}

	// Resolve operation
	meta.operation = defaultOperation
	if val, ok := config.TriggerMetadata["operation"]; ok {
		meta.operation = val
	}

	if meta.useRegex && meta.protocol == amqpProtocol {
		return nil, fmt.Errorf("configure only useRegex with http protocol")
	}

	_, err := parseTrigger(&meta, config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse trigger: %s", err)
	}

	// Resolve metricName
	if val, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("rabbitmq-%s", url.QueryEscape(val)))
	} else {
		meta.metricName = kedautil.NormalizeString(fmt.Sprintf("rabbitmq-%s", url.QueryEscape(meta.queueName)))
	}

	// Resolve timeout
	if val, ok := config.TriggerMetadata["timeout"]; ok {
		timeoutMS, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("unable to parse timeout: %s", err)
		}
		if meta.protocol == amqpProtocol {
			return nil, fmt.Errorf("amqp protocol doesn't support custom timeouts: %s", err)
		}
		if timeoutMS <= 0 {
			return nil, fmt.Errorf("timeout must be greater than 0: %s", err)
		}
		meta.timeout = time.Duration(timeoutMS) * time.Millisecond
	} else {
		meta.timeout = config.GlobalHTTPTimeout
	}

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func parseTrigger(meta *rabbitMQMetadata, config *ScalerConfig) (*rabbitMQMetadata, error) {
	deprecatedQueueLengthValue, deprecatedQueueLengthPresent := config.TriggerMetadata[rabbitQueueLengthMetricName]
	mode, modePresent := config.TriggerMetadata[rabbitModeTriggerConfigName]
	value, valuePresent := config.TriggerMetadata[rabbitValueTriggerConfigName]

	// Initialize to default trigger settings
	meta.mode = rabbitModeQueueLength
	meta.value = defaultRabbitMQQueueLength

	// If nothing is specified for the trigger then return the default
	if !deprecatedQueueLengthPresent && !modePresent && !valuePresent {
		return meta, nil
	}

	// Only allow one of `queueLength` or `mode`/`value`
	if deprecatedQueueLengthPresent && (modePresent || valuePresent) {
		return nil, fmt.Errorf("queueLength is deprecated; configure only %s and %s", rabbitModeTriggerConfigName, rabbitValueTriggerConfigName)
	}

	// Parse deprecated `queueLength` value
	if deprecatedQueueLengthPresent {
		queueLength, err := strconv.ParseInt(deprecatedQueueLengthValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse %s: %s", rabbitQueueLengthMetricName, err)
		}
		meta.mode = rabbitModeQueueLength
		meta.value = queueLength

		return meta, nil
	}

	if !modePresent {
		return nil, fmt.Errorf("%s must be specified", rabbitModeTriggerConfigName)
	}
	if !valuePresent {
		return nil, fmt.Errorf("%s must be specified", rabbitValueTriggerConfigName)
	}

	// Resolve trigger mode
	switch mode {
	case rabbitModeQueueLength:
		meta.mode = rabbitModeQueueLength
	case rabbitModeMessageRate:
		meta.mode = rabbitModeMessageRate
	default:
		return nil, fmt.Errorf("trigger mode %s must be one of %s, %s", mode, rabbitModeQueueLength, rabbitModeMessageRate)
	}
	triggerValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("can't parse %s: %s", rabbitValueTriggerConfigName, err)
	}
	meta.value = triggerValue

	if meta.mode == rabbitModeMessageRate && meta.protocol != httpProtocol {
		return nil, fmt.Errorf("protocol %s not supported; must be http to use mode %s", meta.protocol, rabbitModeMessageRate)
	}

	return meta, nil
}

func getConnectionAndChannel(host string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(host)
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
			rabbitmqLog.Error(err, "Error closing rabbitmq connection")
			return err
		}
	}
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *rabbitMQScaler) IsActive(ctx context.Context) (bool, error) {
	messages, publishRate, err := s.getQueueStatus()
	if err != nil {
		return false, s.anonimizeRabbitMQError(err)
	}

	if s.metadata.mode == rabbitModeQueueLength {
		return messages > 0, nil
	}
	return publishRate > 0 || messages > 0, nil
}

func (s *rabbitMQScaler) getQueueStatus() (int64, float64, error) {
	if s.metadata.protocol == httpProtocol {
		info, err := s.getQueueInfoViaHTTP()
		if err != nil {
			return -1, -1, err
		}

		// messages count includes count of ready and unack-ed
		return int64(info.Messages), info.MessageStat.PublishDetail.Rate, nil
	}

	items, err := s.channel.QueueInspect(s.metadata.queueName)
	if err != nil {
		return -1, -1, err
	}

	return int64(items.Messages), 0, nil
}

func getJSON(s *rabbitMQScaler, url string) (queueInfo, error) {
	var result queueInfo
	r, err := s.httpClient.Get(url)
	if err != nil {
		return result, err
	}
	defer r.Body.Close()

	if r.StatusCode == 200 {
		if s.metadata.useRegex {
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

	body, _ := ioutil.ReadAll(r.Body)
	return result, fmt.Errorf("error requesting rabbitMQ API status: %s, response: %s, from: %s", r.Status, body, url)
}

func (s *rabbitMQScaler) getQueueInfoViaHTTP() (*queueInfo, error) {
	parsedURL, err := url.Parse(s.metadata.host)

	if err != nil {
		return nil, err
	}

	// Extract vhost from URL's path.
	vhost := parsedURL.Path

	// If the URL's path only contains a slash, it represents the trailing slash and
	// must be ignored because it may cause confusion with the '/' vhost.
	if vhost == "/" {
		vhost = ""
	}

	// Override vhost if requested.
	if s.metadata.vhostName != nil {
		// If the desired vhost is "All" vhosts, no path is necessary
		if *s.metadata.vhostName == "" {
			vhost = ""
		} else {
			vhost = "/" + url.QueryEscape(*s.metadata.vhostName)
		}
	}

	// Encode the '/' vhost if necessary.
	if vhost == "//" {
		vhost = rabbitRootVhostPath
	}

	// Clear URL path to get the correct host.
	parsedURL.Path = ""

	var getQueueInfoManagementURI string
	if s.metadata.useRegex {
		getQueueInfoManagementURI = fmt.Sprintf("%s/api/queues%s?page=1&use_regex=true&pagination=false&name=%s&page_size=%d", parsedURL.String(), vhost, url.QueryEscape(s.metadata.queueName), s.metadata.pageSize)
	} else {
		getQueueInfoManagementURI = fmt.Sprintf("%s/api/queues%s/%s", parsedURL.String(), vhost, url.QueryEscape(s.metadata.queueName))
	}

	var info queueInfo
	info, err = getJSON(s, getQueueInfoManagementURI)

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *rabbitMQScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, s.metadata.metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.value),
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: rabbitMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *rabbitMQScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	messages, publishRate, err := s.getQueueStatus()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, s.anonimizeRabbitMQError(err)
	}

	var metricValue resource.Quantity
	if s.metadata.mode == rabbitModeQueueLength {
		metricValue = *resource.NewQuantity(messages, resource.DecimalSI)
	} else {
		metricValue = *resource.NewMilliQuantity(int64(publishRate*1000), resource.DecimalSI)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      metricValue,
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func getComposedQueue(s *rabbitMQScaler, q []queueInfo) (queueInfo, error) {
	var queue = queueInfo{}
	queue.Name = "composed-queue"
	queue.MessagesUnacknowledged = 0
	if len(q) > 0 {
		switch s.metadata.operation {
		case sumOperation:
			sumMessages, sumRate := getSum(q)
			queue.Messages = sumMessages
			queue.MessageStat.PublishDetail.Rate = sumRate
		case avgOperation:
			avgMessages, avgRate := getAverage(q)
			queue.Messages = avgMessages
			queue.MessageStat.PublishDetail.Rate = avgRate
		case maxOperation:
			maxMessages, maxRate := getMaximum(q)
			queue.Messages = maxMessages
			queue.MessageStat.PublishDetail.Rate = maxRate
		default:
			return queue, fmt.Errorf("operation mode %s must be one of %s, %s, %s", s.metadata.operation, sumOperation, avgOperation, maxOperation)
		}
	} else {
		queue.Messages = 0
		queue.MessageStat.PublishDetail.Rate = 0
	}

	return queue, nil
}

func getSum(q []queueInfo) (int, float64) {
	var sumMessages int
	var sumRate float64
	for _, value := range q {
		sumMessages += value.Messages
		sumRate += value.MessageStat.PublishDetail.Rate
	}
	return sumMessages, sumRate
}

func getAverage(q []queueInfo) (int, float64) {
	sumMessages, sumRate := getSum(q)
	len := len(q)
	return sumMessages / len, sumRate / float64(len)
}

func getMaximum(q []queueInfo) (int, float64) {
	var maxMessages int
	var maxRate float64
	for _, value := range q {
		if value.Messages > maxMessages {
			maxMessages = value.Messages
		}
		if value.MessageStat.PublishDetail.Rate > maxRate {
			maxRate = value.MessageStat.PublishDetail.Rate
		}
	}
	return maxMessages, maxRate
}

// Mask host for log purposes
func (s *rabbitMQScaler) anonimizeRabbitMQError(err error) error {
	errorMessage := fmt.Sprintf("error inspecting rabbitMQ: %s", err)
	return fmt.Errorf(rabbitMQAnonymizePattern.ReplaceAllString(errorMessage, "user:password@"))
}
