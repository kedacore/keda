package scalers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	libs "github.com/dysnix/predictkube-libs/external/configs"
	pc "github.com/dysnix/predictkube-libs/external/grpc/client"
	"github.com/dysnix/predictkube-libs/external/http_transport"
	tc "github.com/dysnix/predictkube-libs/external/types_convertation"
	"github.com/dysnix/predictkube-proto/external/proto/commonproto"
	pb "github.com/dysnix/predictkube-proto/external/proto/services"
	validator "github.com/go-playground/validator/v10"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	str2duration "github.com/xhit/go-str2duration/v2"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	predictKubeMetricType = "External"
)

var (
	mlEngineHost = "predictkube-dev.dysnix.org"
	mlEnginePort = 8080

	defaultStep = time.Minute * 5
)

type PredictKubeScaler struct {
	metadata         *predictKubeMetadata
	prometheusClient api.Client
	grpcConn         *grpc.ClientConn
	grpcClient       pb.MlEngineServiceClient
	healthClient     health.HealthClient
	api              v1.API
	transport        http_transport.FastHttpTransport
}

type predictKubeMetadata struct {
	predictHorizon    time.Duration
	historyTimeWindow time.Duration
	stepDuration      time.Duration
	apiKey            string
	prometheusAddress string
	metricName        string
	query             string
	threshold         int64
	scalerIndex       int
}

var predictKubeLog = logf.Log.WithName("predictkube_scaler")

func (pks *PredictKubeScaler) setupClientConn() error {
	clientOpt, err := pc.SetGrpcClientOptions(&libs.GRPC{
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
			Insecure:        true,
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
	},
		&libs.Base{
			Monitoring: libs.Monitoring{
				Enabled: false,
			},
			Profiling: libs.Profiling{
				Enabled: false,
			},
			IsDebugMode: true,
			Single: &libs.Single{
				Enabled: false,
			},
		},
		pc.InjectPublicClientMetadataInterceptor(pks.metadata.apiKey),
	)

	if err != nil {
		return err
	}

	pks.grpcConn, err = grpc.Dial(fmt.Sprintf("%s:%d", mlEngineHost, mlEnginePort), clientOpt...)
	if err != nil {
		return err
	}

	pks.grpcClient = pb.NewMlEngineServiceClient(pks.grpcConn)
	pks.healthClient = health.NewHealthClient(pks.grpcConn)

	return err
}

// NewPredictKubeScaler creates a new PredictKube scaler
func NewPredictKubeScaler(_ context.Context, config *ScalerConfig) (*PredictKubeScaler, error) {
	s := &PredictKubeScaler{}

	meta, err := parsePredictKubeMetadata(config)
	if err != nil {
		predictKubeLog.Error(err, "error parsing PredictKube metadata")
		return nil, fmt.Errorf("error parsing PredictKube metadata: %3s", err)
	}

	s.metadata = meta

	err = s.initPredictKubePrometheusConn()
	if err != nil {
		predictKubeLog.Error(err, "error create Prometheus client and API objects")
		return nil, fmt.Errorf("error create Prometheus client and API objects: %3s", err)
	}

	err = s.setupClientConn()
	if err != nil {
		predictKubeLog.Error(err, "error init GRPC client")
		return nil, fmt.Errorf("error init GRPC client: %3s", err)
	}

	return s, nil
}

// IsActive returns true if we are able to get metrics from PredictKube
func (pks *PredictKubeScaler) IsActive(ctx context.Context) (bool, error) {
	results, err := pks.doQuery(ctx)
	// TODO:
	if err != nil {
		return false, err
	}

	resp, err := pks.healthClient.Check(ctx, &health.HealthCheckRequest{})

	if resp == nil || err != nil {
		err = fmt.Errorf("can't connect grpc server: %v, code: %v", err, status.Code(err))
	}

	return len(results) > 0 && err == nil, err
}

func (pks *PredictKubeScaler) Close(_ context.Context) error {
	pks.transport.Close()
	return pks.grpcConn.Close()
}

func (pks *PredictKubeScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(pks.metadata.threshold, resource.DecimalSI)
	metricName := kedautil.NormalizeString(fmt.Sprintf("predictkube-%s", pks.metadata.metricName))
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(pks.metadata.scalerIndex, metricName),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}

	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: predictKubeMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

