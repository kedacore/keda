package k8s

import (
	"fmt"

	"k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var metricsClientLog = ctrl.Log.WithName("metricsclient")

// InitPodMetricsClient initializes the client for pod metrics. It is used to fetch pod metrics from the metrics server.
func InitPodMetricsClient(mgr ctrl.Manager) (v1beta1.PodMetricsesGetter, error) {
	clientset, err := v1beta1.NewForConfig(mgr.GetConfig())
	if err != nil {
		metricsClientLog.Error(err, "not able to create metrics client")
		return nil, fmt.Errorf("failed to create metrics clientset: %w", err)
	}

	return clientset, nil
}
