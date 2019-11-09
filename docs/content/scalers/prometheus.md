+++
fragment = "content"
weight = 100
title = "Prometheus"
background = "light"
+++

Scale applications based on a Prometheus.

* **Availability:** v1.0 and above
* **Maintainer:** Community

<!--more-->

### Trigger Specification

This specification describes the `prometheus` trigger that scales based on a Prometheus.

```yaml
triggers:
  - type: prometheus
    metadata:
      # Required
      serverAddress: http://<prometheus-host>:9090
      metricName: http_requests_total
      threshold: '100'
      query: sum(rate(http_requests_total{deployment="my-deployment"}[2m])) # Note: query must return a vector/scalar single element response
```

The `serverAddress` indicates where Prometheus is running which contains the configured metric defined in `metricName` or `query`.

### Authentication Parameters

Not supported yet.

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: prometheus-scaledobject
  namespace: default
  labels:
    deploymentName: my-deployment
spec:
  scaleTargetRef:
    deploymentName: my-deployment
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://<prometheus-host>:9090
      metricName: http_requests_total
      threshold: '100'
      query: sum(rate(http_requests_total{deployment="my-deployment"}[2m]))
```