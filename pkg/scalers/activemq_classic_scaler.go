package scalers

import (
        "context"
        "encoding/json"
        "errors"
        "fmt"
        "net/http"
        "net/url"
        "strconv"
        "strings"

        kedautil "github.com/kedacore/keda/v2/pkg/util"
        v2beta2 "k8s.io/api/autoscaling/v2beta2"
        "k8s.io/apimachinery/pkg/api/resource"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        "k8s.io/apimachinery/pkg/labels"
        "k8s.io/metrics/pkg/apis/external_metrics"
        logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type activemqClassicScaler struct {
        metadata   *activemqClassicMetadata
        httpClient *http.Client
}

//revive:disable:var-naming breaking change on restApiTemplate, wouldn't bring any benefit to users
type activemqClassicMetadata struct {
        managementEndpoint string
        destinationName    string
        brokerName         string
        username           string
        password           string
        restApiTemplate    string
        queueSize          int
        corsHeader         string
}

//revive:enable:var-naming
type activemqClassicMonitoring struct {
        MsgCount  int   `json:"value"`
        Status    int   `json:"status"`
        Timestamp int64 `json:"timestamp"`
}

const (
        activemqClassicMetricType        = "External"
        defaultActivemqClassicQueueSize  = 10
        defaultRestApiTemplate           = "http://<<managementEndpoint>>/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=<<brokerName>>,destinationType=Queue,destinationName=<<destinationName>>/QueueSize"
        defaultActivemqClassicCorsHeader = "http://%s"
)

var activemqClassicLog = logf.Log.WithName("activemq_classic_scaler")

// NewActivemqClassicQueueScaler creates a new activemqClassic queue Scaler
func NewActivemqClassicScaler(config *ScalerConfig) (Scaler, error) {
        // do we need to guarantee this timeout for a specific
        // reason? if not, we can have buildScaler pass in
        // the global client
        httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

        activemqClassicMetadata, err := parseActivemqClassicMetadata(config)
        if err != nil {
                return nil, fmt.Errorf("error parsing activemq classic metadata: %s", err)
        }

        return &activemqClassicScaler{
                metadata:   activemqClassicMetadata,
                httpClient: httpClient,
        }, nil
}

func parseActivemqClassicMetadata(config *ScalerConfig) (*activemqClassicMetadata, error) {
        meta := activemqClassicMetadata{}

        meta.queueSize = defaultActivemqClassicQueueSize

        if val, ok := config.TriggerMetadata["restApiTemplate"]; ok && val != "" {
                meta.restApiTemplate = config.TriggerMetadata["restApiTemplate"]
                var err error
                if meta, err = getAPIParameters(meta); err != nil {
                        return nil, fmt.Errorf("can't parse restApiTemplate : %s ", err)
                }
        } else {
                meta.restApiTemplate = defaultRestApiTemplate
                if config.TriggerMetadata["managementEndpoint"] == "" {
                        return nil, errors.New("no management endpoint given")
                }
                meta.managementEndpoint = config.TriggerMetadata["managementEndpoint"]

                if config.TriggerMetadata["destinationName"] == "" {
                        return nil, errors.New("no destination name given")
                }
                meta.destinationName = config.TriggerMetadata["destinationName"]

                if config.TriggerMetadata["brokerName"] == "" {
                        return nil, errors.New("no broker name given")
                }
                meta.brokerName = config.TriggerMetadata["brokerName"]

        }
        if val, ok := config.TriggerMetadata["corsHeader"]; ok && val != "" {
                meta.corsHeader = config.TriggerMetadata["corsHeader"]
        } else {
                meta.corsHeader = fmt.Sprintf(defaultActivemqClassicCorsHeader, meta.managementEndpoint)
        }

        if val, ok := config.TriggerMetadata["queueSize"]; ok {
                queueSize, err := strconv.Atoi(val)
                if err != nil {
                        return nil, fmt.Errorf("can't parse queueSize: %s", err)
                }

                meta.queueSize = queueSize
        }

        if val, ok := config.AuthParams["username"]; ok && val != "" {
                meta.username = val
        } else if val, ok := config.TriggerMetadata["username"]; ok && val != "" {
                username := val

                if val, ok := config.ResolvedEnv[username]; ok && val != "" {
                        meta.username = val
                } else {
                        meta.username = username
                }
        }

        if meta.username == "" {
                return nil, fmt.Errorf("username cannot be empty")
        }

        if val, ok := config.AuthParams["password"]; ok && val != "" {
                meta.password = val
        } else if val, ok := config.TriggerMetadata["password"]; ok && val != "" {
                password := val

                if val, ok := config.ResolvedEnv[password]; ok && val != "" {
                        meta.password = val
                } else {
                        meta.password = password
                }
        }

        if meta.password == "" {
                return nil, fmt.Errorf("password cannot be empty")
        }
        return &meta, nil
}

// IsActive determines if we need to scale from zero
func (s *activemqClassicScaler) IsActive(ctx context.Context) (bool, error) {
        messages, err := s.getQueueMessageCount()
        if err != nil {
                activemqClassicLog.Error(err, "Unable to access the activemq classic management endpoint", "managementEndpoint", s.metadata.managementEndpoint)
                return false, err
        }

        return messages > 0, nil
}

// getAPIParameters parse restApiTemplate to provide managementEndpoint , brokerName, destinationType, destinationName
func getAPIParameters(meta activemqClassicMetadata) (activemqClassicMetadata, error) {
        u, err := url.ParseRequestURI(meta.restApiTemplate)
        if err != nil {
                return meta, fmt.Errorf("unable to parse the activemq classic restApiTemplate: %s", err)
        }

        meta.managementEndpoint = u.Host
        splitURL := strings.Split(strings.Split(u.Path, ":")[1], "/")[0] // This returns : type=Broker,brokerName=<<brokerName>>,destinationType=Queue,destinationName=<<destinationName>>
        replacer := strings.NewReplacer(",", "&")
        v, err := url.ParseQuery(replacer.Replace(splitURL)) // This returns a map with key: string types and element type [] string. : map[brokerName:[<<brokerName>>] destinationName:[<<destinationName>>] destinationType:[Queue] type:[Broker]]
        if err != nil {
                return meta, fmt.Errorf("unable to parse the activemq classic restApiTemplate: %s", err)
        }

        if len(v["destinationName"][0]) == 0 {
                return meta, errors.New("no destinationName is given")
        }
        meta.destinationName = v["destinationName"][0]

        if len(v["brokerName"][0]) == 0 {
                return meta, fmt.Errorf("no brokerName given: %s", meta.restApiTemplate)
        }
        meta.brokerName = v["brokerName"][0]

        return meta, nil
}

func (s *activemqClassicScaler) getMonitoringEndpoint() string {
        replacer := strings.NewReplacer("<<managementEndpoint>>", s.metadata.managementEndpoint,
                "<<brokerName>>", s.metadata.brokerName,
                "<<destinationName>>", s.metadata.destinationName)

        monitoringEndpoint := replacer.Replace(s.metadata.restApiTemplate)

        return monitoringEndpoint
}

func (s *activemqClassicScaler) getQueueMessageCount() (int, error) {
        var monitoringInfo *activemqClassicMonitoring
        messageCount := 0

        client := s.httpClient
        url := s.getMonitoringEndpoint()

        req, err := http.NewRequest("GET", url, nil)

        req.SetBasicAuth(s.metadata.username, s.metadata.password)
        req.Header.Set("Origin", s.metadata.corsHeader)

        if err != nil {
                return -1, err
        }
        resp, err := client.Do(req)
        if err != nil {
                return -1, err
        }

        defer resp.Body.Close()

        if err := json.NewDecoder(resp.Body).Decode(&monitoringInfo); err != nil {
                return -1, err
        }
        if resp.StatusCode == 200 && monitoringInfo.Status == 200 {
                messageCount = monitoringInfo.MsgCount
        } else {
                return -1, fmt.Errorf("activemq classic management endpoint response error code : %d %d", resp.StatusCode, monitoringInfo.Status)
        }

        activemqClassicLog.V(1).Info(fmt.Sprintf("Activemq classic scaler: Providing metrics based on current queue size %d queue size limit %d", messageCount, s.metadata.queueSize))

        return messageCount, nil
}

func (s *activemqClassicScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
        targetMetricValue := resource.NewQuantity(int64(s.metadata.queueSize), resource.DecimalSI)
        externalMetric := &v2beta2.ExternalMetricSource{
                Metric: v2beta2.MetricIdentifier{
                        Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "activemqClassic", s.metadata.brokerName, s.metadata.destinationName)),
                },
                Target: v2beta2.MetricTarget{
                        Type:         v2beta2.AverageValueMetricType,
                        AverageValue: targetMetricValue,
                },
        }
        metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: activemqClassicMetricType}
        return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *activemqClassicScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
        messages, err := s.getQueueMessageCount()

        if err != nil {
                activemqClassicLog.Error(err, "Unable to access the activemq classic management endpoint", "managementEndpoint", s.metadata.managementEndpoint)
                return []external_metrics.ExternalMetricValue{}, err
        }

        metric := external_metrics.ExternalMetricValue{
                MetricName: metricName,
                Value:      *resource.NewQuantity(int64(messages), resource.DecimalSI),
                Timestamp:  metav1.Now(),
        }

        return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Nothing to close here.
func (s *activemqClassicScaler) Close() error {
        return nil
}
