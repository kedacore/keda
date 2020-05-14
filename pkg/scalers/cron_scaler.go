package scalers

import (
	"context"
	"fmt"
	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strconv"
	"time"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultDesiredReplicas        = 1
	cronMetricType                = "External"
)

type cronScaler struct {
	metadata *cronMetadata
	client client.Client
	namespace string
}

type cronMetadata struct {
	startTime        int64
	endTime          int64
	deploymentName   string
	namespace        string
	metricName       string
	desiredReplicas  int64
}

var cronLog = logf.Log.WithName("cron_scaler")

// NewCronScaler creates a new cronScaler
func NewCronScaler(deploymentName, namespace string, resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseCronMetadata(deploymentName, namespace, metadata, resolvedEnv, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing cron metadata: %s", err)
	}

	client, err := getK8sClient()
	if err != nil {
		return nil, err
	}
	return &cronScaler{
		metadata: meta,
		client: client,
	}, nil
}

func getK8sClient() (client.Client, error) {

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cronLog.Error(err, "CronScaler Client init: failed to get the config")
		return nil, err
	}

	scheme := scheme.Scheme
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		cronLog.Error(err, "CronScaler Client init: failed to add apps/v1 scheme to runtime scheme")
		return nil, err
	}
	if err := kedav1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		cronLog.Error(err, "CronScaler Client init: failed to add keda scheme to runtime scheme")
		return nil, err
	}

	kubeclient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		cronLog.Error(err, "CronScaler Client init: unable to construct new client")
		return nil, err
	}

	return kubeclient, nil
}
func parseCronMetadata(deploymentName, namespace string, metadata, resolvedEnv, authParams map[string]string) (*cronMetadata, error) {
	meta := cronMetadata{}
	meta.deploymentName = deploymentName
	meta.namespace = namespace
	meta.startTime = 0
	meta.endTime = 0

	if val, ok := metadata["startTime"]; ok && val != "" {
	    metadataStartTime, err := strconv.Atoi(val)
	    if err != nil {
			cronLog.Error(err, "Error parsing startTime metadata")
		} else {
		    meta.startTime = int64(metadataStartTime)
		}
	}
	if val, ok := metadata["endTime"]; ok && val != "" {
	    metadataEndTime, err := strconv.Atoi(val)
	    if err != nil {
			cronLog.Error(err, "Error parsing endTime metadata")
		} else {
		    meta.endTime = int64(metadataEndTime)
		}
	}
	if val, ok := metadata["metricName"]; ok && val != "" {
		meta.metricName = val
	}
	if val, ok := metadata["desiredReplicas"]; ok && val != "" {
		metadataDesiredReplicas, err := strconv.Atoi(val)
		if err != nil {
			cronLog.Error(err, "Error parsing maxPods metadata")
		} else {
			meta.desiredReplicas = int64(metadataDesiredReplicas)
		}
	}

	return &meta, nil
}

// IsActive checks if the startTime or endTime has reached
func (s *cronScaler) IsActive(ctx context.Context) (bool, error) {
    var currentTime = time.Now().Unix()

    //IST, _ := time.LoadLocation("Asia/Kolkata")
    //if(IST == nil) {
    //	cronLog.V(0).Info("Unable to load time. INACTIVE SCALER ")
    //	return false, nil
	//} else {
	//	cronLog.V(0).Info(fmt.Sprintf("Time present", IST))
	//}
	//c := cron.New(cron.WithLocation(IST))
	//c.AddFunc("0 30 * * * *", func() {fmt.Println("Every half an hour" )})
	//c.AddFunc("0 10 20 13 May ?", func() { fmt.Println("13th May 8:10 PM" )})
	////c.Start()
	//
	//cronLog.V(0).Info(fmt.Sprintf("CK Next cron start: %s", c.Entries()))
	//cronLog.V(0).Info(fmt.Sprintf("CK Next cron start: %s", c.Entries()[0].Next.Unix()))
	//cronLog.V(0).Info(fmt.Sprintf("CK Next cron end: %s", c.Entries()[1].Next.Unix()))
	//
	////for _, entry := range c.Entries() {
	////	sec := entry.Next.Unix()
	////	cronLog.V(0).Info(fmt.Sprintf("CK Next cron start: %s", sec))
	////}
	//
	//c.Stop()

	if currentTime >= s.metadata.startTime && currentTime < s.metadata.endTime {
        cronLog.Info("CK SCALER Active")
        return true, nil
    } else {
        cronLog.Info("CK SCALER Inactive")
        return false, nil
    }
}

func (s *cronScaler) Close() error {
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
	err := s.client.Get(context.TODO(), types.NamespacedName{Name: s.metadata.deploymentName, Namespace: s.metadata.namespace}, deployment)
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
