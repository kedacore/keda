package kafka

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go/protocol/createtopics"
)

// CreateTopicsRequest represents a request sent to a kafka broker to create
// new topics.
type CreateTopicsRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// List of topics to create and their configuration.
	Topics []TopicConfig

	// When set to true, topics are not created but the configuration is
	// validated as if they were.
	//
	// This field will be ignored if the kafka broker did not support the
	// CreateTopics API in version 1 or above.
	ValidateOnly bool
}

// CreateTopicsResponse represents a response from a kafka broker to a topic
// creation request.
type CreateTopicsResponse struct {
	// The amount of time that the broker throttled the request.
	//
	// This field will be zero if the kafka broker did not support the
	// CreateTopics API in version 2 or above.
	Throttle time.Duration

	// Mapping of topic names to errors that occurred while attempting to create
	// the topics.
	//
	// The errors contain the kafka error code. Programs may use the standard
	// errors.Is function to test the error against kafka error codes.
	Errors map[string]error
}

// CreateTopics sends a topic creation request to a kafka broker and returns the
// response.
func (c *Client) CreateTopics(ctx context.Context, req *CreateTopicsRequest) (*CreateTopicsResponse, error) {
	topics := make([]createtopics.RequestTopic, len(req.Topics))

	for i, t := range req.Topics {
		topics[i] = createtopics.RequestTopic{
			Name:              t.Topic,
			NumPartitions:     int32(t.NumPartitions),
			ReplicationFactor: int16(t.ReplicationFactor),
			Assignments:       t.assignments(),
			Configs:           t.configs(),
		}
	}

	m, err := c.roundTrip(ctx, req.Addr, &createtopics.Request{
		Topics:       topics,
		TimeoutMs:    c.timeoutMs(ctx, defaultCreateTopicsTimeout),
		ValidateOnly: req.ValidateOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka.(*Client).CreateTopics: %w", err)
	}

	res := m.(*createtopics.Response)
	ret := &CreateTopicsResponse{
		Throttle: makeDuration(res.ThrottleTimeMs),
		Errors:   make(map[string]error, len(res.Topics)),
	}

	for _, t := range res.Topics {
		ret.Errors[t.Name] = makeError(t.ErrorCode, t.ErrorMessage)
	}

	return ret, nil
}

type ConfigEntry struct {
	ConfigName  string
	ConfigValue string
}

func (c ConfigEntry) toCreateTopicsRequestV0ConfigEntry() createTopicsRequestV0ConfigEntry {
	return createTopicsRequestV0ConfigEntry(c)
}

type createTopicsRequestV0ConfigEntry struct {
	ConfigName  string
	ConfigValue string
}

func (t createTopicsRequestV0ConfigEntry) size() int32 {
	return sizeofString(t.ConfigName) +
		sizeofString(t.ConfigValue)
}

func (t createTopicsRequestV0ConfigEntry) writeTo(wb *writeBuffer) {
	wb.writeString(t.ConfigName)
	wb.writeString(t.ConfigValue)
}

type ReplicaAssignment struct {
	Partition int
	// The list of brokers where the partition should be allocated. There must
	// be as many entries in thie list as there are replicas of the partition.
	// The first entry represents the broker that will be the preferred leader
	// for the partition.
	//
	// This field changed in 0.4 from `int` to `[]int`. It was invalid to pass
	// a single integer as this is supposed to be a list. While this introduces
	// a breaking change, it probably never worked before.
	Replicas []int
}

func (a *ReplicaAssignment) partitionIndex() int32 {
	return int32(a.Partition)
}

func (a *ReplicaAssignment) brokerIDs() []int32 {
	if len(a.Replicas) == 0 {
		return nil
	}
	replicas := make([]int32, len(a.Replicas))
	for i, r := range a.Replicas {
		replicas[i] = int32(r)
	}
	return replicas
}

func (a ReplicaAssignment) toCreateTopicsRequestV0ReplicaAssignment() createTopicsRequestV0ReplicaAssignment {
	return createTopicsRequestV0ReplicaAssignment{
		Partition: int32(a.Partition),
		Replicas:  a.brokerIDs(),
	}
}

type createTopicsRequestV0ReplicaAssignment struct {
	Partition int32
	Replicas  []int32
}

func (t createTopicsRequestV0ReplicaAssignment) size() int32 {
	return sizeofInt32(t.Partition) +
		(int32(len(t.Replicas)+1) * sizeofInt32(0)) // N+1 because the array length is a int32
}

func (t createTopicsRequestV0ReplicaAssignment) writeTo(wb *writeBuffer) {
	wb.writeInt32(t.Partition)
	wb.writeInt32(int32(len(t.Replicas)))
	for _, r := range t.Replicas {
		wb.writeInt32(int32(r))
	}
}

type TopicConfig struct {
	// Topic name
	Topic string

	// NumPartitions created. -1 indicates unset.
	NumPartitions int

	// ReplicationFactor for the topic. -1 indicates unset.
	ReplicationFactor int

	// ReplicaAssignments among kafka brokers for this topic partitions. If this
	// is set num_partitions and replication_factor must be unset.
	ReplicaAssignments []ReplicaAssignment

	// ConfigEntries holds topic level configuration for topic to be set.
	ConfigEntries []ConfigEntry
}

func (t *TopicConfig) assignments() []createtopics.RequestAssignment {
	if len(t.ReplicaAssignments) == 0 {
		return nil
	}
	assignments := make([]createtopics.RequestAssignment, len(t.ReplicaAssignments))
	for i, a := range t.ReplicaAssignments {
		assignments[i] = createtopics.RequestAssignment{
			PartitionIndex: a.partitionIndex(),
			BrokerIDs:      a.brokerIDs(),
		}
	}
	return assignments
}

func (t *TopicConfig) configs() []createtopics.RequestConfig {
	if len(t.ConfigEntries) == 0 {
		return nil
	}
	configs := make([]createtopics.RequestConfig, len(t.ConfigEntries))
	for i, c := range t.ConfigEntries {
		configs[i] = createtopics.RequestConfig{
			Name:  c.ConfigName,
			Value: c.ConfigValue,
		}
	}
	return configs
}

func (t TopicConfig) toCreateTopicsRequestV0Topic() createTopicsRequestV0Topic {
	requestV0ReplicaAssignments := make([]createTopicsRequestV0ReplicaAssignment, 0, len(t.ReplicaAssignments))
	for _, a := range t.ReplicaAssignments {
		requestV0ReplicaAssignments = append(
			requestV0ReplicaAssignments,
			a.toCreateTopicsRequestV0ReplicaAssignment())
	}
	requestV0ConfigEntries := make([]createTopicsRequestV0ConfigEntry, 0, len(t.ConfigEntries))
	for _, c := range t.ConfigEntries {
		requestV0ConfigEntries = append(
			requestV0ConfigEntries,
			c.toCreateTopicsRequestV0ConfigEntry())
	}

	return createTopicsRequestV0Topic{
		Topic:              t.Topic,
		NumPartitions:      int32(t.NumPartitions),
		ReplicationFactor:  int16(t.ReplicationFactor),
		ReplicaAssignments: requestV0ReplicaAssignments,
		ConfigEntries:      requestV0ConfigEntries,
	}
}

type createTopicsRequestV0Topic struct {
	// Topic name
	Topic string

	// NumPartitions created. -1 indicates unset.
	NumPartitions int32

	// ReplicationFactor for the topic. -1 indicates unset.
	ReplicationFactor int16

	// ReplicaAssignments among kafka brokers for this topic partitions. If this
	// is set num_partitions and replication_factor must be unset.
	ReplicaAssignments []createTopicsRequestV0ReplicaAssignment

	// ConfigEntries holds topic level configuration for topic to be set.
	ConfigEntries []createTopicsRequestV0ConfigEntry
}

func (t createTopicsRequestV0Topic) size() int32 {
	return sizeofString(t.Topic) +
		sizeofInt32(t.NumPartitions) +
		sizeofInt16(t.ReplicationFactor) +
		sizeofArray(len(t.ReplicaAssignments), func(i int) int32 { return t.ReplicaAssignments[i].size() }) +
		sizeofArray(len(t.ConfigEntries), func(i int) int32 { return t.ConfigEntries[i].size() })
}

func (t createTopicsRequestV0Topic) writeTo(wb *writeBuffer) {
	wb.writeString(t.Topic)
	wb.writeInt32(t.NumPartitions)
	wb.writeInt16(t.ReplicationFactor)
	wb.writeArray(len(t.ReplicaAssignments), func(i int) { t.ReplicaAssignments[i].writeTo(wb) })
	wb.writeArray(len(t.ConfigEntries), func(i int) { t.ConfigEntries[i].writeTo(wb) })
}

// See http://kafka.apache.org/protocol.html#The_Messages_CreateTopics
type createTopicsRequest struct {
	v apiVersion // v0, v1, v2

	// Topics contains n array of single topic creation requests. Can not
	// have multiple entries for the same topic.
	Topics []createTopicsRequestV0Topic

	// Timeout ms to wait for a topic to be completely created on the
	// controller node. Values <= 0 will trigger topic creation and return immediately
	Timeout int32

	// If true, check that the topics can be created as specified, but don't create anything.
	// Internal use only for Kafka 4.0 support.
	ValidateOnly bool
}

func (t createTopicsRequest) size() int32 {
	sz := sizeofArray(len(t.Topics), func(i int) int32 { return t.Topics[i].size() }) +
		sizeofInt32(t.Timeout)
	if t.v >= v1 {
		sz += 1
	}
	return sz
}

func (t createTopicsRequest) writeTo(wb *writeBuffer) {
	wb.writeArray(len(t.Topics), func(i int) { t.Topics[i].writeTo(wb) })
	wb.writeInt32(t.Timeout)
	if t.v >= v1 {
		wb.writeBool(t.ValidateOnly)
	}
}

type createTopicsResponseTopicError struct {
	v apiVersion

	// Topic name
	Topic string

	// ErrorCode holds response error code
	ErrorCode int16

	// ErrorMessage holds response error message string
	ErrorMessage string
}

func (t createTopicsResponseTopicError) size() int32 {
	sz := sizeofString(t.Topic) +
		sizeofInt16(t.ErrorCode)
	if t.v >= v1 {
		sz += sizeofString(t.ErrorMessage)
	}
	return sz
}

func (t createTopicsResponseTopicError) writeTo(wb *writeBuffer) {
	wb.writeString(t.Topic)
	wb.writeInt16(t.ErrorCode)
	if t.v >= v1 {
		wb.writeString(t.ErrorMessage)
	}
}

func (t *createTopicsResponseTopicError) readFrom(r *bufio.Reader, size int) (remain int, err error) {
	if remain, err = readString(r, size, &t.Topic); err != nil {
		return
	}
	if remain, err = readInt16(r, remain, &t.ErrorCode); err != nil {
		return
	}
	if t.v >= v1 {
		if remain, err = readString(r, remain, &t.ErrorMessage); err != nil {
			return
		}
	}
	return
}

// See http://kafka.apache.org/protocol.html#The_Messages_CreateTopics
type createTopicsResponse struct {
	v apiVersion

	ThrottleTime int32 // v2+
	TopicErrors  []createTopicsResponseTopicError
}

func (t createTopicsResponse) size() int32 {
	sz := sizeofArray(len(t.TopicErrors), func(i int) int32 { return t.TopicErrors[i].size() })
	if t.v >= v2 {
		sz += sizeofInt32(t.ThrottleTime)
	}
	return sz
}

func (t createTopicsResponse) writeTo(wb *writeBuffer) {
	if t.v >= v2 {
		wb.writeInt32(t.ThrottleTime)
	}
	wb.writeArray(len(t.TopicErrors), func(i int) { t.TopicErrors[i].writeTo(wb) })
}

func (t *createTopicsResponse) readFrom(r *bufio.Reader, size int) (remain int, err error) {
	fn := func(r *bufio.Reader, size int) (fnRemain int, fnErr error) {
		topic := createTopicsResponseTopicError{v: t.v}
		if fnRemain, fnErr = (&topic).readFrom(r, size); fnErr != nil {
			return
		}
		t.TopicErrors = append(t.TopicErrors, topic)
		return
	}
	remain = size
	if t.v >= v2 {
		if remain, err = readInt32(r, size, &t.ThrottleTime); err != nil {
			return
		}
	}
	if remain, err = readArrayWith(r, remain, fn); err != nil {
		return
	}

	return
}

func (c *Conn) createTopics(request createTopicsRequest) (createTopicsResponse, error) {
	version, err := c.negotiateVersion(createTopics, v0, v1, v2)
	if err != nil {
		return createTopicsResponse{}, err
	}

	request.v = version
	response := createTopicsResponse{v: version}

	err = c.writeOperation(
		func(deadline time.Time, id int32) error {
			if request.Timeout == 0 {
				now := time.Now()
				deadline = adjustDeadlineForRTT(deadline, now, defaultRTT)
				request.Timeout = milliseconds(deadlineToTimeout(deadline, now))
			}
			return c.writeRequest(createTopics, version, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return response, err
	}
	for _, tr := range response.TopicErrors {
		if tr.ErrorCode == int16(TopicAlreadyExists) {
			continue
		}
		if tr.ErrorCode != 0 {
			return response, Error(tr.ErrorCode)
		}
	}

	return response, nil
}

// CreateTopics creates one topic per provided configuration with idempotent
// operational semantics. In other words, if CreateTopics is invoked with a
// configuration for an existing topic, it will have no effect.
func (c *Conn) CreateTopics(topics ...TopicConfig) error {
	requestV0Topics := make([]createTopicsRequestV0Topic, 0, len(topics))
	for _, t := range topics {
		requestV0Topics = append(
			requestV0Topics,
			t.toCreateTopicsRequestV0Topic())
	}

	_, err := c.createTopics(createTopicsRequest{
		Topics: requestV0Topics,
	})
	return err
}
