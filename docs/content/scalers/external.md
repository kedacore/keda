+++
fragment = "content"
weight = 100
title = "External"
background = "light"
+++

Scale applications based on an external scaler.

* **Availability:** v1.0 and above
* **Maintainer:** Microsoft

<!--more-->

### Trigger Specification

This specification describes the `external` trigger for an external scaler.

```yaml
triggers:
  - type: external
    metadata:
      scalerAddress: redis-external-scaler-service:8080
      address: REDIS_HOST # Required host:port format
      password: REDIS_PASSWORD
      listName: mylist # Required
      listLength: "5" # Required
```

### Authentication Parameters

Not supported yet.

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
  - type: external
    metadata:
      scalerAddress: redis-external-scaler-service:8080
      address: REDIS_HOST
      password: REDIS_PASSWORD
      listName: mylist
      listLength: "5"
```