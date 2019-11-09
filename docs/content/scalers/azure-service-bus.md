+++
fragment = "content"
weight = 100
title = "Azure Service Bus"
background = "light"
+++

Scale applications based on Azure Service Bus Queues or Topics.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Microsoft

### Trigger Specification

This specification describes the `azure-servicebus` trigger for Azure Service Bus Queue or Topic.

```yaml
  triggers:
  - type: azure-servicebus
    metadata:
      # Required: queueName OR topicName and subscriptionName
      queueName: functions-sbqueue
      # or
      topicName: functions-sbtopic
      subscriptionName: sbtopic-sub1
      # Required
      connection: SERVICEBUS_CONNECTIONSTRING_ENV_NAME # This must be a connection string for a queue itself, and not a namespace level (e.g. RootAccessPolicy) connection string [#215](https://github.com/kedacore/keda/issues/215)
      # Optional
      queueLength: "5" # Optional. Subscription length target for HPA. Default: 5 messages
```

The `connection` value is the name of the environment variable your deployment uses to get the connection string. This is usually resolved from a `Secret V1` or a `ConfigMap V1` collections. `env` and `envFrom` are both supported.

### Authentication Parameters

To be documented.

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: azure-servicebus-queue-scaledobject
  namespace: default
  labels:
    deploymentName: azure-servicebus-queue-function
spec:
  scaleTargetRef:
    deploymentName: azure-servicebus-queue-function
  triggers:
  - type: azure-servicebus
    metadata:
      # Required: queueName OR topicName and subscriptionName
      queueName: functions-sbqueue
      # or
      topicName: functions-sbtopic
      subscriptionName: sbtopic-sub1
      # Required
      connection: SERVICEBUS_CONNECTIONSTRING_ENV_NAME
      # Optional
      queueLength: "5" # default 5

```