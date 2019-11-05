+++
fragment = "content"
weight = 100
title = "Azure Event Hubs"
background = "light"
+++

Scale applications based on Azure Event Hubs.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Microsoft

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: azure-eventhub-scaledobject
  namespace: default
  labels:
    deploymentName: azureeventhub-function
spec:
  scaleTargetRef:
    deploymentName: azureeventhub-function
  triggers:
  - type: azure-eventhub
    metadata:
      # Required
      connection: EventHub
      storageConnection: AzureWebJobsStorage
      # Optional
      consumerGroup: $Default # default: $Default
      unprocessedEventThreshold: '64' # default 64 events.
```