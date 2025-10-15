package converter

import (
	"fmt"

	"google.golang.org/grpc"

	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/proxy"
)

// PayloadCodecGRPCClientInterceptorOptions holds interceptor options.
// Currently this is just the list of codecs to use.
type PayloadCodecGRPCClientInterceptorOptions struct {
	Codecs []PayloadCodec
}

// NewPayloadCodecGRPCClientInterceptor returns a GRPC Client Interceptor that will mimic the encoding
// that the SDK system would perform when configured with a matching EncodingDataConverter.
// When combining this with NewFailureGRPCClientInterceptor you should ensure that NewFailureGRPCClientInterceptor is
// before NewPayloadCodecGRPCClientInterceptor in the chain.
//
// Note: This approach does not support use cases that rely on the ContextAware DataConverter interface as
// workflow context is not available at the GRPC level.
func NewPayloadCodecGRPCClientInterceptor(options PayloadCodecGRPCClientInterceptorOptions) (grpc.UnaryClientInterceptor, error) {
	return proxy.NewPayloadVisitorInterceptor(proxy.PayloadVisitorInterceptorOptions{
		Outbound: &proxy.VisitPayloadsOptions{
			Visitor: func(vpc *proxy.VisitPayloadsContext, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
				var err error
				for i := len(options.Codecs) - 1; i >= 0; i-- {
					if payloads, err = options.Codecs[i].Encode(payloads); err != nil {
						return payloads, err
					}
				}

				return payloads, nil
			},
			SkipSearchAttributes: true,
		},
		Inbound: &proxy.VisitPayloadsOptions{
			Visitor: func(vpc *proxy.VisitPayloadsContext, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
				var err error
				for _, codec := range options.Codecs {
					if payloads, err = codec.Decode(payloads); err != nil {
						return payloads, err
					}
				}

				return payloads, nil
			},
			SkipSearchAttributes: true,
		},
	})
}

// NewFailureGRPCClientInterceptorOptions holds interceptor options.
type NewFailureGRPCClientInterceptorOptions struct {
	// DataConverter is optional. If not set the SDK's dataconverter will be used.
	DataConverter DataConverter
	// Whether to Encode attributes. The current implementation requires this be true.
	EncodeCommonAttributes bool
}

// NewFailureGRPCClientInterceptor returns a GRPC Client Interceptor that will mimic the encoding
// that the SDK system would perform when configured with a FailureConverter with the EncodeCommonAttributes option set.
// When combining this with NewPayloadCodecGRPCClientInterceptor you should ensure that NewFailureGRPCClientInterceptor is
// before NewPayloadCodecGRPCClientInterceptor in the chain.
func NewFailureGRPCClientInterceptor(options NewFailureGRPCClientInterceptorOptions) (grpc.UnaryClientInterceptor, error) {
	if !options.EncodeCommonAttributes {
		return nil, fmt.Errorf("EncodeCommonAttributes must be set for this interceptor to function")
	}

	dc := options.DataConverter
	if dc == nil {
		dc = GetDefaultDataConverter()
	}

	return proxy.NewFailureVisitorInterceptor(proxy.FailureVisitorInterceptorOptions{
		Outbound: &proxy.VisitFailuresOptions{
			Visitor: func(vpc *proxy.VisitFailuresContext, failure *failurepb.Failure) error {
				return EncodeCommonFailureAttributes(dc, failure)
			},
		},
		Inbound: &proxy.VisitFailuresOptions{
			Visitor: func(vpc *proxy.VisitFailuresContext, failure *failurepb.Failure) error {
				DecodeCommonFailureAttributes(dc, failure)

				return nil
			},
		},
	})
}
