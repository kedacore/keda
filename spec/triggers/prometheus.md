# Prometheus Trigger

This specification describes the `prometheus` trigger.

```yaml
  triggers:
  - type: prometheus
    metadata:
      serverAddress: <Prometheus server URL e.g. http://<prometheus-host>:9090> #Required
      metricName: <name of metric e.g. http_requests_total> #Required
      threshold: '<threshold at which auto-scale is triggered e.g. 100>' #Required
      query: <PromQL query e.g. sum(rate(http_requests_total{deployment="my-deployment"}[2m]))> #Required
```

## Example

[`examples/prometheus_scaledobject.yaml`](./../../examples/prometheus_scaledobject.yaml)