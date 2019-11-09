+++
fragment = "content"
weight = 100
title = "Apache Kafka Topic"
background = "light"
+++

Scale applications based on Apache Kafka Topic.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Community

### Trigger Specification

This specification describes the `kafka` trigger for Apache Kafka Topic.

```yaml
  triggers:
  - type: kafka
    metadata:
      brokerList: kafka.svc:9092
      consumerGroup: my-group
      topic: test-topic
      lagThreshold: '5' # Optional. How much the stream is lagging on the current consumer group
```

### Authentication Parameters

Not supported yet.

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: kafka-scaledobject
  namespace: default
  labels:
    deploymentName: azure-functions-deployment
spec:
  scaleTargetRef:
    deploymentName: azure-functions-deployment
  pollingInterval: 30
  triggers:
  - type: kafka
    metadata:
      # Required
      brokerList: localhost:9092
      consumerGroup: my-group       # Make sure that this consumer group name is the same one as the one that is consuming topics
      topic: test-topic
      lagThreshold: "50"
```