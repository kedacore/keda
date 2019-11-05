+++
fragment = "content"
weight = 100
title = "Rabbit MQ Queue"
background = "light"
+++

Scale applications based on Rabbit MQ Queue.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Community

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: rabbitmq-scaledobject
  namespace: default
  labels:
    deploymentName: rabbitmq-deployment
spec:
  scaleTargetRef:
    deploymentName: rabbitmq-deployment
  triggers:
  - type: rabbitmq
    metadata:
      # Required
      host: RabbitMqHost # references a value of format amqp://guest:password@localhost:5672/
      queueName: testqueue
      queueLength: "20"
```