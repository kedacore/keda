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
authentication:
    podIdentity:
        provider: none | azure | gcp | spiffe # Optional. Default: none
    secretTargetRef: # Optional.
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

Every trigger needs to define a `type` which refers to the dependency on which the ScaledObject should scale.

Every dependency requires a set of things to be configured via the `metadata` section, such as the name of the queue.
For more information, read [the supported trigger type](](#supported-trigger-types)) specification.

```yaml
  type: {trigger-type}
    metadata:
      # {list of properties to configure a trigger}
```

## Authentication

Every trigger needs to authenticate to the dependency to scale on which is configured via the `authentication` section.

Trigger types can define one or more `parameter` that have to be configured, which you can configure here - We provide a variety of sources.

```yaml
    authentication: # required
```

In order to determine what set of parameters you need to define we recommend reading the specification for the trigger type that you need.

### Environment variable(s)

You can pull information via one or more environment variables by providing the `name` of the variable for a given `containerName`.

```yaml
    env: # Optional.
    - parameter: region # Required.
      name: my-env-var # Required.
      containerName: my-container # Optional. Default: scaleTargetRef.containerName of ScaledObject
```

**Assumptions:** `containerName` is in the same deployment as the configured `scaleTargetRef.deploymentName` in the ScaledObject, unless specified otherwise.

### Secret(s)

You can pull one or more secrets into the trigger by defining the `name` of the Kubernetes Secret and the `key` to use.

```yaml
    secretTargetRef: # Optional.
    - parameter: connectionString # Required.
      name: my-keda-secret-entity # Required.
      key: azure-storage-connectionstring # Required.
      namespace: my-keda-namespace # Optional. Default: Namespace of KEDA
```
**Assumptions:** `namespace` is in the same deployment as the configured `scaleTargetRef.deploymentName` in the ScaledObject, unless specified otherwise.

### Pod Authentication Providers

Several service providers allow you to assign an identity to a pod. By using that identity, you can defer authentication to the pod & the service provider, rather than configuring secrets.

Currently we support the following:

```yaml
    podIdentity:
        provider: none | azure  # Optional. Default: false
```

#### Azure Pod Identity

Azure Pod Identity is an implementation of [Azure AD Pod Identity](https://github.com/Azure/aad-pod-identity) which let's you bind an [Azure Managed Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/) to a Pod in a Kubernetes cluster as delegated access.

You can tell KEDA to use Azure AD Pod Identity via `podIdentity.provider`.

 - https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/
```yaml
    podIdentity:
        provider: azure  # Optional. Default: false
```

# Supported Trigger Types

Here is an overview:

| Name                          | Type               | Specification                             |
|:------------------------------|:-------------------|:------------------------------------------|
| Apache Kafka Topic            | `kafka`            | [spec](./apache-kafka-topic.md)  |
| Azure Event Hub               | `azure-eventhub`   | [spec](./azure-event-hub.md)     |
| Azure Service Bus Queue/Topic | `azure-servicebus` | [spec](./azure-service-bus.md)   |
| Azure Storage Queue           | `azure-queue`      | [spec](./azure-storage-queue.md) |
| RabbitMQ Queue                | `rabbitmq`         | [spec](./rabbit-mq-queue.md)     |
