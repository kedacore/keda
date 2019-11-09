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

### Trigger Specification

This specification describes the `azure-eventhub` trigger for Azure Event Hubs.

```yaml
  triggers:
  - type: azure-eventhub
    metadata:
      connection: EVENTHUB_CONNECTIONSTRING_ENV_NAME # Connection string for Event Hub namespace
      storageConnection: STORAGE_CONNECTIONSTRING_ENV_NAME # Connection string for account used to store checkpoint. As of now the Event Hub scaler only reads from Azure Blob Storage. 
      consumerGroup: $Default # Optional. Consumer group of event hub consumer. Default: $Default
      unprocessedEventThreshold: '64' # Optional. Target number of unprocessed events across all partitions in Event Hub for HPA. Default: 64 events.
```

The `connection` value is the name of the environment variable your deployment uses to get the Event Hub connection string. `storageConnection` is the name of the environment variable your deployment uses to get the Storage connection string.

Environment variables are usually resolved from a `Secret V1` or a `ConfigMap V1` collections. `env` and `envFrom` are both supported.

### Authentication Parameters

Not supported yet.

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