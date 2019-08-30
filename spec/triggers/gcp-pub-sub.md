#  Google Cloud Platform‎ Pub/Sub

This specification describes the `gcp-pub-sub` trigger.

```yaml
triggers:
- type: gcp-pubsub
  metadata:
    subscriptionSize: "5" # Optional - Default is 5
    subscriptionName: "mysubscription" # Required 
    credentials: GOOGLE_APPLICATION_CREDENTIALS_JSON # Required
```

The Google Cloud Platform‎ (GCP) Pub/Sub trigger allows you to scale based on the number of messages in your Pub/Sub subscription.

The `credentials` property maps to the name of an environment variable in the scale target (`scaleTargetRef`) that contains the service account credentials (JSON). KEDA will use those to connect to Google Cloud Platform and collect the required stack driver metrics in order to read the number of messages in the Pub/Sub subscription.

`subscriptionName` defines the subscription that should be monitored. The `subscriptionSize` determines the target average which the deployment will be scaled on. The default `subscriptionSize` is 5.

## Example

[`examples/gcppubsub_scaledobject.yaml`](./../../examples/gcppubsub_scaledobject.yaml)
