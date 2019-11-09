+++
fragment = "content"
weight = 100
title = "AWS SQS Queue"
background = "light"
+++

Scale applications based on AWS SQS Queue.

* **Availability:** v1.0 and above
* **Maintainer:** Community

<!--more-->

### Trigger Specification

This specification describes the `aws-sqs-queue` trigger for AWS SQS Queue.

```yaml
triggers:
  - type: aws-sqs-queue
    metadata:
      # Required: queueURL
      queueURL: https://sqs.eu-west-1.amazonaws.com/<acccount_id>/testQueue
      # Optional: region
      awsRegion: "eu-west-1"
      # Optional: AWS Access Key ID
      awsAccessKeyID: AWS_ACCESS_KEY_ID_ENV_VAR # default AWS_ACCESS_KEY_ID
      # Optional: AWS Secret Access Key
      awsSecretAccessKey: AWS_SECRET_ACCESS_KEY_ENV_VAR # default AWS_SECRET_ACCESS_KEY
      # Optional
      queueLength: "5" # default 5
```

### Authentication Parameters

Not supported yet.

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: aws-sqs-queue-scaledobject
  namespace: default
  labels:
    deploymentName: nginx-deployment
    test: nginx-deployment
spec:
  scaleTargetRef:
    deploymentName: nginx-deployment
  triggers:
  - type: aws-sqs-queue
    metadata:
      queueURL: https://sqs.eu-west-1.amazonaws.com/<acccount_id>/testQueue
      awsRegion: "eu-west-1"
      awsAccessKeyID: AWS_ACCESS_KEY_ID_ENV_VAR
      awsSecretAccessKey: AWS_SECRET_ACCESS_KEY_ENV_VAR
      queueLength: "5"
```