package scalers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"io/ioutil"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"
)

type pulsarScaler struct {
	metadata pulsarMetadata
	client   *http.Client
}

type pulsarMetadata struct {
	statsURL            string
	tenant              string
	namespace           string
	topic               string
	subscription        string
	msgBacklogThreshold int64

	// TLS
	enableTLS bool
	cert      string
	key       string
	ca        string
}

const (
	msgBacklogMetricName       = "msgBacklog"
	pulsarMetricType           = "External"
	defaultMsgBacklogThreshold = 10
)

var pulsarLog = logf.Log.WithName("pulsar_scaler")

type pulsarSubscription struct {
	Msgrateout                       float64       `json:"msgRateOut"`
	Msgthroughputout                 float64       `json:"msgThroughputOut"`
	Bytesoutcounter                  int           `json:"bytesOutCounter"`
	Msgoutcounter                    int           `json:"msgOutCounter"`
	Msgrateredeliver                 float64       `json:"msgRateRedeliver"`
	Chuckedmessagerate               int           `json:"chuckedMessageRate"`
	Msgbacklog                       int           `json:"msgBacklog"`
	Msgbacklognodelayed              int           `json:"msgBacklogNoDelayed"`
	Blockedsubscriptiononunackedmsgs bool          `json:"blockedSubscriptionOnUnackedMsgs"`
	Msgdelayed                       int           `json:"msgDelayed"`
	Unackedmessages                  int           `json:"unackedMessages"`
	Type                             string        `json:"type"`
	Msgrateexpired                   float64       `json:"msgRateExpired"`
	Lastexpiretimestamp              int           `json:"lastExpireTimestamp"`
	Lastconsumedflowtimestamp        int64         `json:"lastConsumedFlowTimestamp"`
	Lastconsumedtimestamp            int           `json:"lastConsumedTimestamp"`
	Lastackedtimestamp               int           `json:"lastAckedTimestamp"`
	Consumers                        []interface{} `json:"consumers"`
	Isdurable                        bool          `json:"isDurable"`
	Isreplicated                     bool          `json:"isReplicated"`
	Consumersaftermarkdeleteposition struct {
	} `json:"consumersAfterMarkDeletePosition"`
}

type pulsarStats struct {
	Msgratein         float64                       `json:"msgRateIn"`
	Msgthroughputin   float64                       `json:"msgThroughputIn"`
	Msgrateout        float64                       `json:"msgRateOut"`
	Msgthroughputout  float64                       `json:"msgThroughputOut"`
	Bytesincounter    int                           `json:"bytesInCounter"`
	Msgincounter      int                           `json:"msgInCounter"`
	Bytesoutcounter   int                           `json:"bytesOutCounter"`
	Msgoutcounter     int                           `json:"msgOutCounter"`
	Averagemsgsize    float64                       `json:"averageMsgSize"`
	Msgchunkpublished bool                          `json:"msgChunkPublished"`
	Storagesize       int                           `json:"storageSize"`
	Backlogsize       int                           `json:"backlogSize"`
	Publishers        []interface{}                 `json:"publishers"`
	Subscriptions     map[string]pulsarSubscription `json:"subscriptions"`
	Replication       struct {
	} `json:"replication"`
	Deduplicationstatus string `json:"deduplicationStatus"`
}

// NewPulsarScaler creates a new PulsarScaler
func NewPulsarScaler(config *ScalerConfig) (Scaler, error) {
	pulsarMetadata, err := parsePulsarMetadata(config)
	if err != nil {
		pulsarLog.Error(err, "error parsing pulsar metadata")
		return nil, fmt.Errorf("error parsing pulsar metadata: %s", err)
	}

	client := &http.Client{}

	// If TLS Authentication enabled, do
	if pulsarMetadata.enableTLS {
		// Load client certificate (PEM format)
		cert, err := tls.LoadX509KeyPair(pulsarMetadata.cert, pulsarMetadata.key)
		if err != nil {
			pulsarLog.Error(err, "unable to load cert or key file")
			return nil, fmt.Errorf("unable to load cert or key file: %s", err)
		}

		// Load CA certificate (PEM format)
		caCert, err := ioutil.ReadFile(pulsarMetadata.ca)
		if err != nil {
			pulsarLog.Error(err, "unable to load ca cert file")
			return nil, fmt.Errorf("unable to load ca cert file: %s", err)
		}

		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		// Append our cert to the system pool
		if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
			pulsarLog.Info("No certs appended, using system certs only")
		}
		// Setup HTTPS client
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			Certificates:       []tls.Certificate{cert},
			RootCAs:            rootCAs,
		}
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		client = &http.Client{Transport: transport}
	}

	return &pulsarScaler{
		client:   client,
		metadata: pulsarMetadata,
	}, nil
}

