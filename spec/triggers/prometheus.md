# Prometheus Trigger

This specification describes the `prometheus` trigger.

```yaml
  triggers:
  - type: prometheus
    metadata:
      serverAddress: <Prometheus server URL> #Required
      metricName: <name of metric> #Required
      threshold: '<threshold at which auto-scale is triggered>' #Required
      query: <PromQL query> # Note: query must return a vector/scalar single element response
```

## Example

[`examples/prometheus_scaledobject.yaml`](./../../examples/prometheus_scaledobject.yaml)