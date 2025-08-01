package proxy

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func (s *workflowServiceProxyServer) reqCtx(ctx context.Context) context.Context {
	if s.disableHeaderForwarding {
		return ctx
	}

	// Copy incoming header to outgoing if not already present in outgoing. We
	// have confirmed in gRPC that incoming is a copy so we can mutate it.
	incoming, _ := metadata.FromIncomingContext(ctx)

	// Remove common headers and if there's nothing left, return early
	incoming.Delete("user-agent")
	incoming.Delete(":authority")
	incoming.Delete("content-type")
	if len(incoming) == 0 {
		return ctx
	}

	// Put all incoming on outgoing if they are not already there. We have
	// confirmed in gRPC that outgoing is a copy so we can mutate it.
	outgoing, _ := metadata.FromOutgoingContext(ctx)
	if outgoing == nil {
		outgoing = metadata.MD{}
	}
	for k, v := range incoming {
		if len(outgoing.Get(k)) == 0 {
			outgoing.Set(k, v...)
		}
	}
	return metadata.NewOutgoingContext(ctx, outgoing)
}
