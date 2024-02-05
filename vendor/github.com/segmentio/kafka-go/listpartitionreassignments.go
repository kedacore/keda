package kafka

import (
	"context"
	"net"
	"time"

	"github.com/segmentio/kafka-go/protocol/listpartitionreassignments"
)

// ListPartitionReassignmentsRequest is a request to the ListPartitionReassignments API.
type ListPartitionReassignmentsRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// Topics we want reassignments for, mapped by their name, or nil to list everything.
	Topics map[string]ListPartitionReassignmentsRequestTopic

	// Timeout is the amount of time to wait for the request to complete.
	Timeout time.Duration
}

// ListPartitionReassignmentsRequestTopic contains the requested partitions for a single
// topic.
type ListPartitionReassignmentsRequestTopic struct {
	// The partitions to list partition reassignments for.
	PartitionIndexes []int
}

// ListPartitionReassignmentsResponse is a response from the ListPartitionReassignments API.
type ListPartitionReassignmentsResponse struct {
	// Error is set to a non-nil value including the code and message if a top-level
	// error was encountered.
	Error error

	// Topics contains results for each topic, mapped by their name.
	Topics map[string]ListPartitionReassignmentsResponseTopic
}

// ListPartitionReassignmentsResponseTopic contains the detailed result of
// ongoing reassignments for a topic.
type ListPartitionReassignmentsResponseTopic struct {
	// Partitions contains result for topic partitions.
	Partitions []ListPartitionReassignmentsResponsePartition
}

// ListPartitionReassignmentsResponsePartition contains the detailed result of
// ongoing reassignments for a single partition.
type ListPartitionReassignmentsResponsePartition struct {
	// PartitionIndex contains index of the partition.
	PartitionIndex int

	// Replicas contains the current replica set.
	Replicas []int

	// AddingReplicas contains the set of replicas we are currently adding.
	AddingReplicas []int

	// RemovingReplicas contains the set of replicas we are currently removing.
	RemovingReplicas []int
}

func (c *Client) ListPartitionReassignments(
	ctx context.Context,
	req *ListPartitionReassignmentsRequest,
) (*ListPartitionReassignmentsResponse, error) {
	apiReq := &listpartitionreassignments.Request{
		TimeoutMs: int32(req.Timeout.Milliseconds()),
	}

	for topicName, topicReq := range req.Topics {
		apiReq.Topics = append(
			apiReq.Topics,
			listpartitionreassignments.RequestTopic{
				Name:             topicName,
				PartitionIndexes: intToInt32Array(topicReq.PartitionIndexes),
			},
		)
	}

	protoResp, err := c.roundTrip(
		ctx,
		req.Addr,
		apiReq,
	)
	if err != nil {
		return nil, err
	}
	apiResp := protoResp.(*listpartitionreassignments.Response)

	resp := &ListPartitionReassignmentsResponse{
		Error:  makeError(apiResp.ErrorCode, apiResp.ErrorMessage),
		Topics: make(map[string]ListPartitionReassignmentsResponseTopic),
	}

	for _, topicResult := range apiResp.Topics {
		respTopic := ListPartitionReassignmentsResponseTopic{}
		for _, partitionResult := range topicResult.Partitions {
			respTopic.Partitions = append(
				respTopic.Partitions,
				ListPartitionReassignmentsResponsePartition{
					PartitionIndex:   int(partitionResult.PartitionIndex),
					Replicas:         int32ToIntArray(partitionResult.Replicas),
					AddingReplicas:   int32ToIntArray(partitionResult.AddingReplicas),
					RemovingReplicas: int32ToIntArray(partitionResult.RemovingReplicas),
				},
			)
		}
		resp.Topics[topicResult.Name] = respTopic
	}

	return resp, nil
}

func intToInt32Array(arr []int) []int32 {
	if arr == nil {
		return nil
	}
	res := make([]int32, len(arr))
	for i := range arr {
		res[i] = int32(arr[i])
	}
	return res
}

func int32ToIntArray(arr []int32) []int {
	if arr == nil {
		return nil
	}
	res := make([]int, len(arr))
	for i := range arr {
		res[i] = int(arr[i])
	}
	return res
}
