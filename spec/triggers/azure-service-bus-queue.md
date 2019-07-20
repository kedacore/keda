# Azure Service Bus Queue
Example: [`examples/azureservicebus_scaledobject.yaml`](./../../examples/azurequeue_scaledobject.yaml)

```yaml
  triggers:
  - type: azure-servicebus
    metadata:
      queueName: functions-sbqueue
      connection: SERVICEBUS_CONNECTIONSTRING_ENV_NAME # This must be a connection string for a queue itself, and not a namespace level (e.g. RootAccessPolicy) connection string [#215](https://github.com/kedacore/keda/issues/215)
      queueLength: '5' # Optional. Subscription length target for HPA. Default: 5 messages
```

The `connection` value is the name of the environment variable your deployment uses to get the connection string. This is usually resolved from a `Secret V1` or a `ConfigMap V1` collections. `env` and `envFrom` are both supported.