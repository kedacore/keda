package metrics

import (
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	scaledJobMetricLabels      = []string{"namespace", "metric", "scaledJob", "scaler", "scalerIndex"}
	scaledJobScalerErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "keda_operator",
			Subsystem: "scaler",
			Name:      "errors_total",
			Help:      "Total number of errors for all scalers",
		},
		[]string{},
	)
	scaledJobScalerMetricsValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "keda_operator",
			Subsystem: "scaler",
			Name:      "metrics_value",
			Help:      "Metric Value used for ScaledJob",
		},
		scaledJobMetricLabels,
	)
	scaledJobScalerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "keda_operator",
			Subsystem: "scaler",
			Name:      "errors",
			Help:      "Number of scaler errors",
		},
		scaledJobMetricLabels,
	)
	scaledJobErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "keda_operator",
			Subsystem: "scaled_job",
			Name:      "errors",
			Help:      "Number of scaled object errors",
		},
		[]string{"namespace", "scaledObject"},
	)
)

// PrometheusMetricServer the type of MetricsServer
type ScaledJobPrometheusMetricServer struct{}

var scaledJobRegistry *prometheus.Registry

func init() {
	scaledJobRegistry = prometheus.NewRegistry()
	scaledJobRegistry.MustRegister(scaledJobScalerErrorsTotal)
	scaledJobRegistry.MustRegister(scaledJobScalerMetricsValue)
	scaledJobRegistry.MustRegister(scaledJobScalerErrors)
	scaledJobRegistry.MustRegister(scaledJobErrors)
}

// NewServer creates a new http serving instance of prometheus metrics
func (metricsServer ScaledJobPrometheusMetricServer) NewServer(address string, pattern string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Fatalf("Unable to write to serve custom metrics for scaledJob: %v", err)
		}
	})
	log.Printf("Starting ScaledJob metrics server at %v", address)
	http.Handle(pattern, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// initialize the total error metric
	_, errscaler := scaledJobScalerErrorsTotal.GetMetricWith(prometheus.Labels{})
	if errscaler != nil {
		log.Fatalf("Unable to initialize scaledJob total error metrics as : %v", errscaler)
	}

	log.Fatal(http.ListenAndServe(address, nil))
}

// RecordScaledJobScalerMetric create a measurement of the external metric used by the ScaledJob
func (metricsServer ScaledJobPrometheusMetricServer) RecordScaledJobScalerMetric(namespace string, scaledJob string, scaler string, scalerIndex int, metric string, value int64) {
	scaledJobScalerMetricsValue.With(metricsServer.getLabels(namespace, scaledJob, scaler, scalerIndex, metric)).Set(float64(value))
}

// RecordScaledJobScalerError counts the number of errors occurred in trying get an external metric used by the ScaledJob
func (metricsServer ScaledJobPrometheusMetricServer) RecordScaledJobScalerError(namespace string, scaledJob string, scaler string, scalerIndex int, metric string, err error) {
	if err != nil {
		scaledJobScalerErrors.With(metricsServer.getLabels(namespace, scaledJob, scaler, scalerIndex, metric)).Inc()
		// scaledJobErrors.With(prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject}).Inc()
		metricsServer.RecordScalerObjectError(namespace, scaledJob, err)
		scaledJobScalerErrorsTotal.With(prometheus.Labels{}).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scaledJobScalerErrors.GetMetricWith(metricsServer.getLabels(namespace, scaledJob, scaler, scalerIndex, metric))
	if errscaler != nil {
		log.Fatalf("Unable to write to serve custom metrics for scaledJob: %v", errscaler)
	}
}

// RecordScalerObjectError counts the number of errors with the scaled job
func (metricsServer ScaledJobPrometheusMetricServer) RecordScalerObjectError(namespace string, scaledJob string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledJob": scaledJob}
	if err != nil {
		scaledJobErrors.With(labels).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaledjob := scaledJobErrors.GetMetricWith(labels)
	if errscaledjob != nil {
		log.Fatalf("Unable to write to serve custom metrics for scaledJob: %v", errscaledjob)
		return
	}
}

func (metricsServer ScaledJobPrometheusMetricServer) getLabels(namespace string, scaledJob string, scaler string, scalerIndex int, metric string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "scaledJob": scaledJob, "scaler": scaler, "scalerIndex": strconv.Itoa(scalerIndex), "metric": metric}
}
