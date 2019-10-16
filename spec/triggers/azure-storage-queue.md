# Azure Storage Queue Trigger

This specification describes the `azure-queue` trigger for Azure Storage Queue.

```yaml
triggers:
  - type: azure-queue
    metadata:
      queueName: functionsqueue
      queueLength: '5' # Optional. Queue length target for HPA. Default: 5 messages
      connection: STORAGE_CONNECTIONSTRING_ENV_NAME
```

The `connection` value is the name of the environment variable your deployment uses to get the connection string. This is usually resolved from a `Secret V1` or a `ConfigMap V1` collections. `env` and `envFrom` are both supported.

## Example

[`examples/azurequeue_scaledobject.yaml`](./../../examples/azurequeue_scaledobject.yaml)

## Using TriggerAuthentication

```yaml
# trigger
triggers:
  - type: azure-queue
    authenticationRef:
      name: azure-queue-auth
    metadata:
      queueName: queueName
```

```yaml
# auth object
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
  name: azure-queue-auth
spec:
  secretTargetRef:
  - parameter: connection
    name: my-secret-for-azure-storage
    key: connectionString
```

```yaml
# secret object for the TriggerAuthentication ref above
apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
  labels:
data:
  connectionString: Q29ubmVjdGlvbiBzdHJpbmcgdmFsdWUgaW4gYmFzZTY0IGVuY29kaW5nIGdvZXMgaGVyZQ==
```

## Using Pod Identity

**Note:** Only `azure` pod identity is implemented.

```yaml
# trigger
triggers:
  - type: azure-queue
    authenticationRef:
      name: azure-queue-auth
    metadata:
      queueName: queueName
```

```yaml
# auth object
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
metadata:
  name: azure-queue-auth
spec:
  podIdentity:
    provider: azure
```
