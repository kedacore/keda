# Troubleshooting Azure Event Hubs module issues

This troubleshooting guide contains instructions to diagnose frequently encountered issues while using the Azure Event Hubs module for Go.

## Table of contents

- [General Troubleshooting](#general-troubleshooting)
    - [Error Handling](#error-handling)
    - [Logging](#logging)
- [Common Error Scenarios](#common-error-scenarios)
    - [Unauthorized Access Errors](#unauthorized-access-errors)
    - [Connection Lost Errors](#connection-lost-errors)
    - [Ownership Lost Errors](#ownership-lost-errors)
    - [Performance Considerations](#performance-considerations)
- [Connectivity Issues](#connectivity-issues)
    - [Enterprise Environments and Firewalls](#enterprise-environments-and-firewalls)
- [Advanced Troubleshooting](#advanced-troubleshooting)
    - [Logs to collect](#logs-to-collect)
    - [Interpreting Logs](#interpreting-logs)
    - [Additional Resources](#additional-resources)
    - [Filing GitHub Issues](#filing-github-issues)

## General Troubleshooting

### Error Handling

azeventhubs can return two types of errors: `azeventhubs.Error`, which contains a code you can use programatically, and `error`s which only contain an error message.

Here's an example of how to check the `Code` from an `azeventhubs.Error`:

```go
if err != nil {
    var azehErr *azeventhubs.Error
    
    if errors.As(err, &azehErr) {
        switch azehErr.Code {
        case azeventhubs.ErrorCodeUnauthorizedAccess:
            // Handle authentication errors
        case azeventhubs.ErrorCodeConnectionLost:
            // This error is only returned if all configured retries have been exhausted.
            // An example of configuring retries can be found here: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2#example-NewConsumerClient-ConfiguringRetries
        }
    }

    // Handle other error types
}
```

### Logging

Event Hubs uses the classification-based logging implementation in `azcore`. You can enable logging for all Azure SDK modules by setting the environment variable `AZURE_SDK_GO_LOGGING` to `all`.

For more fine-grained control, use the `azcore/log` package to enable specific log events:

```go
import (
    "fmt"
    azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
    "github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
)

// Print log output to stdout
azlog.SetListener(func(event azlog.Event, s string) {
    fmt.Printf("[%s] %s\n", event, s)
})

// Enable specific event types
azlog.SetEvents(
    azeventhubs.EventConn,    // Connection-related events
    azeventhubs.EventAuth,    // Authentication events
    azeventhubs.EventProducer, // Producer operations
    azeventhubs.EventConsumer, // Consumer operations
)
```

## Common Error Scenarios

### Unauthorized Access Errors

If you receive an `ErrorCodeUnauthorizedAccess` error, it means the credentials provided are not valid for use with a particular entity, or they have expired.

**Common causes and solutions:**

- **Expired credentials**: If using SAS tokens, they expire after a certain duration. Generate a new token or use a credential that automatically refreshes, like one of the TokenCredential types from the [Azure Identity module][azidentity_tokencredentials].
- **Missing permissions**: Ensure the identity you're using has the correct role assigned from the [built-in roles for Azure Event Hubs](https://learn.microsoft.com/azure/event-hubs/authenticate-application#built-in-roles-for-azure-event-hubs).
- **Incorrect entity name**: Verify that the Event Hub name, consumer group, or namespace name is spelled correctly.

For more help with troubleshooting authentication errors when using Azure Identity, see the Azure Identity client library [troubleshooting guide][azidentity_troubleshooting].

### Connection Lost Errors

An `azeventhubs.ErrorCodeConnectionLost` error indicates that the connection was lost and all retry attempts failed. This typically reflects an extended outage or connection disruption.

**Common causes and solutions:**

- **Network instability**: Check your network connection and try again after ensuring stability.
- **Service outage**: Check the [Azure status page](https://status.azure.com) for any ongoing Event Hubs outages.
- **Firewall or proxy issues**: Ensure firewall rules aren't blocking the connection.

### Ownership Lost Errors

An `azeventhubs.ErrorCodeOwnershipLost` error occurs when a partition that you were reading from was opened by another link with a higher epoch/owner level.

* If you're using the azeventhubs.Processor, you will occasionally see this error when the individual Processors are allocating partition ownerships. This is expected, and the Processors will handle the error, internally.
* If you're NOT using the Processor, this indicates you have two PartitionClient instances, both of which are using the same consumer group, opening the same partition, but with different owner levels.

### Performance Considerations

**If the processor can't keep up with event flow:**

1. **Increase processor instances**: Add more Processor instances to distribute the load. The number of Processor instances cannot exceed the number of partitions for your Event Hub.
2. **Increase Event Hubs partitions**: Consider creating an Event Hub with more partitions, to allow for more parallel consumers. NOTE: requires a new Event Hub.
3. **Call `ProcessorPartitionClient.UpdateCheckpoint` less often**: some alternate strategies: 
    - Call only after a requisite number of events has been received
    - Call only after a certain amount of time has expired.

## Connectivity Issues

### Enterprise Environments and Firewalls

In corporate networks with strict firewall rules, you may encounter connectivity issues when connecting to Event Hubs.

**Common solutions:**

1. **Allow the necessary endpoints**: See [Event Hubs FAQ: "What ports do I need to open on the firewall?"][eventhubs_faq_ports].
2. **Use a proxy**: If you require a proxy to connect to Azure resources you can configure your client to use it: [Example using a proxy and/or Websockets][example_proxy_websockets]
3. **Use Websockets**: If you can only connect to Azure resources using HTTPs (443) you can configure your client to use Websockets. See this example for how to enable websockets with Event Hubs: [Example using a proxy and/or Websockets][example_proxy_websockets].
4. **Configure network security rules**: If using Azure VNet integration, configure service endpoints or private endpoints

## Advanced Troubleshooting

### Logs to collect

When troubleshooting issues with Event Hubs that you need to escalate to support or report in GitHub issues, collect the following logs:

1. **Enable debug logging**: To enable logs, see [logging](#logging).
2. **Timeframe**: Capture logs from at least 5 minutes before until 5 minutes after the issue occurs
3. **Include timestamps**: Ensure your logging setup includes timestamps. By default `AZURE_SDK_GO_LOGGING` logging includes timestamps.

### Interpreting Logs

When analyzing Event Hubs logs:

1. **Connection errors**: Look for AMQP connection and link errors in `EventConn` logs
2. **Authentication failures**: Check `EventAuth` logs for credential or authorization failures
3. **Producer errors**: `EventProducer` logs show message send operations and errors
4. **Consumer errors**: `EventConsumer` logs show message receive operations and partition ownership changes
5. **Load balancing**: Look for ownership claims and changes in `EventConsumer` logs

### Additional Resources

- [Event Hubs Documentation](https://learn.microsoft.com/azure/event-hubs/)
- [Event Hubs Pricing](https://azure.microsoft.com/pricing/details/event-hubs/)
- [Event Hubs Quotas](https://learn.microsoft.com/azure/event-hubs/event-hubs-quotas)
- [Event Hubs FAQ](https://learn.microsoft.com/azure/event-hubs/event-hubs-faq)

### Filing GitHub Issues

To file an issue in Github, use this [link](https://github.com/Azure/azure-sdk-for-go/issues/new/choose) and include the following information:

1. **Event Hub details**:
   - How many partitions?
   - What tier (Standard/Premium/Dedicated)?

2. **Client environment**:
   - Machine specifications
   - Number of client instances running
   - Go version

3. **Message patterns**:
   - Average message size
   - Throughput (messages per second)
   - Whether traffic is consistent or bursty

4. **Reproduction steps**:
   - A minimal code example that reproduces the issue
   - Steps to reproduce the problem

5. **Logs**:
   - Include diagnostic loogs from before, during and after the failure. For instructions on enabling logging see the [Logging](#logs-to-collect) section above.
   - **NOTE**: the information in Github issues and logs are publicly viewable. Please keep this in mind when posting any information.

<!-- LINKS -->
[azidentity_troubleshooting]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/azidentity/TROUBLESHOOTING.md
[amqp_errors]: https://learn.microsoft.com/azure/event-hubs/event-hubs-amqp-troubleshoot
[azidentity_tokencredentials]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#readme-credential-chains
[eventhubs_faq_ports]: https://learn.microsoft.com/azure/event-hubs/event-hubs-faq#what-ports-do-i-need-to-open-on-the-firewall
[example_proxy_websockets]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_websockets_and_proxies_test.go
