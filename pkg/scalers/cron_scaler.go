package scalers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ringtail/go-cron"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultDesiredReplicas        = 1
	cronMetricType                = "External"
)

type cronScaler struct {
	metadata               *cronMetadata
	deploymentName         string
	namespace              string
	startCron              *cron.Cron
	endCron                *cron.Cron
	client                 client.Client
}

type cronMetadata struct {
	start            string
	end              string
	timezone         string
	metricName       string
	desiredReplicas  int64
}

var cronLog = logf.Log.WithName("cron_scaler")

// NewCronScaler creates a new cronScaler
func NewCronScaler(client client.Client, deploymentName, namespace string, resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, parseErr := parseCronMetadata(metadata, resolvedEnv)
	if parseErr != nil {
		return nil, fmt.Errorf("error parsing cron metadata: %s", parseErr)
	}

	location, err := time.LoadLocation(meta.timezone)
	if err != nil {
		return nil, fmt.Errorf("Unable to load timezone. Error: %s", err)
	}

	startCron, scronErr := initCron(location, meta.start)
	if scronErr != nil {
		return nil, fmt.Errorf("error initializing start cron: %s", scronErr)
	}

	endCron, ecronErr := initCron(location, meta.end)
	if ecronErr != nil {
		return nil, fmt.Errorf("error intializing end cron: %s", ecronErr)
	}

	startTime := startCron.Entries()[0].Next.Unix()
	endTime   := endCron.Entries()[0].Next.Unix()

	if startTime > endTime {
		return nil, fmt.Errorf("start time cannot be greater than end time while initializing itself. %s", metadata)
	}

	return &cronScaler{
		metadata               : meta,
		deploymentName         : deploymentName,
		namespace              : namespace,
		startCron              : startCron,
		endCron                : endCron,
		client                 : client,
	}, nil
}

func initCron(location *time.Location, spec string) (*cron.Cron, error) {
	cron := cron.NewWithLocation(location)
	err := cron.AddFunc( spec , func() (msg string, err error) {
		return "Cron initialized", nil
	})
	if err != nil {
		return nil, err
	}

	cron.Start()

	return cron, nil
}

func parseCronMetadata(metadata, resolvedEnv map[string]string) (*cronMetadata, error) {
	if len(metadata) == 0 {
		return nil, fmt.Errorf("Invalid Input Metadata. %s", metadata)
	}

	meta := cronMetadata{}
	if val, ok := metadata["timezone"]; ok && val != "" {
		meta.timezone = val
	} else {
		return nil, fmt.Errorf("No timezone specified. %s", metadata)
	}
	if val, ok := metadata["start"]; ok && val != "" {
		meta.start = val
	} else {
		return nil, fmt.Errorf("No start schedule specified. %s", metadata)
	}
	if val, ok := metadata["end"]; ok && val != "" {
		meta.end = val
	} else {
		return nil, fmt.Errorf("No end schedule specified. %s", metadata)
	}
	if val, ok := metadata["metricName"]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("No metricName specified. %s", metadata)
	}
	if val, ok := metadata["desiredReplicas"]; ok && val != "" {
		metadataDesiredReplicas, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("Error parsing desiredReplicas metadata. %s", metadata)
		} else {
			meta.desiredReplicas = int64(metadataDesiredReplicas)
		}
	}

	return &meta, nil
}

// IsActive checks if the startTime or endTime has reached
func (s *cronScaler) IsActive(ctx context.Context) (bool, error) {
    var currentTime = time.Now().Unix()
    startTime := s.startCron.Entries()[0].Next.Unix()
	endTime := s.endCron.Entries()[0].Next.Unix()

    if startTime < endTime && currentTime < startTime {
    	return false, nil
    } else if currentTime <= endTime {
    	return true, nil
	} else {
		return false, nil
    }
}

func (s *cronScaler) Close() error {
	s.startCron.Stop()
	s.endCron.Stop()
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *cronScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
    return []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				MetricName:         s.metadata.metricName,
				TargetAverageValue: resource.NewQuantity(int64(defaultDesiredReplicas), resource.DecimalSI),
			},
			Type: cronMetricType,
		},
	}
}

// GetMetrics finds the current value of the metric
func (s *cronScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {

	deployment := &appsv1.Deployment{}
	err := s.client.Get(context.TODO(), types.NamespacedName{Name: s.deploymentName, Namespace: s.namespace}, deployment)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting deployment: %s", err)
	}

	var currentReplicas = int64(defaultDesiredReplicas)
	isActive, _ := s.IsActive(ctx)
	if isActive {
		currentReplicas = s.metadata.desiredReplicas
	}

    /*******************************************************************************/
	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(currentReplicas, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
