## Scalers 
_This document is in an early stage, contributions and help is appreciated_.

The scalers in KEDA are implementations of a KEDA Go interface called `scaler.go`. The key function in a scaler is `GetMetrics`; it returns a value that represents a current state of an external metric (e.g. length of a queue). Kubernetes HPA will poll `GetMetrics` regularly (as long as there is at least one pod) and compare the returned value to a configured value in the ScaledObject configuration (more about ScaledObjects below). Kubernetes will use the following formula to decide whether to scale the pods up and down:  

`desiredReplicas = ceil[currentReplicas * ( currentMetricValue / desiredMetricValue )]`. 

For more details check [Kubernetes HPA documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).
