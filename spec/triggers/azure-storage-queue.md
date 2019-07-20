# Azure Storage Queue
Example: [`examples/azurequeue_scaledobject.yaml`](./../../examples/azurequeue_scaledobject.yaml)

```yaml
  triggers:
  - type: azure-queue
    metadata:
      queueName: functionsqueue
      queueLength: '5' # Optional. Queue length target for HPA. Default: 5 messages
      connection: STORAGE_CONNECTIONSTRING_ENV_NAME
```

The `connection` value is the name of the environment variable your deployment uses to get the connection string. This is usually resolved from a `Secret V1` or a `ConfigMap V1` collections. `env` and `envFrom` are both supported.