# Apache Kafka Topic Trigger

This specification describes the `kafka` trigger for Apache Kafka Topic.

```yaml
  triggers:
  - type: kafka
    metadata:
      brokerList: kafka.svc:9092
      consumerGroup: my-group
      topic: test-topic
      lagThreshold: '5' # Optional. How much the stream is lagging on the current consumer group
```

## Example

[`examples/kafka_scaledobject.yaml`](./../../examples/kafka_scaledobject.yaml)