func parsePulsarMetadata(config *ScalerConfig) (pulsarMetadata, error) {
	meta := pulsarMetadata{}
	switch {
	case config.TriggerMetadata["statsURLFromEnv"] != "":
		meta.statsURL = config.ResolvedEnv[config.TriggerMetadata["statsURLFromEnv"]]
	case config.TriggerMetadata["statsURL"] != "":
		meta.statsURL = config.TriggerMetadata["statsURL"]
	default:
		return meta, errors.New("no statsURL given")
	}

	switch {
	case config.TriggerMetadata["tenantFromEnv"] != "":
		meta.tenant = config.ResolvedEnv[config.TriggerMetadata["tenantFromEnv"]]
	case config.TriggerMetadata["tenant"] != "":
		meta.tenant = config.TriggerMetadata["tenant"]
	default:
		return meta, errors.New("no tenant given")
	}

	switch {
	case config.TriggerMetadata["namespaceFromEnv"] != "":
		meta.namespace = config.ResolvedEnv[config.TriggerMetadata["namespaceFromEnv"]]
	case config.TriggerMetadata["namespace"] != "":
		meta.namespace = config.TriggerMetadata["namespace"]
	default:
		return meta, errors.New("no namespace given")
	}

	switch {
	case config.TriggerMetadata["topicFromEnv"] != "":
		meta.topic = config.ResolvedEnv[config.TriggerMetadata["topicFromEnv"]]
	case config.TriggerMetadata["topic"] != "":
		meta.topic = config.TriggerMetadata["topic"]
	default:
		return meta, errors.New("no topic given")
	}

	switch {
	case config.TriggerMetadata["subscriptionFromEnv"] != "":
		meta.subscription = config.ResolvedEnv[config.TriggerMetadata["subscriptionFromEnv"]]
	case config.TriggerMetadata["subscription"] != "":
		meta.subscription = config.TriggerMetadata["subscription"]
	default:
		return meta, errors.New("no subscription given")
	}

	meta.msgBacklogThreshold = defaultMsgBacklogThreshold

	if val, ok := config.TriggerMetadata[msgBacklogMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %s", msgBacklogMetricName, err)
		}
		meta.msgBacklogThreshold = t
	}

	meta.enableTLS = false
	if val, ok := config.AuthParams["tls"]; ok {
		val = strings.TrimSpace(val)

		if val == "enable" {
			certGiven := config.AuthParams["cert"] != ""
			keyGiven := config.AuthParams["key"] != ""
			if certGiven && !keyGiven {
				return meta, errors.New("key must be provided with cert")
			}
			if keyGiven && !certGiven {
				return meta, errors.New("cert must be provided with key")
			}
			meta.ca = config.AuthParams["ca"]
			meta.cert = config.AuthParams["cert"]
			meta.key = config.AuthParams["key"]
			meta.enableTLS = true
		} else {
			return meta, fmt.Errorf("err incorrect value for TLS given: %s", val)
		}
	}
	return meta, nil
}

func (s *pulsarScaler) GetStats() (*pulsarStats, error) {
	stats := new(pulsarStats)

	req, err := http.NewRequest("GET", s.metadata.statsURL, nil)
	if err != nil {
		pulsarLog.Error(err, "error requesting stats from url")
		return nil, fmt.Errorf("error requesting stats from url: %s", err)
	}

	res, err := s.client.Do(req)
	if res == nil || err != nil {
		pulsarLog.Error(err, "error requesting stats from url")
		return nil, fmt.Errorf("error requesting stats from url: %s", err)
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			pulsarLog.Error(err, "error requesting stats from url")
			return nil, fmt.Errorf("error requesting stats from url: %s", err)
		}
		err = json.Unmarshal(body, stats)
		if err != nil {
			pulsarLog.Error(err, "error unmarshalling response")
			return nil, fmt.Errorf("error unmarshalling response: %s", err)
		}
		return stats, nil
	case 404:
		return nil, nil
	default:
		pulsarLog.Error(err, "error requesting stats from url")
		return nil, fmt.Errorf("error requesting stats from url: %s", err)
	}
}

func (s *pulsarScaler) getMsgBackLog() (int, bool, error) {
	stats, err := s.GetStats()
	if err != nil {
		return 0, false, err
	}

	if stats == nil {
		return 0, false, nil
	}

	v, found := stats.Subscriptions[s.metadata.subscription]

	return v.Msgbacklog, found, nil
}

// IsActive determines if we need to scale from zero
func (s *pulsarScaler) IsActive(_ context.Context) (bool, error) {
	msgBackLog, found, err := s.getMsgBackLog()
	if err != nil {
		return false, err
	}

	if !found || msgBackLog == 0 {
		pulsarLog.Info("Pulsar subscription is not active, either no subscription found or no backlog detected")
		return false, nil
	}

	pulsarLog.Info("Pulsar subscription is active with backlog")
	return true, nil
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *pulsarScaler) GetMetrics(_ context.Context, metricName string, _ labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	msgBacklog, found, err := s.getMsgBackLog()
	if err != nil {
		return nil, fmt.Errorf("error requesting stats from url: %s", err)
	}

	if !found {
		return nil, nil
	}

	pulsarLog.Info(fmt.Sprintf("Pulsar MsgBacklog: %d", msgBacklog))

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(msgBacklog), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *pulsarScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(s.metadata.msgBacklogThreshold, resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s-%s", "pulsar", s.metadata.tenant, s.metadata.namespace, s.metadata.topic, s.metadata.subscription)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: pulsarMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

func (s *pulsarScaler) Close() error {
	s.client = nil
	return nil
}
