# Rabbit MQ Queue
Example: [`examples/rabbitmq_scaledobject.yaml`](./../../examples/azurequeue_scaledobject.yaml)

```yaml
  triggers:
  - type: rabbitmq
    metadata:
      host: amqp://guest:guest@rabbitmq.svc:5672/
      queueLength: '20' # Optional. Queue length target for HPA. Default: 20 messages
      queueName: testqueue
```
