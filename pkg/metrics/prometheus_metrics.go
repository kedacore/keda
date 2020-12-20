package metrics

import (
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricLabels      = []string{"namespace", "metric", "scaledObject", "scaler", "scalerIndex"}
	scalerErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "keda_metrics_adapter",
			Subsystem: "scaler",
			Name:      "errors_total",
			Help:      "Total number of errors for all scalers",
		},
		[]string{},
	)
	scalerMetricsValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "keda_metrics_adapter",
			Subsystem: "scaler",
			Name:      "metrics_value",
			Help:      "Metric Value used for HPA",
		},
		metricLabels,
	)
	scalerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "keda_metrics_adapter",
			Subsystem: "scaler",
			Name:      "errors",
			Help:      "Number of scaler errors",
		},
		metricLabels,
	)
	scaledObjectErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "keda_metrics_adapter",
			Subsystem: "scaled_object",
			Name:      "errors",
			Help:      "Number of scaled object errors",
		},
		[]string{"namespace", "scaledObject"},
	)
)

// PrometheusMetricServer the type of MetricsServer
type PrometheusMetricServer struct{}

var registry *prometheus.Registry

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(scalerErrorsTotal)
	registry.MustRegister(scalerMetricsValue)
	registry.MustRegister(scalerErrors)
	registry.MustRegister(scaledObjectErrors)
}

// NewServer creates a new http serving instance of prometheus metrics
func (metricsServer PrometheusMetricServer) NewServer(address string, pattern string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Fatalf("Unable to write to serve custom metrics: %v", err)
		}
	})
	log.Printf("Starting metrics server at %v", address)
	http.Handle(pattern, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// initialize the total error metric
	_, errscaler := scalerErrorsTotal.GetMetricWith(prometheus.Labels{})
	if errscaler != nil {
		log.Fatalf("Unable to initialize total error metrics as : %v", errscaler)
	}

	log.Fatal(http.ListenAndServe(address, nil))
}

// RecordHPAScalerMetric create a measurement of the external metric used by the HPA
func (metricsServer PrometheusMetricServer) RecordHPAScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value int64) {
	scalerMetricsValue.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(float64(value))
}

// RecordHPAScalerError counts the number of errors occurred in trying get an external metric used by the HPA
func (metricsServer PrometheusMetricServer) RecordHPAScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error) {
	if err != nil {
		scalerErrors.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Inc()
		// scaledObjectErrors.With(prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject}).Inc()
		metricsServer.RecordScalerObjectError(namespace, scaledObject, err)
		scalerErrorsTotal.With(prometheus.Labels{}).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scalerErrors.GetMetricWith(getLabels(namespace, scaledObject, scaler, scalerIndex, metric))
	if errscaler != nil {
		log.Fatalf("Unable to write to serve custom metrics: %v", errscaler)
	}
}

// RecordScalerObjectError counts the number of errors with the scaled object
func (metricsServer PrometheusMetricServer) RecordScalerObjectError(namespace string, scaledObject string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject}
	if err != nil {
		scaledObjectErrors.With(labels).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaledobject := scaledObjectErrors.GetMetricWith(labels)
	if errscaledobject != nil {
		log.Fatalf("Unable to write to serve custom metrics: %v", errscaledobject)
		return
	}
}

func getLabels(namespace string, scaledObject string, scaler string, scalerIndex int, metric string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject, "scaler": scaler, "scalerIndex": strconv.Itoa(scalerIndex), "metric": metric}
}
