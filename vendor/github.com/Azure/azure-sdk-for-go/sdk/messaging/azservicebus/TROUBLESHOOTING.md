# Troubleshooting Azure Service Bus module issues

This troubleshooting guide contains instructions to diagnose frequently encountered issues while using the Azure Service Bus module for Go (`azservicebus`).

## Table of contents

- [General Troubleshooting](#general-troubleshooting)
    - [Error Handling](#error-handling)
    - [Logging](#logging)
- [Common Error Scenarios](#common-error-scenarios)
    - [Unauthorized Access Errors](#unauthorized-access-errors)
    - [Connection Lost Errors](#connection-lost-errors)
    - [Lock Lost Errors](#lock-lost-errors)
    - [Performance Considerations](#performance-considerations)
        - [Receiver](#receiver)
        - [Sender](#sender)
- [Connectivity Issues](#connectivity-issues)
    - [Enterprise Environments and Firewalls](#enterprise-environments-and-firewalls)
- [Advanced Troubleshooting](#advanced-troubleshooting)
    - [Logs to collect](#logs-to-collect)
    - [Interpreting Logs](#interpreting-logs)
    - [Additional Resources](#additional-resources)
    - [Filing GitHub Issues](#filing-github-issues)

---

## General Troubleshooting

### Error Handling

`azservicebus` can return two types of errors: `azservicebus.Error`, which contains a code you can use programmatically, and generic `error`s which only contain an error message.

Here's an example of how to check the `Code` from an `azservicebus.Error`:

```go
if err != nil {
    var sbErr *azservicebus.Error
    if errors.As(err, &sbErr) {
        switch sbErr.Code {
        case azservicebus.CodeUnauthorizedAccess:
            // Handle authentication errors
        case azservicebus.CodeConnectionLost:
            // Handle connection lost errors
        case azservicebus.CodeNotFound:
            // Handle queue/topic/subscription not existing
        default:
            // Handle other error codes.
        }
    }
    // Handle other error types
}
```

### Logging

Service Bus uses the classification-based logging implementation in `azcore`. You can enable logging for all Azure SDK modules by setting the environment variable `AZURE_SDK_GO_LOGGING` to `all`.

For more fine-grained control, use the `azcore/log` package to enable specific log events:

```go
import (
    "fmt"

    azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
    "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// Print log output to stdout
azlog.SetListener(func(event azlog.Event, s string) {
    fmt.Printf("[%s] %s\n", event, s)
})

// Enable specific event types
azlog.SetEvents(
    azservicebus.EventConn,    // Connection-related events
    azservicebus.EventAuth,    // Authentication events
    azservicebus.EventSender,  // Sender operations
    azservicebus.EventReceiver, // Receiver operations
    azservicebus.EventAdmin,     // operations in the azservicebus/admin.Client
)
```

---

## Common Error Scenarios

### Unauthorized Access Errors

If you receive a `CodeUnauthorizedAccess` error, it means the credentials provided are not valid for use with a particular entity, or they have expired.

**Common causes and solutions:**

- **Expired credentials**: If using SAS tokens, they expire after a certain duration. Generate a new token or use a credential that automatically refreshes, like one of the TokenCredential types from the [Azure Identity module][azidentity_tokencredentials].
- **Missing permissions**: Ensure the identity you're using has the correct role assigned from the [built-in roles for Azure Service Bus](https://learn.microsoft.com/azure/service-bus-messaging/service-bus-authentication-and-authorization#microsoft-entra-id).
- **Incorrect entity name**: Verify that the queue, topic, or namespace name is spelled correctly.

For more help with troubleshooting authentication errors when using Azure Identity, see the Azure Identity client library [troubleshooting guide][azidentity_troubleshooting].

### Connection Lost Errors

An `azservicebus.CodeConnectionLost` error indicates that the connection was lost and all retry attempts failed. This typically reflects an extended outage or connection disruption.

**Common causes and solutions:**

- **Network instability**: Check your network connection and try again after ensuring stability.
- **Service outage**: Check the [Azure status page](https://status.azure.com) for any ongoing Service Bus outages.
- **Firewall or proxy issues**: Ensure firewall rules aren't blocking the connection.

### Lock Lost Errors

A `CodeLockLost` error occurs when the lock on a session or message is lost due to exceeding the lock duration. If you find this happening, you can either:
- Increase the [LockDuration](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus@v1.9.0/admin#QueueProperties) for your queue or subscription.
- Call `Receiver.RenewMessageLock` or `SessionReceiver.RenewSessionLock`. Sample code for automatic lock renewal can be found here: [Example_autoRenewLocks](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#example-package-AutoRenewLocks).

### Performance Considerations

#### Receiver

- **Process messages in multiple goroutines**: Receiver settlement methods, like CompleteMessage, are goroutine-safe and can be called concurrently, allowing message processing to run in multiple goroutines.
- **Increase client instances**: Add more receiver clients to distribute the load. The number of concurrent receivers for a session-enabled entity is limited by the number of sessions.

#### Sender

- **Use message batching**: [Sender.NewMessageBatch](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#Sender.NewMessageBatch) and [Sender.SendMessageBatch](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#Sender.SendMessageBatch) methods allow you to send batches of messages, reducing network traffic between the service and the client.

---

## Connectivity Issues

### Enterprise Environments and Firewalls

In corporate networks with strict firewall rules, you may encounter connectivity issues when connecting to Service Bus.

**Common solutions:**

1. **Allow the necessary endpoints**: See [Service Bus FAQ: "What ports do I need to open on the firewall?"](https://learn.microsoft.com/azure/service-bus-messaging/service-bus-faq#what-ports-do-i-need-to-open-on-the-firewall).
2. **Use a proxy**: If you require a proxy to connect to Azure resources you can configure your client to use it. See [example using a proxy and/or Websockets][example_proxy_websockets].
3. **Use Websockets**: If you can only connect to Azure resources using HTTPS (443) you can configure your client to use Websockets. See [example using a proxy and/or Websockets][example_proxy_websockets].
4. **Configure network security rules**: If using Azure VNet integration, configure service endpoints or private endpoints as needed.

---

## Advanced Troubleshooting

### Logs to collect

When troubleshooting issues with Service Bus that you need to escalate to support or report in GitHub issues, collect the following logs:

1. **Enable debug logging**: To enable logs, see [logging](#logging).
2. **Timeframe**: Capture logs from at least 5 minutes before until 5 minutes after the issue occurs.
3. **Include timestamps**: Ensure your logging setup includes timestamps. By default, `AZURE_SDK_GO_LOGGING` logging includes timestamps.

### Interpreting Logs

When analyzing Service Bus logs:

1. **Connection errors**: Look for AMQP connection and link errors in `EventConn` logs.
2. **Authentication failures**: Check `EventAuth` logs for credential or authorization failures.
3. **Sender errors**: `EventSender` logs show message send operations and errors.
4. **Receiver errors**: `EventReceiver` logs show message receive operations and session/message lock changes.

### Additional Resources

- [Service Bus Documentation](https://learn.microsoft.com/azure/service-bus-messaging/)
- [Service Bus Pricing](https://azure.microsoft.com/pricing/details/service-bus/)
- [Service Bus Quotas](https://learn.microsoft.com/azure/service-bus-messaging/service-bus-quotas)
- [Service Bus FAQ](https://learn.microsoft.com/azure/service-bus-messaging/service-bus-faq)

### Filing GitHub Issues

To file an issue in GitHub, use this [link](https://github.com/Azure/azure-sdk-for-go/issues/new/choose) and include the following information:

1. **Service Bus details**:
   - Session-enabled?
   - SKU (Standard/Premium)
2. **Client environment**:
   - Machine specifications
   - Number of client instances running
   - Package version
   - Go version
3. **Message patterns**:
   - Average message size
   - Throughput (messages per second)
   - Whether traffic is consistent or bursty
4. **Reproduction steps**:
   - A minimal code example that reproduces the issue
   - Steps to reproduce the problem
5. **Logs**:
   - Include diagnostic logs from before, during, and after the failure. For instructions on enabling logging see the [Logs to collect](#logs-to-collect) section above.

**NOTE**: The information in Github issues and logs are publicly viewable. Please keep this in mind when posting any information.

<!-- LINKS -->
[azidentity_troubleshooting]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity/TROUBLESHOOTING.md
[azidentity_tokencredentials]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#readme-credential-chains
[example_proxy_websockets]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azservicebus/example_websockets_and_proxies_test.go
