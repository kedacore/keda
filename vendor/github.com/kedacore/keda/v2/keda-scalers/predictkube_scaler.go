package scalers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	libs "github.com/dysnix/predictkube-libs/external/configs"
	pc "github.com/dysnix/predictkube-libs/external/grpc/client"
	tc "github.com/dysnix/predictkube-libs/external/types_convertation"
	"github.com/dysnix/predictkube-proto/external/proto/commonproto"
	pb "github.com/dysnix/predictkube-proto/external/proto/services"
	"github.com/go-logr/logr"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/xhit/go-str2duration/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	health "google.golang.org/grpc/health/grpc_health_v1"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/authentication"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	predictKubeMetricType   = "External"
	predictKubeMetricPrefix = "predictkube_metric"

	invalidMetricTypeErr = "metric type is invalid"
)

var (
	mlEngineHost = "api.predictkube.com"
	mlEnginePort = 443

	defaultStep = time.Minute * 5

	grpcConf = &libs.GRPC{
		Enabled:       true,
		UseReflection: true,
		Compression: libs.Compression{
			Enabled: false,
		},
		Conn: &libs.Connection{
			Host:            mlEngineHost,
			Port:            uint16(mlEnginePort),
			ReadBufferSize:  50 << 20,
			WriteBufferSize: 50 << 20,
			MaxMessageSize:  50 << 20,
			Insecure:        false,
			Timeout:         time.Second * 15,
		},
		Keepalive: &libs.Keepalive{
			Time:    time.Minute * 5,
			Timeout: time.Minute * 5,
			EnforcementPolicy: &libs.EnforcementPolicy{
				MinTime:             time.Minute * 20,
				PermitWithoutStream: false,
			},
		},
	}
)

type PredictKubeScaler struct {
	metricType       v2.MetricTargetType
	metadata         *predictKubeMetadata
	prometheusClient api.Client
	grpcConn         *grpc.ClientConn
	grpcClient       pb.MlEngineServiceClient
	healthClient     health.HealthClient
	api              v1.API
	logger           logr.Logger
}

type predictKubeMetadata struct {
	PrometheusAddress   string                 `keda:"name=prometheusAddress, order=triggerMetadata"`
	PrometheusAuth      *authentication.Config `keda:"optional"`
	Query               string                 `keda:"name=query, order=triggerMetadata"`
	PredictHorizon      string                 `keda:"name=predictHorizon, order=triggerMetadata"`
	QueryStep           string                 `keda:"name=queryStep, order=triggerMetadata"`
	HistoryTimeWindow   string                 `keda:"name=historyTimeWindow, order=triggerMetadata"`
	APIKey              string                 `keda:"name=apiKey, order=authParams"`
	Threshold           float64                `keda:"name=threshold, order=triggerMetadata, optional"`
	ActivationThreshold float64                `keda:"name=activationThreshold, order=triggerMetadata, optional"`

	predictHorizon    time.Duration
	historyTimeWindow time.Duration
	stepDuration      time.Duration
	triggerIndex      int
}

