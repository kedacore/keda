# Rabbit MQ Queue Trigger

This specification describes the `rabbitmq` trigger for Rabbit MQ Queue.

```yaml
  triggers:
  - type: rabbitmq
    metadata:
      host: amqp://guest:guest@rabbitmq.svc:5672/
      queueLength: '20' # Optional. Queue length target for HPA. Default: 20 messages
      queueName: testqueue
```

## Example

[`examples/rabbitmq_scaledobject.yaml`](./../../examples/rabbitmq_scaledobject.yaml)
