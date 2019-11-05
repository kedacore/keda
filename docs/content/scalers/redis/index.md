+++
fragment = "content"
weight = 100
title = "Redis Lists"
background = "light"
+++

Scale applications based on Redis Lists.

<!--more-->

* **Availability:** v1.0 and above
* **Maintainer:** Community

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: redis-scaledobject
  namespace: keda-redis-test
  labels:
    deploymentName: keda-redis-node
spec:
  scaleTargetRef:
    deploymentName: keda-redis-node
  triggers:
  - type: redis
    metadata:
      address: REDIS_HOST # Required host:port format
      password: REDIS_PASSWORD
      listName: mylist # Required
      listLength: "5" # Required
```