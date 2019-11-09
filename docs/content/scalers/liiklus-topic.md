+++
fragment = "content"
weight = 100
title = "Liiklus Topic"
background = "light"
+++

Scale applications based on Liiklus Topic.

* **Availability:** v1.0 and above
* **Maintainer:** Community

<!--more-->

### Trigger Specification

This specification describes the `liiklus` trigger for Liiklus Topic.

```yaml
  triggers:
  - type: liiklus
    metadata:
      # Required
      address: localhost:6565 # Address of the gRPC liiklus API endpoint
      group: my-group         # Make sure that this consumer group name is the same one as the one that is consuming topics
      topic: test-topic
      # Optional
      lagThreshold: "50"      # default 10, the target lag for HPA
      groupVersion: 1         # default 0, the groupVersion to consider when looking at messages. See https://github.com/bsideup/liiklus/blob/22efb7049ebcdd0dcf6f7f5735cdb5af1ae014de/app/src/test/java/com/github/bsideup/liiklus/GroupVersionTest.java
```

### Authentication Parameters

Not supported yet.

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