package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/oauth2/clientcredentials"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type pulsarScaler struct {
	metadata pulsarMetadata
	client   *http.Client
	logger   logr.Logger
}

type pulsarMetadata struct {
	adminURL                      string
	topic                         string
	subscription                  string
	msgBacklogThreshold           int64
	activationMsgBacklogThreshold int64

	pulsarAuth *authentication.AuthMeta

	statsURL    string
	metricName  string
	scalerIndex int
}

const (
	msgBacklogMetricName       = "msgBacklog"
	pulsarMetricType           = "External"
	defaultMsgBacklogThreshold = 10
	enable                     = "enable"
	stringTrue                 = "true"
	pulsarAuthModeHeader       = "X-Pulsar-Auth-Method-Name"
)

type pulsarSubscription struct {
	Msgrateout                       float64       `json:"msgRateOut"`
	Msgthroughputout                 float64       `json:"msgThroughputOut"`
	Bytesoutcounter                  int           `json:"bytesOutCounter"`
	Msgoutcounter                    int           `json:"msgOutCounter"`
	Msgrateredeliver                 float64       `json:"msgRateRedeliver"`
	Chuckedmessagerate               int           `json:"chuckedMessageRate"`
	Msgbacklog                       int64         `json:"msgBacklog"`
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
	logger := InitializeLogger(config, "pulsar_scaler")
	pulsarMetadata, err := parsePulsarMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing pulsar metadata: %w", err)
	}

	client := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	if pulsarMetadata.pulsarAuth != nil {
		if pulsarMetadata.pulsarAuth.CA != "" || pulsarMetadata.pulsarAuth.EnableTLS {
			config, err := authentication.NewTLSConfig(pulsarMetadata.pulsarAuth, false)
			if err != nil {
				return nil, err
			}
			client.Transport = kedautil.CreateHTTPTransportWithTLSConfig(config)
		}

		if pulsarMetadata.pulsarAuth.EnableBearerAuth || pulsarMetadata.pulsarAuth.EnableBasicAuth {
			// The pulsar broker redirects HTTP calls to other brokers and expects the Authorization header
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				if len(via) != 0 && via[0].Response.StatusCode == http.StatusTemporaryRedirect {
					addAuthHeaders(req, &pulsarMetadata)
				}
				return nil
			}
		}
	}

	return &pulsarScaler{
		client:   client,
		metadata: pulsarMetadata,
		logger:   logger,
	}, nil
}

func parsePulsarMetadata(config *ScalerConfig, logger logr.Logger) (pulsarMetadata, error) {
	meta := pulsarMetadata{}
	switch {
	case config.TriggerMetadata["adminURLFromEnv"] != "":
		meta.adminURL = config.ResolvedEnv[config.TriggerMetadata["adminURLFromEnv"]]
	case config.TriggerMetadata["adminURL"] != "":
		meta.adminURL = config.TriggerMetadata["adminURL"]
	default:
		return meta, errors.New("no adminURL given")
	}

	switch {
	case config.TriggerMetadata["topicFromEnv"] != "":
		meta.topic = config.ResolvedEnv[config.TriggerMetadata["topicFromEnv"]]
	case config.TriggerMetadata["topic"] != "":
		meta.topic = config.TriggerMetadata["topic"]
	default:
		return meta, errors.New("no topic given")
	}

	topic := strings.ReplaceAll(meta.topic, "persistent://", "")
	if config.TriggerMetadata["isPartitionedTopic"] == stringTrue {
		meta.statsURL = meta.adminURL + "/admin/v2/persistent/" + topic + "/partitioned-stats"
	} else {
		meta.statsURL = meta.adminURL + "/admin/v2/persistent/" + topic + "/stats"
	}

	switch {
	case config.TriggerMetadata["subscriptionFromEnv"] != "":
		meta.subscription = config.ResolvedEnv[config.TriggerMetadata["subscriptionFromEnv"]]
	case config.TriggerMetadata["subscription"] != "":
		meta.subscription = config.TriggerMetadata["subscription"]
	default:
		return meta, errors.New("no subscription given")
	}

	meta.metricName = fmt.Sprintf("%s-%s-%s", "pulsar", meta.topic, meta.subscription)

	meta.activationMsgBacklogThreshold = 0
	if val, ok := config.TriggerMetadata["activationMsgBacklogThreshold"]; ok {
		activationMsgBacklogThreshold, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("activationMsgBacklogThreshold parsing error %w", err)
		}
		meta.activationMsgBacklogThreshold = activationMsgBacklogThreshold
	}

	meta.msgBacklogThreshold = defaultMsgBacklogThreshold

	// FIXME: msgBacklog support DEPRECATED to be removed in v2.14
	fmt.Println(config.TriggerMetadata)
	if val, ok := config.TriggerMetadata[msgBacklogMetricName]; ok {
		logger.V(1).Info("\"msgBacklog\" is deprecated and will be removed in v2.14, please use \"msgBacklogThreshold\" instead")
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %w", msgBacklogMetricName, err)
		}
		meta.msgBacklogThreshold = t
	} else if val, ok := config.TriggerMetadata["msgBacklogThreshold"]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %w", msgBacklogMetricName, err)
		}
		meta.msgBacklogThreshold = t
	}
	// END FIXME

	// For backwards compatibility, we need to map "tls: enable" to
	if tls, ok := config.TriggerMetadata["tls"]; ok {
		if tls == enable && (config.AuthParams["cert"] != "" || config.AuthParams["key"] != "") {
			if authModes, authModesOk := config.TriggerMetadata[authentication.AuthModesKey]; authModesOk {
				config.TriggerMetadata[authentication.AuthModesKey] = fmt.Sprintf("%s,%s", authModes, authentication.TLSAuthType)
			} else {
				config.TriggerMetadata[authentication.AuthModesKey] = string(authentication.TLSAuthType)
			}
		}
	}
	auth, err := authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return meta, fmt.Errorf("error parsing %s: %w", msgBacklogMetricName, err)
	}

	if auth != nil && auth.EnableOAuth {
		if auth.OauthTokenURI == "" {
			auth.OauthTokenURI = config.TriggerMetadata["oauthTokenURI"]
		}
		if auth.Scopes == nil {
			auth.Scopes = authentication.ParseScope(config.TriggerMetadata["scope"])
		}
		if auth.ClientID == "" {
			auth.ClientID = config.TriggerMetadata["clientID"]
		}
		// client_secret is not required for mtls OAuth(RFC8705)
		// set secret to random string to work around the Go OAuth lib
		if auth.ClientSecret == "" {
			auth.ClientSecret = time.Now().String()
		}
		if auth.EndpointParams == nil {
			v, err := authentication.ParseEndpointParams(config.TriggerMetadata["EndpointParams"])
			if err != nil {
				return meta, fmt.Errorf("error parsing EndpointParams: %s", config.TriggerMetadata["EndpointParams"])
			}
			auth.EndpointParams = v
		}
	}
	meta.pulsarAuth = auth
	meta.scalerIndex = config.ScalerIndex
	return meta, nil
}

