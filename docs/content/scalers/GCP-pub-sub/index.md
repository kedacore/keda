+++
fragment = "content"
weight = 100
title = "Google Cloud Platform‎ Pub/Sub"
background = "light"
+++

Scale applications based on Google Cloud Platform‎ Pub/Sub.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Community

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: pubsub-scaledobject
  namespace: keda-pubsub-test
  labels:
    deploymentName: keda-pubsub-go
spec:
  scaleTargetRef:
    deploymentName: keda-pubsub-go
  triggers:
  - type: gcp-pubsub
    metadata:
      subscriptionSize: "5"
      subscriptionName: "mysubscription" # Required 
      credentials: GOOGLE_APPLICATION_CREDENTIALS_JSON # Required
```