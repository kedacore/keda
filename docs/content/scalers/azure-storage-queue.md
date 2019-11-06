+++
fragment = "content"
weight = 100
title = "Azure Storage Queue"
background = "light"
+++

Scale applications based on Azure Storage Queues.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Microsoft

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