func (p *predictKubeMetadata) Validate() error {
	validate := validator.New()
	err := validate.Var(p.PrometheusAddress, "url")
	if err != nil {
		return fmt.Errorf("invalid prometheusAddress")
	}

	p.predictHorizon, err = str2duration.ParseDuration(p.PredictHorizon)
	if err != nil {
		return fmt.Errorf("predictHorizon parsing error %w", err)
	}

	p.stepDuration, err = str2duration.ParseDuration(p.QueryStep)
	if err != nil {
		return fmt.Errorf("queryStep parsing error %w", err)
	}

	p.historyTimeWindow, err = str2duration.ParseDuration(p.HistoryTimeWindow)
	if err != nil {
		return fmt.Errorf("historyTimeWindow parsing error %w", err)
	}

	err = validate.Var(p.APIKey, "jwt")
	if err != nil {
		return fmt.Errorf("invalid apiKey")
	}

	return nil
}
func (s *PredictKubeScaler) setupClientConn() error {
	clientOpt, err := pc.SetGrpcClientOptions(grpcConf,
		&libs.Base{
			Monitoring: libs.Monitoring{
				Enabled: false,
			},
			Profiling: libs.Profiling{
				Enabled: false,
			},
			Single: &libs.Single{
				Enabled: false,
			},
		},
		pc.InjectPublicClientMetadataInterceptor(s.metadata.APIKey),
	)

	if !grpcConf.Conn.Insecure {
		clientOpt = append(clientOpt, grpc.WithTransportCredentials(
			credentials.NewTLS(&tls.Config{
				MinVersion: kedautil.GetMinTLSVersion(),
				ServerName: mlEngineHost,
			}),
		))
	}

	if err != nil {
		return err
	}

	connection, err := grpc.NewClient(net.JoinHostPort(mlEngineHost, fmt.Sprintf("%d", mlEnginePort)), clientOpt...)
	if err != nil {
		return err
	}
	s.grpcConn = connection

	s.grpcClient = pb.NewMlEngineServiceClient(s.grpcConn)
	s.healthClient = health.NewHealthClient(s.grpcConn)

	return err
}

// NewPredictKubeScaler creates a new PredictKube scaler
func NewPredictKubeScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (*PredictKubeScaler, error) {
	s := &PredictKubeScaler{}

	logger := InitializeLogger(config, "predictkube_scaler")
	s.logger = logger

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		logger.Error(err, "error getting scaler metric type")
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}
	s.metricType = metricType

	meta, err := parsePredictKubeMetadata(config)
	if err != nil {
		logger.Error(err, "error parsing PredictKube metadata")
		return nil, fmt.Errorf("error parsing PredictKube metadata: %3s", err)
	}

	s.metadata = meta

	err = s.initPredictKubePrometheusConn(ctx)
	if err != nil {
		logger.Error(err, "error create Prometheus client and API objects")
		return nil, fmt.Errorf("error create Prometheus client and API objects: %3s", err)
	}

	err = s.setupClientConn()
	if err != nil {
		logger.Error(err, "error init GRPC client")
		return nil, fmt.Errorf("error init GRPC client: %3s", err)
	}

	return s, nil
}

func (s *PredictKubeScaler) Close(_ context.Context) error {
	if s != nil && s.grpcConn != nil {
		return s.grpcConn.Close()
	}
	return nil
}

func (s *PredictKubeScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("predictkube-%s", predictKubeMetricPrefix))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: predictKubeMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *PredictKubeScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	value, activationValue, err := s.doPredictRequest(ctx)
	if err != nil {
		s.logger.Error(err, "error executing query to predict controller service")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	if value == 0 {
		s.logger.V(1).Info("empty response after predict request")
		return []external_metrics.ExternalMetricValue{}, false, nil
	}

	s.logger.V(1).Info(fmt.Sprintf("predict value is: %f", value))

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, activationValue > s.metadata.ActivationThreshold, nil
}

func (s *PredictKubeScaler) doPredictRequest(ctx context.Context) (float64, float64, error) {
	results, err := s.doQuery(ctx)
	if err != nil {
		return 0, 0, err
	}

	resp, err := s.grpcClient.GetPredictMetric(ctx, &pb.ReqGetPredictMetric{
		ForecastHorizon: uint64(math.Round(float64(s.metadata.predictHorizon / s.metadata.stepDuration))),
		Observations:    results,
	})

	if err != nil {
		return 0, 0, err
	}

	var y float64
	if len(results) > 0 {
		y = results[len(results)-1].Value
	}

	x := float64(resp.GetResultMetric())

	return func(x, y float64) float64 {
		if x < y {
			return y
		}
		return x
	}(x, y), y, nil
}

