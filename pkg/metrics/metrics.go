package metrics

// Server an HTTP serving instance to track metrics
type Server interface {
	NewServer(address string, pattern string)
	RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error)
	RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value int64)
	RecordScalerObjectError(namespace string, scaledObject string, err error)
}
