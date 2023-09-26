# AWS MSK IAM V2

This extension provides a capability to get authenticated with [AWS Managed Apache Kafka](https://aws.amazon.com/msk/)
through AWS IAM.

## How to use

This module is an extension for MSK users and thus this is isolated from `kafka-go` module.
You can add this module to your dependency by running the command below.

```shell
go get github.com/segmentio/kafka-go/sasl/aws_msk_iam_v2
```

Please find the sample code in [example_test.go](./example_test.go), you can use the `Mechanism` for SASL authentication of `Reader` and `Writer`.
