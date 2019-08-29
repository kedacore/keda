# Liiklus Topic Trigger

This specification describes the `liiklus` trigger for [Liiklus](https://github.com/bsideup/liiklus).

```yaml
  triggers:
  - type: liiklus
    metadata:
      # Required
      address: localhost:6565 # Address of the gRPC liiklus API endpoint
      group: my-group         # Make sure that this consumer group name is the same one as the one that is consuming topics
      topic: test-topic
      # Optional
      lagThreshold: "50"      # default 10, the target lag for HPA
      groupVersion: 1         # default 0, the groupVersion to consider when looking at messages. See https://github.com/bsideup/liiklus/blob/22efb7049ebcdd0dcf6f7f5735cdb5af1ae014de/app/src/test/java/com/github/bsideup/liiklus/GroupVersionTest.java
```

## Example

[`examples/liiklus_scaledobject.yaml`](./../../examples/liiklus_scaledobject.yaml)