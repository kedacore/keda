# AWS Cloudwatch Trigger

This specification describes the `aws-cloudwatch` trigger for AWS Cloudwatch.

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
```

The `namespace` value is the service you want to get metrics for, in this case 'AWS/SQS'. Many other options exist.

The `dimensionName` value is the selector criteria for which resource to monitor. 

The `dimensionValue` value is the value of the `dimensionName` you want to match.

The `metricName` value is the metric you want to measure, these are different between namespaces.

The `targetMetricValue` value is the average value you want to target.

## Example

[`examples/awscloudwatch_scaledobject.yaml`](./../../examples/awscloudwatch_scaledobject.yaml)

## Using TriggerAuthentication

This specification describes the `aws-role` TriggerAuthentication for AWS Cloudwatch.

```yaml
# trigger
triggers:
  - type: aws-cloudwatch
    authenticationRef:
      name: keda-trigger-auth-aws-role
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
```

```yaml
# auth object
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
metadata:
  name: keda-trigger-auth-aws-role
  namespace: keda-test
spec:
  podIdentity:
      provider: aws-role
  secretTargetRef:
  - parameter: awsRoleArn            # Required.
    name: keda-aws-secrets           # Required.
    key: AWS_ROLE_ARN                # Required.   
```

```yaml
# secret object for the TriggerAuthentication ref above
apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
  labels:
data:
  AWS_ROLE_ARN: Q29ubmVjdGlvbiBzdHJpbmcgdmFsdWUgaW4gYmFzZTY0IGVuY29kaW5nIGdvZXMgaGVyZQ==
```

This specification describes the `aws-credentials` TriggerAuthentication for AWS Cloudwatch.

```yaml
# trigger
triggers:
  - type: aws-cloudwatch
    authenticationRef:
      name: keda-trigger-auth-aws-credentials
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
```

```yaml
# auth object
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
metadata:
  name: keda-trigger-auth-aws-credentials
  namespace: keda-test
spec:
  podIdentity:
      provider: aws-credentials
  secretTargetRef:
  - parameter: awsAccessKeyID     # Required.
    name: keda-aws-secrets        # Required.
    key: AWS_ACCESS_KEY_ID        # Required.
  - parameter: awsSecretAccessKey # Required.
    name: keda-aws-secrets        # Required.
    key: AWS_SECRET_ACCESS_KEY    # Required.   
```

```yaml
# secret object for the TriggerAuthentication ref above
apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
  labels:
data:
  AWS_ACCESS_KEY_ID: Q29ubmVjdGlvbiBzdHJpbmcgdmFsdWUgaW4gYmFzZTY0IGVuY29kaW5nIGdvZXMgaGVyZQ==
  AWS_SECRET_ACCESS_KEY: Q29ubmVjdGlvbiBzdHJpbmcgdmFsdWUgaW4gYmFzZTY0IGVuY29kaW5nIGdvZXMgaGVyZQ==
```
