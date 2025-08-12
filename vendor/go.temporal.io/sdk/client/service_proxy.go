package client

import (
	"go.temporal.io/api/proxy"
	"go.temporal.io/api/workflowservice/v1"
)

// WorkflowServiceProxyOptions provides options for configuring a WorkflowServiceProxyServer.
// Client is a WorkflowServiceClient used to forward requests received by the server to the
// Temporal Frontend.
type WorkflowServiceProxyOptions struct {
	Client workflowservice.WorkflowServiceClient
	// DisableHeaderForwarding disables the forwarding of headers from the incoming request to the outgoing request.
	DisableHeaderForwarding bool
}

// NewWorkflowServiceProxyServer creates a WorkflowServiceServer suitable for registering with a GRPC Server. Requests will
// be forwarded to the passed in WorkflowService Client. GRPC interceptors can be added on the Server or Client to adjust
// requests and responses.
func NewWorkflowServiceProxyServer(options WorkflowServiceProxyOptions) (workflowservice.WorkflowServiceServer, error) {
	return proxy.NewWorkflowServiceProxyServer(proxy.WorkflowServiceProxyOptions{
		// These options are expected to be kept mostly in sync, but we can't do a
		// naive type conversion because we want users to be able to update to newer
		// API library versions with older SDK versions.
		Client:                  options.Client,
		DisableHeaderForwarding: options.DisableHeaderForwarding,
	})
}
