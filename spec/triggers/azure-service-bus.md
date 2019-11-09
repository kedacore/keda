# Azure Service Bus Trigger

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

You can also use `TriggerAuthentication` CRD with `azure-servicebus`. The trigger will look like:

```yaml
  triggers:
  - type: azure-servicebus
    metadata:
      # Required: queueName OR topicName and subscriptionName
      queueName: functions-sbqueue
      # or
      topicName: functions-sbtopic
      subscriptionName: sbtopic-sub1
    authenticationRef:
      name: azure-servicebus-auth
```

and a `TriggerAuthentication` object

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
  name: azure-servicebus-auth
spec:
  secretTargetRef:
  - parameter: connection
    name: test-auth-secrets
    key: connectionString
```

## Example

[`examples/azureservicebus_scaledobject.yaml`](./../../examples/azureservicebus_scaledobject.yaml)