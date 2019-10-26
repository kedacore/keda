## Scalers 
_This document is in an early stage, contributions and help is appreciated_.

The scalers in KEDA are implementations of a KEDA Go interface called `scaler.go`. 

### `GetMetrics`

This is the key function of a scaler; it returns a value that represents a current state of an external metric (e.g. length of a queue). The return type is an `ExternalMetricValue` struct which has the following fields:
- MetricName: this is the name of the metric that we are returning.
- Timestamp: indicates the time at which the metrics were produced.
- WindowSeconds: //TODO sorry what is this again?
- Value: A numerical value that represents the state of the metric. It could be the length of a queue, or it can be the amount of lag in a stream, but it can also be a simple representation of the state.

Kubernetes HPA (Horizontal Pod Autoscaler) will poll `GetMetrics` reulgarly through KEDA's metric server (as long as there is at least one pod), and compare the returned value to a configured value in the ScaledObject configuration. Kubernetes will use the following formula to decide whether to scale the pods up and down:  

`desiredReplicas = ceil[currentReplicas * ( currentMetricValue / desiredMetricValue )]`. 

For more details check [Kubernetes HPA documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).

### `GetMetricSpecForScaling`

