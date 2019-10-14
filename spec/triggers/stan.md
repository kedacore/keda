# Nats Streaming Trigger

The specification describes the `stan` trigger.

```yaml
  triggers:
  - type: stan
    metadata:
      natsServerMonitoringEndpoint: "stan-nats-ss.stan.svc.cluster.local:8222"
      queueGroup: "grp1"
      durableName: "ImDurable"
      subject: "Test"
      lagThreshold: "10"
```

Where:

* `natsServerMonitoringEndpoint` : Is the location of the Nats Streaming monitoring endpoint.
* `queuGroup` : The queue group name of the subscribers.
* `durableName` :  Must identify the durability name used by the subscribers.
* `subject` : Sometimes called the channel name.
* `lagThreshold` : This value is used to tell the Horizontal Pod Autoscaler to use as TargetAverageValue.


Example [`examples/stan_scaledobject.yaml`](./../../examples/stan_scaledobject.yaml)