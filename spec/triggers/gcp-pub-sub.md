#  GCP PubSub Trigger

This specification describes the `gcp-pub-sub` trigger.

```yaml
triggers:
- type: gcp-pubsub
  metadata:
    subscriptionSize: "5"
    subscriptionName: "mysubscription" # Required 
    credentials: GOOGLE_APPLICATION_CREDENTIALS_JSON # Required
```

The GCP PubSub trigger allows you to scale based on the number of messages in your GCP PubSub subscription. The **credentials** property maps to the name of environment variable in the scale target (deployment) which contains the service account credentials (JSON) that would allow KEDA to connect to GCP and collect the required stack driver metrics to read the number of messages in the PubSub subscription. The **subscriptionSize** will be set as the target average value for the HPA and the **subscriptionName** is the subscription to be monitored.

## Example

[`examples/gcppubsub_scaledobject.yaml`](./../../examples/gcppubsub_scaledobject.yaml)