func (s *pulsarScaler) GetStats(ctx context.Context) (*pulsarStats, error) {
	stats := new(pulsarStats)

	req, err := http.NewRequestWithContext(ctx, "GET", s.metadata.statsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error requesting stats from admin url: %w", err)
	}

	client := s.client
	if s.metadata.pulsarAuth.EnableOAuth {
		config := clientcredentials.Config{
			ClientID:       s.metadata.pulsarAuth.ClientID,
			ClientSecret:   s.metadata.pulsarAuth.ClientSecret,
			TokenURL:       s.metadata.pulsarAuth.OauthTokenURI,
			Scopes:         s.metadata.pulsarAuth.Scopes,
			EndpointParams: s.metadata.pulsarAuth.EndpointParams,
		}
		client = config.Client(context.Background())
	}
	addAuthHeaders(req, &s.metadata)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error requesting stats from admin url: %w", err)
	}
	if res == nil {
		return nil, fmt.Errorf("error requesting stats from admin url, got empty response")
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("error requesting stats from admin url: %w", err)
		}
		err = json.Unmarshal(body, stats)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response: %w", err)
		}
		return stats, nil
	case 404:
		return nil, fmt.Errorf("error requesting stats from admin url, response status is (404): %s", res.Status)
	default:
		return nil, fmt.Errorf("error requesting stats from admin url, response status is: %s", res.Status)
	}
}

func (s *pulsarScaler) getMsgBackLog(ctx context.Context) (int64, bool, error) {
	stats, err := s.GetStats(ctx)
	if err != nil {
		return 0, false, err
	}

	if stats == nil {
		return 0, false, nil
	}

	v, found := stats.Subscriptions[s.metadata.subscription]

	return v.Msgbacklog, found, nil
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *pulsarScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	msgBacklog, found, err := s.getMsgBackLog(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("error requesting stats from url: %w", err)
	}

	if !found {
		return nil, false, fmt.Errorf("have not subscription found! %s", s.metadata.subscription)
	}

	metric := GenerateMetricInMili(metricName, float64(msgBacklog))

	return []external_metrics.ExternalMetricValue{metric}, msgBacklog > s.metadata.activationMsgBacklogThreshold, nil
}

func (s *pulsarScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	targetMetricValue := resource.NewQuantity(s.metadata.msgBacklogThreshold, resource.DecimalSI)

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(s.metadata.metricName)),
		},
		Target: v2.MetricTarget{
			Type:         v2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: pulsarMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *pulsarScaler) Close(context.Context) error {
	s.client = nil
	return nil
}

// addAuthHeaders add the relevant headers used by Pulsar to authenticate and authorize http requests
func addAuthHeaders(req *http.Request, metadata *pulsarMetadata) {
	if metadata.pulsarAuth == nil {
		return
	}
	switch {
	case metadata.pulsarAuth.EnableBearerAuth:
		req.Header.Add("Authorization", authentication.GetBearerToken(metadata.pulsarAuth))
		req.Header.Add(pulsarAuthModeHeader, "token")
	case metadata.pulsarAuth.EnableBasicAuth:
		req.SetBasicAuth(metadata.pulsarAuth.Username, metadata.pulsarAuth.Password)
		req.Header.Add(pulsarAuthModeHeader, "basic")
	case metadata.pulsarAuth.EnableTLS:
		// When BearerAuth or BasicAuth are also configured, let them take precedence for the purposes of
		// the authMode header.
		req.Header.Add(pulsarAuthModeHeader, "tls")
	}
}
