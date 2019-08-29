# Trigger Specification

This specification describes the `trigger` section of the `ScaledObject` used to define how what triggers KEDA should use to scale your application.

<details>
  <summary><b>Table of Contents</b></summary>

- [Metadata](#metadata)
- [Authentication](#authentication)
    - [Environment variable(s)](#environment-variables)
    - [Secret(s)](#secrets)
    - [Pod Authentication Providers](#pod-authentication-providers)
        - [Azure Pod Identity](#azure-pod-identity)
- [Supported Trigger Typess](#supported-trigger-types)
</details>

[`types.go`](./../pkg/apis/keda/v1alpha1/types.go)

```yaml
type: {trigger-type} # Required.
metadata:
    # {list of properties to configure a trigger}
authenticationRef:
    - name: keda-trigger-auth-azure-queue-secret
      namespace: keda
```

## Metadata

Every trigger needs to define a `type` which refers to the dependency on which the ScaledObject should scale.

Every dependency requires a set of things to be configured via the `metadata` section, such as the name of the queue.
For more information, read [the supported trigger type](](#supported-trigger-types)) specification.

```yaml
  type: {trigger-type}
    metadata:
      # {list of properties to configure a trigger}
```

## Authentication

Every trigger needs to authenticate to the dependency to scale on which is configured via the `authenticationRef` section.

Trigger types can define one or more `parameter` that have to be configured, which you can configure here by referring to a [`TriggerAuthentication` CRD](./../TriggerAuthentication.md) that is deployed in the cluster.

```yaml
    authenticationRef:  # required
    - name: keda-trigger-auth-azure-queue-secret # required
      namespace: keda # optional
```

`name` refers to the name of the `TriggerAuthentication` that is deployed and describes how the trigger should authenticate.

**Assumptions:** `namespace` is in the same deployment as the configured `scaleTargetRef.deploymentName` in the ScaledObject, unless specified otherwise.

# Supported Trigger Types

Here is an overview:

| Name                          | Type               | Specification                             |
|:------------------------------|:-------------------|:------------------------------------------|
| Apache Kafka Topic            | `kafka`            | [spec](./apache-kafka-topic.md)  |
| Azure Event Hub               | `azure-eventhub`   | [spec](./azure-event-hub.md)     |
| Azure Service Bus Queue/Topic | `azure-servicebus` | [spec](./azure-service-bus.md)   |
| Azure Storage Queue           | `azure-queue`      | [spec](./azure-storage-queue.md) |
| RabbitMQ Queue                | `rabbitmq`         | [spec](./rabbit-mq-queue.md)     |
