package scalers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	clientv3 "go.etcd.io/etcd/client/v3"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	endpoints                          = "endpoints"
	watchKey                           = "watchKey"
	value                              = "value"
	activationValue                    = "activationValue"
	watchProgressNotifyInterval        = "watchProgressNotifyInterval"
	etcdMetricType                     = "External"
	etcdTLSEnable                      = "enable"
	etcdTLSDisable                     = "disable"
	defaultWatchProgressNotifyInterval = 600
)

type etcdScaler struct {
	metricType v2.MetricTargetType
	metadata   *etcdMetadata
	client     *clientv3.Client
	logger     logr.Logger
}

type etcdMetadata struct {
	triggerIndex int

	Endpoints                   []string `keda:"name=endpoints,                   order=triggerMetadata"`
	WatchKey                    string   `keda:"name=watchKey,                    order=triggerMetadata"`
	Value                       float64  `keda:"name=value,                       order=triggerMetadata"`
	ActivationValue             float64  `keda:"name=activationValue,             order=triggerMetadata, default=0"`
	WatchProgressNotifyInterval int      `keda:"name=watchProgressNotifyInterval, order=triggerMetadata, default=600"`

	Username string `keda:"name=username,order=authParams;resolvedEnv, optional"`
	Password string `keda:"name=password,order=authParams;resolvedEnv, optional"`

	// TLS
	EnableTLS   string `keda:"name=tls,         order=authParams, default=disable"`
	Cert        string `keda:"name=cert,        order=authParams, optional"`
	Key         string `keda:"name=key,         order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword, order=authParams, optional"`
	Ca          string `keda:"name=ca,          order=authParams, optional"`
}

func (meta *etcdMetadata) Validate() error {
	if meta.WatchProgressNotifyInterval <= 0 {
		return errors.New("watchProgressNotifyInterval must be greater than 0")
	}

	if meta.EnableTLS == etcdTLSEnable {
		if meta.Cert == "" && meta.Key != "" {
			return errors.New("cert must be provided with key")
		}
		if meta.Key == "" && meta.Cert != "" {
			return errors.New("key must be provided with cert")
		}
	} else if meta.EnableTLS != etcdTLSDisable {
		return fmt.Errorf("incorrect value for TLS given: %s", meta.EnableTLS)
	}

	if meta.Password != "" && meta.Username == "" {
		return errors.New("username must be provided with password")
	}
	if meta.Username != "" && meta.Password == "" {
		return errors.New("password must be provided with username")
	}

	return nil
}

// NewEtcdScaler creates a new etcdScaler
func NewEtcdScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, parseErr := parseEtcdMetadata(config)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing kubernetes workload metadata: %w", parseErr)
	}

	cli, err := getEtcdClients(meta)
	if err != nil {
		return nil, err
	}
	return &etcdScaler{
		metricType: metricType,
		metadata:   meta,
		client:     cli,
		logger:     InitializeLogger(config, "etcd_scaler"),
	}, nil
}

func parseEtcdMetadata(config *scalersconfig.ScalerConfig) (*etcdMetadata, error) {
	meta := &etcdMetadata{}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing redis metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}

func getEtcdClients(metadata *etcdMetadata) (*clientv3.Client, error) {
	var tlsConfig *tls.Config
	var err error
	if metadata.EnableTLS == etcdTLSEnable {
		tlsConfig, err = kedautil.NewTLSConfigWithPassword(metadata.Cert, metadata.Key, metadata.KeyPassword, metadata.Ca, false)
		if err != nil {
			return nil, err
		}
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   metadata.Endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
		Username:    metadata.Username,
		Password:    metadata.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to etcd server: %w", err)
	}

	return cli, nil
}

// Close closes the etcd client
func (s *etcdScaler) Close(context.Context) error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *etcdScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	v, err := s.getMetricValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metric value: %w", err)
	}

	metric := GenerateMetricInMili(metricName, v)
	return append([]external_metrics.ExternalMetricValue{}, metric), v > s.metadata.ActivationValue, nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA.
func (s *etcdScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("etcd-%s", s.metadata.WatchKey))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Value),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: etcdMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *etcdScaler) Run(ctx context.Context, active chan<- bool) {
	defer close(active)

	// It's possible for the watch to get terminated anytime, we need to run this in a retry loop
	runWithWatch := func() {
		s.logger.Info("run watch", "watchKey", s.metadata.WatchKey, "endpoints", s.metadata.Endpoints)
		subCtx, cancel := context.WithCancel(ctx)
		subCtx = clientv3.WithRequireLeader(subCtx)
		rch := s.client.Watch(subCtx, s.metadata.WatchKey, clientv3.WithProgressNotify())

		// rewatch to another etcd server when the network is isolated from the current etcd server.
		progress := make(chan bool)
		defer close(progress)
		go func() {
			delayDuration := time.Duration(s.metadata.WatchProgressNotifyInterval) * 2 * time.Second
			delay := time.NewTimer(delayDuration)
			defer delay.Stop()
			for {
				delay.Reset(delayDuration)
				select {
				case <-progress:
				case <-subCtx.Done():
					return
				case <-delay.C:
					s.logger.Info("no watch progress notification in the interval", "watchKey", s.metadata.WatchKey, "endpoints", s.metadata.Endpoints)
					cancel()
					return
				}
			}
		}()

		for wresp := range rch {
			progress <- wresp.IsProgressNotify()

			// rewatch to another etcd server when there is an error form the current etcd server, such as 'no leader','required revision has been compacted'
			if wresp.Err() != nil {
				s.logger.Error(wresp.Err(), "an error occurred in the watch process", "watchKey", s.metadata.WatchKey, "endpoints", s.metadata.Endpoints)
				cancel()
				return
			}

			for _, ev := range wresp.Events {
				v, err := strconv.ParseFloat(string(ev.Kv.Value), 64)
				if err != nil {
					s.logger.Error(err, "etcdValue invalid will be treated as 0")
					v = 0
				}
				active <- v > s.metadata.ActivationValue
			}
		}
	}

	// retry on error from runWithWatch() starting by 2 sec backing off * 2 with a max of 1 minute
	retryDuration := time.Second * 2
	// the caller of this function needs to ensure that they call Stop() on the resulting
	// timer, to release background resources.
	retryBackoff := func() *time.Timer {
		tmr := time.NewTimer(retryDuration)
		retryDuration *= 2
		if retryDuration > time.Minute*1 {
			retryDuration = time.Minute * 1
		}
		return tmr
	}

	// start the first runWithWatch without delay
	runWithWatch()

	for {
		backoffTimer := retryBackoff()
		select {
		case <-ctx.Done():
			backoffTimer.Stop()
			return
		case <-backoffTimer.C:
			backoffTimer.Stop()
			runWithWatch()
		}
	}
}

func (s *etcdScaler) getMetricValue(ctx context.Context) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	resp, err := s.client.Get(ctx, s.metadata.WatchKey)
	if err != nil {
		return 0, err
	}
	if resp.Kvs == nil {
		return 0, fmt.Errorf("watchKey %s doesn't exist", s.metadata.WatchKey)
	}
	v, err := strconv.ParseFloat(string(resp.Kvs[0].Value), 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}
