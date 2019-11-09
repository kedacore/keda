+++
fragment = "content"
weight = 100
title = "AWS Cloudwatch"
background = "light"
+++

Scale applications based on a AWS Cloudwatch.

* **Availability:** v1.0 and above
* **Maintainer:** Community

<!--more-->

### Trigger Specification

This specification describes the `aws-cloudwatch` trigger that scales based on a AWS Cloudatch.

```yaml
triggers:
  - type: aws-cloudwatch
    metadata:
      # Required: namespace
      namespace: AWS/SQS
      # Required: Dimension Name
      dimensionName: QueueName
      dimensionValue: keda
      metricName: ApproximateNumberOfMessagesVisible
      targetMetricValue: "2"
      minMetricValue: "0"
      # Required: region
      awsRegion: "eu-west-1"
      # Optional: AWS Access Key ID
      awsAccessKeyID: AWS_ACCESS_KEY_ID # default AWS_ACCESS_KEY_ID
      # Optional: AWS Secret Access Key
      awsSecretAccessKey: AWS_SECRET_ACCESS_KEY # default AWS_SECRET_ACCESS_KEY
```

### Authentication Parameters

Not supported yet.

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: aws-cloudwatch-queue-scaledobject
  namespace: keda-test
  labels:
    deploymentName: nginx-deployment
    test: nginx-deployment
spec:
  scaleTargetRef:
    deploymentName: nginx-deployment
  triggers:
  - type: aws-cloudwatch
    metadata:
      namespace: AWS/SQS
      dimensionName: QueueName
      dimensionValue: keda
      metricName: ApproximateNumberOfMessagesVisible
      targetMetricValue: "2"
      minMetricValue: "0"
      awsRegion: "eu-west-1"
      awsAccessKeyID: AWS_ACCESS_KEY_ID
      awsSecretAccessKey: AWS_SECRET_ACCESS_KEY
```