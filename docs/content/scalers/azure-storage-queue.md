+++
fragment = "content"
weight = 100
title = "Azure Storage Queue"
background = "light"
+++

Scale applications based on Azure Storage Queues.

* **Availability:** v1.0 and above
* **Maintainer:** Microsoft

<!--more-->

### Trigger Specification

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

### Authentication Parameters

Not supported yet.

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: azure-queue-scaledobject
  namespace: default
  labels:
    deploymentName: azurequeue-function
spec:
  scaleTargetRef:
    deploymentName: azurequeue-function
  triggers:
  - type: azure-queue
    metadata:
      # Required
      queueName: functionsqueue
      # Optional
      connection: STORAGE_CONNECTIONSTRING_ENV_NAME # default AzureWebJobsStorage
      queueLength: "5" # default 5
```