func (s *PredictKubeScaler) doQuery(ctx context.Context) ([]*commonproto.Item, error) {
	currentTime := time.Now().UTC()

	if s.metadata.stepDuration == 0 {
		s.metadata.stepDuration = defaultStep
	}

	r := v1.Range{
		Start: currentTime.Add(-s.metadata.historyTimeWindow),
		End:   currentTime,
		Step:  s.metadata.stepDuration,
	}

	val, warns, err := s.api.QueryRange(ctx, s.metadata.Query, r)

	if len(warns) > 0 {
		s.logger.V(1).Info("warnings", warns)
	}

	if err != nil {
		return nil, err
	}

	return s.parsePrometheusResult(val)
}

// parsePrometheusResult parsing response from prometheus server.
func (s *PredictKubeScaler) parsePrometheusResult(result model.Value) (out []*commonproto.Item, err error) {
	metricName := GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("predictkube-%s", predictKubeMetricPrefix)))
	switch result.Type() {
	case model.ValVector:
		if res, ok := result.(model.Vector); ok {
			for _, val := range res {
				t, err := tc.AdaptTimeToPbTimestamp(tc.TimeToTimePtr(val.Timestamp.Time()))
				if err != nil {
					return nil, err
				}

				out = append(out, &commonproto.Item{
					Timestamp:  t,
					Value:      float64(val.Value),
					MetricName: metricName,
				})
			}
		}
	case model.ValMatrix:
		if res, ok := result.(model.Matrix); ok {
			for _, val := range res {
				for _, v := range val.Values {
					t, err := tc.AdaptTimeToPbTimestamp(tc.TimeToTimePtr(v.Timestamp.Time()))
					if err != nil {
						return nil, err
					}

					out = append(out, &commonproto.Item{
						Timestamp:  t,
						Value:      float64(v.Value),
						MetricName: metricName,
					})
				}
			}
		}
	case model.ValScalar:
		if res, ok := result.(*model.Scalar); ok {
			t, err := tc.AdaptTimeToPbTimestamp(tc.TimeToTimePtr(res.Timestamp.Time()))
			if err != nil {
				return nil, err
			}

			out = append(out, &commonproto.Item{
				Timestamp:  t,
				Value:      float64(res.Value),
				MetricName: metricName,
			})
		}
	case model.ValString:
		if res, ok := result.(*model.String); ok {
			t, err := tc.AdaptTimeToPbTimestamp(tc.TimeToTimePtr(res.Timestamp.Time()))
			if err != nil {
				return nil, err
			}

			s, err := strconv.ParseFloat(res.Value, 64)
			if err != nil {
				return nil, err
			}

			out = append(out, &commonproto.Item{
				Timestamp:  t,
				Value:      s,
				MetricName: metricName,
			})
		}
	default:
		return nil, errors.New(invalidMetricTypeErr)
	}

	return out, nil
}

func parsePredictKubeMetadata(config *scalersconfig.ScalerConfig) (result *predictKubeMetadata, err error) {
	meta := &predictKubeMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing predictKube metadata: %w", err)
	}

	if !config.AsMetricSource && meta.Threshold == 0 {
		return nil, fmt.Errorf("no threshold given")
	}

	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}

func (s *PredictKubeScaler) ping(ctx context.Context) (err error) {
	_, err = s.api.Runtimeinfo(ctx)
	return err
}

// initPredictKubePrometheusConn init prometheus client and setup connection to API
func (s *PredictKubeScaler) initPredictKubePrometheusConn(ctx context.Context) (err error) {
	// create http.RoundTripper with auth settings from ScalerConfig
	roundTripper, err := authentication.CreateHTTPRoundTripper(
		authentication.FastHTTP,
		s.metadata.PrometheusAuth.ToAuthMeta(),
	)
	if err != nil {
		s.logger.V(1).Error(err, "init Prometheus client http transport")
		return err
	}
	client, err := api.NewClient(api.Config{
		Address:      s.metadata.PrometheusAddress,
		RoundTripper: roundTripper,
	})
	if err != nil {
		s.logger.V(1).Error(err, "init Prometheus client")
		return err
	}
	s.prometheusClient = client
	s.api = v1.NewAPI(s.prometheusClient)

	return s.ping(ctx)
}
