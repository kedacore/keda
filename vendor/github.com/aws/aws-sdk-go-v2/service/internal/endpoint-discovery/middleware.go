package endpointdiscovery

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/aws/smithy-go/logging"
	"github.com/aws/smithy-go/middleware"
)

// DiscoverEndpointOptions are optionals used with DiscoverEndpoint operation.
type DiscoverEndpointOptions struct {

	// EndpointResolverUsedForDiscovery is the endpoint resolver used to
	// resolve an endpoint for discovery api call.
	EndpointResolverUsedForDiscovery interface{}

	// DisableHTTPS will disable tls for endpoint discovery call and
	// subsequent discovered endpoint if service did not return an
	// endpoint scheme.
	DisableHTTPS bool

	// Logger to log warnings or debug statements.
	Logger logging.Logger
}

// DiscoverEndpoint is a serialize step middleware used to discover endpoint
// for an API operation.
type DiscoverEndpoint struct {

	// Options provides optional settings used with
	// Discover Endpoint operation.
	Options []func(*DiscoverEndpointOptions)

	// DiscoverOperation represents the endpoint discovery operation that
	// returns an Endpoint or error.
	DiscoverOperation func(ctx context.Context, input interface{}, options ...func(*DiscoverEndpointOptions)) (WeightedAddress, error)

	// EndpointDiscoveryEnableState represents the customer configuration for endpoint
	// discovery feature.
	EndpointDiscoveryEnableState aws.EndpointDiscoveryEnableState

	// EndpointDiscoveryRequired states if an operation requires to perform
	// endpoint discovery.
	EndpointDiscoveryRequired bool
}

// ID represents the middleware identifier
func (*DiscoverEndpoint) ID() string {
	return "DiscoverEndpoint"
}

// HandleSerialize is the serialize step function handler that must be placed after
// "ResolveEndpoint" middleware, but before "OperationSerializer" middleware.
func (d *DiscoverEndpoint) HandleSerialize(
	ctx context.Context, in middleware.SerializeInput, next middleware.SerializeHandler,
) (out middleware.SerializeOutput, metadata middleware.Metadata, err error) {
	// if endpoint discovery is explicitly disabled, skip this workflow
	if d.EndpointDiscoveryEnableState == aws.EndpointDiscoveryDisabled {
		return next.HandleSerialize(ctx, in)
	}

	// if operation does not require endpoint discovery, and endpoint discovery is not explicitly enabled,
	// skip this workflow
	if !d.EndpointDiscoveryRequired && d.EndpointDiscoveryEnableState != aws.EndpointDiscoveryEnabled {
		return next.HandleSerialize(ctx, in)
	}

	// when custom endpoint is provided
	if es := awsmiddleware.GetEndpointSource(ctx); es == aws.EndpointSourceCustom {
		// error if endpoint discovery was explicitly enabled
		if d.EndpointDiscoveryEnableState == aws.EndpointDiscoveryEnabled {
			return middleware.SerializeOutput{}, middleware.Metadata{},
				fmt.Errorf("Invalid configuration: endpoint discovery is enabled, but a custom endpoint is provided")
		}

		// else skip this workflow
		return next.HandleSerialize(ctx, in)
	}

	// fetch address using discover operation
	weightedAddress, err := d.DiscoverOperation(ctx, in.Parameters, d.Options...)
	if err != nil {
		return middleware.SerializeOutput{}, middleware.Metadata{}, err
	}

	req, ok := in.Request.(*smithyhttp.Request)
	if !ok {
		return middleware.SerializeOutput{}, middleware.Metadata{},
			fmt.Errorf("expected request to be of type *smithyhttp.Request, got %T", in.Request)
	}

	if weightedAddress.URL != nil {
		// assign discovered endpoint to request url
		req.URL = weightedAddress.URL
	}

	return next.HandleSerialize(ctx, in)
}
