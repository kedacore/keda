package kinesis

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// DescribeStreamPaginatorOptions is the paginator options for DescribeStream
type DescribeStreamPaginatorOptions struct {
	// (Optional) The maximum number of shards to return in a single call
	Limit *int32

	// Set to true if pagination should stop if the service returns a pagination token
	// that matches the most recent token provided to the service.
	StopOnDuplicateToken bool
}

// DescribeStreamPaginator is a paginator for DescribeStream
type DescribeStreamPaginator struct {
	options               DescribeStreamPaginatorOptions
	client                DescribeStreamAPIClient
	params                *DescribeStreamInput
	firstPage             bool
	exclusiveStartShardID *string
	isTruncated           *bool
}

// NewDescribeStreamPaginator returns a new DescribeStreamPaginator
func NewDescribeStreamPaginator(client DescribeStreamAPIClient, params *DescribeStreamInput, optFns ...func(*DescribeStreamPaginatorOptions)) *DescribeStreamPaginator {
	if params == nil {
		params = &DescribeStreamInput{}
	}

	options := DescribeStreamPaginatorOptions{}
	options.Limit = params.Limit

	for _, fn := range optFns {
		fn(&options)
	}

	return &DescribeStreamPaginator{
		options:               options,
		client:                client,
		params:                params,
		firstPage:             true,
		exclusiveStartShardID: params.ExclusiveStartShardId,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *DescribeStreamPaginator) HasMorePages() bool {
	return p.firstPage || *p.isTruncated
}

// NextPage retrieves the next DescribeStream page.
func (p *DescribeStreamPaginator) NextPage(ctx context.Context, optFns ...func(*Options)) (*DescribeStreamOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.ExclusiveStartShardId = p.exclusiveStartShardID

	var limit *int32
	if *p.options.Limit > 0 {
		limit = p.options.Limit
	}
	params.Limit = limit

	result, err := p.client.DescribeStream(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.exclusiveStartShardID
	p.isTruncated = result.StreamDescription.HasMoreShards
	p.exclusiveStartShardID = nil
	if *result.StreamDescription.HasMoreShards {
		shardsLength := len(result.StreamDescription.Shards)
		p.exclusiveStartShardID = result.StreamDescription.Shards[shardsLength-1].ShardId
	}

	if p.options.StopOnDuplicateToken &&
		prevToken != nil &&
		p.exclusiveStartShardID != nil &&
		*prevToken == *p.exclusiveStartShardID {
		p.isTruncated = aws.Bool(false)
	}

	return result, nil
}