func (pks *PredictKubeScaler) GetMetrics(ctx context.Context, metricName string, _ labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	value, err := pks.doPredictRequest(ctx)
	if err != nil {
		predictKubeLog.Error(err, "error executing query to predict controller service")
		return []external_metrics.ExternalMetricValue{}, err
	}

	if value == 0 {
		err = errors.New("empty response after predict request")
		predictKubeLog.Error(err, "")
		return nil, err
	}

	predictKubeLog.Info(fmt.Sprintf("predict value is: %d", value))

	val := *resource.NewQuantity(value, resource.DecimalSI)

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      val,
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (pks *PredictKubeScaler) doPredictRequest(ctx context.Context) (int64, error) {
	results, err := pks.doQuery(ctx)
	if err != nil {
		return 0, err
	}

	resp, err := pks.grpcClient.GetPredictMetric(ctx, &pb.ReqGetPredictMetric{
		ForecastHorizon: uint64(math.Round(float64(pks.metadata.predictHorizon / pks.metadata.stepDuration))),
		Observations:    results,
	})

	if err != nil {
		return 0, err
	}

	var y int64
	if len(results) > 0 {
		y = int64(results[len(results)-1].Value)
	}

	x := resp.GetResultMetric()

	return func(x, y int64) int64 {
		if x < y {
			return y
		}
		return x
	}(x, y), nil
}

func (pks *PredictKubeScaler) doQuery(ctx context.Context) ([]*commonproto.Item, error) {
	currentTime := time.Now().UTC()

	if pks.metadata.stepDuration == 0 {
		pks.metadata.stepDuration = defaultStep
	}

	r := v1.Range{
		Start: currentTime.Add(-pks.metadata.historyTimeWindow),
		End:   currentTime,
		Step:  pks.metadata.stepDuration,
	}

	val, warns, err := pks.api.QueryRange(ctx, pks.metadata.query, r)

	if len(warns) > 0 {
		predictKubeLog.V(1).Info("warnings", warns)
	}

	if err != nil {
		return nil, err
	}

	return pks.parsePrometheusResult(val)
}

func (pks *PredictKubeScaler) parsePrometheusResult(result model.Value) (out []*commonproto.Item, err error) {
	metricName := GenerateMetricNameWithIndex(pks.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("predictkube-%s", pks.metadata.metricName)))
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
	}

	return out, nil
}

func parsePredictKubeMetadata(config *ScalerConfig) (result *predictKubeMetadata, err error) {
	validate := validator.New()
	meta := predictKubeMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		if len(val) == 0 {
			return nil, fmt.Errorf("no query given")
		}

		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["prometheusAddress"]; ok {
		err = validate.Var(val, "url")
		if err != nil {
			return nil, fmt.Errorf("invalid prometheusAddress")
		}

		meta.prometheusAddress = val
	} else {
		return nil, fmt.Errorf("no prometheusAddress given")
	}

	if val, ok := config.TriggerMetadata["predictHorizon"]; ok {
		meta.predictHorizon, err = str2duration.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("predictHorizon parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no predictHorizon given")
	}

	if val, ok := config.TriggerMetadata["queryStep"]; ok {
		meta.stepDuration, err = str2duration.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("queryStep parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no queryStep given")
	}

	if val, ok := config.TriggerMetadata["historyTimeWindow"]; ok {
		meta.historyTimeWindow, err = str2duration.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("historyTimeWindow parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no historyTimeWindow given")
	}

	if val, ok := config.TriggerMetadata["threshold"]; ok {
		meta.threshold, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("threshold parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no threshold given")
	}

	if val, ok := config.TriggerMetadata["metricName"]; ok {
		if len(val) == 0 {
			return nil, fmt.Errorf("no metric name given")
		}

		err = validate.Var(val, "ascii")
		if err != nil {
			return nil, fmt.Errorf("invalid apiKey")
		}

		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no metric name given")
	}

	meta.scalerIndex = config.ScalerIndex

	if val, ok := config.AuthParams["apiKey"]; ok {
		err = validate.Var(val, "jwt")
		if err != nil {
			return nil, fmt.Errorf("invalid apiKey")
		}

		meta.apiKey = val
	} else {
		return nil, fmt.Errorf("no api key given")
	}

	return &meta, nil
}

func (pks *PredictKubeScaler) ping(ctx context.Context) (err error) {
	_, err = pks.api.Runtimeinfo(ctx)
	return err
}

// initPredictKubePrometheusConn init prometheus client and setup connection to API
func (pks *PredictKubeScaler) initPredictKubePrometheusConn() (err error) {
	pks.transport = http_transport.NewTransport(&libs.HTTPTransport{
		MaxIdleConnDuration: 10,
		ReadTimeout:         time.Second * 15,
		WriteTimeout:        time.Second * 15,
	})

	pks.prometheusClient, err = api.NewClient(api.Config{
		Address:      pks.metadata.prometheusAddress,
		RoundTripper: pks.transport,
	})

	if err != nil {
		predictKubeLog.V(1).Error(err, "init Prometheus client")
		return err
	}

	pks.api = v1.NewAPI(pks.prometheusClient)

	return pks.ping(context.Background())
}
