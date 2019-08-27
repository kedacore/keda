# Redis Trigger

This specification describes the `redis` trigger.

```yaml
  triggers:
  - type: redis
    metadata:
      address: REDIS_HOST # Required host:port format
      password: REDIS_PASSWORD
      listName: mylist # Required
      listLength: "5" # Required
```

This trigger scales based on the length of a list in Redis. The **address** field in the spec holds the host and port of the redis server. This could be an external redis server or one running in the kubernetes cluster. Provide the **password** field if the redis server requires a password. Both the hostname and password fields need to be set to the names of the environment variables in the target deployment that contain the host name and password respectively.

The **listName** parameter in the spec points to the Redis List that you want to monitor. The **listLength** parameter defines the average target value for the Horizontal Pod Autoscaler (HPA).


## Example

[`examples/azureeventhub_scaledobject.yaml`](./../../examples/redis_scaledobject.yaml)