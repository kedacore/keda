+++
fragment = "content"
weight = 100
title = "Rabbit MQ Queue"
background = "light"
+++

Scale applications based on Rabbit MQ Queue.

* **Availability:** v1.0 and above
* **Maintainer:** Community

<!--more-->

### Trigger Specification

This specification describes the `rabbitmq` trigger for Rabbit MQ Queue.

```yaml
  triggers:
  - type: rabbitmq
    metadata:
      host: RabbitMqHost
      queueLength: '20' # Optional. Queue length target for HPA. Default: 20 messages
      queueName: testqueue
```

The `host` value is the name of the environment variable your deployment uses to get the connection string. This is usually resolved from a `Secret V1` or a `ConfigMap V1` collections. `env` and `envFrom` are both supported.  The resolved host should follow a format like `amqp://guest:password@localhost:5672/`

### Authentication Parameters

Not supported yet.

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