# Trigger Specification

This specification describes the `trigger` section of the `ScaledObject` used to define how what triggers KEDA should use to scale your application.

- [Metadata](#metadata)
- [Authentication](#authentication)
    - [Environment variable(s)](#authentication)
    - [Secret(s)](#authentication)
    - [Azure Pod Identity](#authentication)
- [Supported Triggers](#supported-triggers)

```yaml
type: {trigger-type} # Required.
metadata:
    # {list of properties to configure a trigger}
authentication:
    azurePodIdentity: true # Optional. Default: false
    secretRef: # Optional.
    - parameter: connectionString # Required.
      name: my-keda-secret-entity # Required.
      key: azure-storage-connectionstring # Required.
      namespace: my-keda-namespace  # Optional. Default: Namespace of KEDA
    env: # Optional.
    - parameter: region # Required.
      name: my-env-var # Required.
      containerName: my-container # Optional. Default: scaleTargetRef.containerName of ScaledObject
```

## Metadata

```yaml
  type: {trigger-type}
    metadata:
        # {list of properties to configure a trigger}
```

## Authentication

```yaml
    authentication: # required
```

### Environment variable(s)
```yaml
    env: # Optional.
    - parameter: region # Required.
    name: my-env-var # Required.
    containerName: my-container # Optional. Default: scaleTargetRef.containerName of ScaledObject
```

### Secret(s)
```yaml
    secretRef: # Optional.
    - parameter: connectionString # Required.
    name: my-keda-secret-entity # Required.
    key: azure-storage-connectionstring # Required.
    namespace: my-keda-namespace # Optional. Default: Namespace of KEDA
```

### Azure Pod Identity
https://github.com/Azure/aad-pod-identity - https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/
```yaml
    azurePodIdentity: true # Optional. Default: false
```

# Supported Triggers

Here is an overview:

| Name                          | Type               | Specification                             |
|:------------------------------|:-------------------|:------------------------------------------|
| Apache Kafka Topic            | `kafka`            | [spec](./triggers/apache-kafka-topic.md)  |
| Azure Event Hub               | `azure-eventhub`   | [spec](./triggers/azure-event-hub.md)     |
| Azure Service Bus Queue/Topic | `azure-servicebus` | [spec](./triggers/azure-service-bus.md)   |
| Azure Storage Queue           | `azure-queue`      | [spec](./triggers/azure-storage-queue.md) |
| RabbitMQ Queue                | `rabbitmq`         | [spec](./triggers/rabbit-mq-queue.md)     |
