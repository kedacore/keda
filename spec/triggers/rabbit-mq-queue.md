# Rabbit MQ Queue Trigger

This specification describes the `rabbitmq` trigger for Rabbit MQ Queue.

```yaml
  triggers:
  - type: rabbitmq
    metadata:
      host: RabbitMqHost
      queueLength: '20' # Optional. Queue length target for HPA. Default: 20 messages
      queueName: testqueue
```

The `host` value is the name of the environment variable your deployment uses to get the connection string. This is usually resolved from a `Secret V1` or a `ConfigMap V1` collections. `env` and `envFrom` are both supported.  The resolved host should follow a format like `amqp://guest:password@localhost:5672/`

## Example

[`examples/rabbitmq_scaledobject.yaml`](./../../examples/rabbitmq_scaledobject.yaml)
