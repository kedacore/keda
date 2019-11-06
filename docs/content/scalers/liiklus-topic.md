+++
fragment = "content"
weight = 100
title = "Liiklus Topic"
background = "light"
+++

Scale applications based on Liiklus Topic.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Community

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: liiklus-scaledobject
  namespace: default
  labels:
    deploymentName: function-deployment
spec:
  scaleTargetRef:
    deploymentName: function-deployment
  pollingInterval: 30
  triggers:
  - type: liiklus
    metadata:
      # Required
      address: localhost:6565
      group: my-group       # Make sure that this consumer group name is the same one as the one that is consuming topics
      topic: test-topic
      lagThreshold: "50"
```