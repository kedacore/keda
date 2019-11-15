# AWS SQS queue Trigger

This specification describes the `aws-sqs-queue` trigger for AWS SQS queue.

```yaml
triggers:
- type: aws-sqs-queue
  authenticationRef: 
    name: keda-trigger-auth-aws-role
  metadata:
    # Required: queueURL
    queueURL: myQueue
    queueLength: "5"  # Default: "5"
    # Required: awsRegion
    awsRegion: "eu-west-1" 
```

- `queueURL` value is the name of the SQS Queue you want to monitor
- `queueLength` value is the average value you want to target.

## Example

[`examples/awssqs-queue_scaledobject.yaml`](./../../examples/awssqs-queue_scaledobject.yaml)

## Using TriggerAuthentication

Authentication can be handled by providing either a role ARN or a set of IAM credentials. The user will need access to read data from AWS Sqs-queue.

### Role based authentication

```yaml
# auth object
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-role
  namespace: keda-test
spec:
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

```yaml
# trigger
triggers:
- type: aws-sqs-queue
  authenticationRef: 
    name: keda-trigger-auth-aws-role
  metadata:
    # Required: queueURL
    queueURL: myQueue
    queueLength: "5"  # Default: "5"
    # Required: awsRegion
    awsRegion: "eu-west-1"  
```

### Credential based authentication

```yaml
# auth object
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-credentials
  namespace: keda-test
spec:
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
data:
  AWS_ACCESS_KEY_ID: Q29ubmVjdGlvbiBzdHJpbmcgdmFsdWUgaW4gYmFzZTY0IGVuY29kaW5nIGdvZXMgaGVyZQ==
  AWS_SECRET_ACCESS_KEY: Q29ubmVjdGlvbiBzdHJpbmcgdmFsdWUgaW4gYmFzZTY0IGVuY29kaW5nIGdvZXMgaGVyZQ==
```
```yaml
# trigger
triggers:
- type: aws-sqs-queue
  authenticationRef: 
    name: keda-trigger-auth-aws-credentials
  metadata:
    # Required: queueURL
    queueURL: myQueue
    queueLength: "5"  # Default: "5"
    # Required: awsRegion
    awsRegion: "eu-west-1" 
```
