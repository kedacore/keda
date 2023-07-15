package kafka

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errInvalidWriteTopic     = errors.New("writes must NOT set Topic on kafka.Message")
	errInvalidWritePartition = errors.New("writes must NOT set Partition on kafka.Message")
)

// Conn represents a connection to a kafka broker.
//
// Instances of Conn are safe to use concurrently from multiple goroutines.
type Conn struct {
	// base network connection
	conn net.Conn

	// number of inflight requests on the connection.
	inflight int32

	// offset management (synchronized on the mutex field)
	mutex  sync.Mutex
	offset int64

	// read buffer (synchronized on rlock)
	rlock sync.Mutex
	rbuf  bufio.Reader

	// write buffer (synchronized on wlock)
	wlock sync.Mutex
	wbuf  bufio.Writer
	wb    writeBuffer

	// deadline management
	wdeadline connDeadline
	rdeadline connDeadline

	// immutable values of the connection object
	clientID      string
	topic         string
	partition     int32
	fetchMaxBytes int32
	fetchMinSize  int32
	broker        int32
	rack          string

	// correlation ID generator (synchronized on wlock)
	correlationID int32

	// number of replica acks required when publishing to a partition
	requiredAcks int32

	// lazily loaded API versions used by this connection
	apiVersions atomic.Value // apiVersionMap

	transactionalID *string
}

type apiVersionMap map[apiKey]ApiVersion

func (v apiVersionMap) negotiate(key apiKey, sortedSupportedVersions ...apiVersion) apiVersion {
	x := v[key]

	for i := len(sortedSupportedVersions) - 1; i >= 0; i-- {
		s := sortedSupportedVersions[i]

		if apiVersion(x.MaxVersion) >= s {
			return s
		}
	}

	return -1
}

// ConnConfig is a configuration object used to create new instances of Conn.
type ConnConfig struct {
	ClientID  string
	Topic     string
	Partition int
	Broker    int
	Rack      string

	// The transactional id to use for transactional delivery. Idempotent
	// deliver should be enabled if transactional id is configured.
	// For more details look at transactional.id description here: http://kafka.apache.org/documentation.html#producerconfigs
	// Empty string means that this connection can't be transactional.
	TransactionalID string
}

// ReadBatchConfig is a configuration object used for reading batches of messages.
type ReadBatchConfig struct {
	// MinBytes indicates to the broker the minimum batch size that the consumer
	// will accept. Setting a high minimum when consuming from a low-volume topic
	// may result in delayed delivery when the broker does not have enough data to
	// satisfy the defined minimum.
	MinBytes int

	// MaxBytes indicates to the broker the maximum batch size that the consumer
	// will accept. The broker will truncate a message to satisfy this maximum, so
	// choose a value that is high enough for your largest message size.
	MaxBytes int

	// IsolationLevel controls the visibility of transactional records.
	// ReadUncommitted makes all records visible. With ReadCommitted only
	// non-transactional and committed records are visible.
	IsolationLevel IsolationLevel

	// MaxWait is the amount of time for the broker while waiting to hit the
	// min/max byte targets.  This setting is independent of any network-level
	// timeouts or deadlines.
	//
	// For backward compatibility, when this field is left zero, kafka-go will
	// infer the max wait from the connection's read deadline.
	MaxWait time.Duration
}

type IsolationLevel int8

const (
	ReadUncommitted IsolationLevel = 0
	ReadCommitted   IsolationLevel = 1
)

var (
	// DefaultClientID is the default value used as ClientID of kafka
	// connections.
	DefaultClientID string
)

func init() {
	progname := filepath.Base(os.Args[0])
	hostname, _ := os.Hostname()
	DefaultClientID = fmt.Sprintf("%s@%s (github.com/segmentio/kafka-go)", progname, hostname)
}

// NewConn returns a new kafka connection for the given topic and partition.
func NewConn(conn net.Conn, topic string, partition int) *Conn {
	return NewConnWith(conn, ConnConfig{
		Topic:     topic,
		Partition: partition,
	})
}

func emptyToNullable(transactionalID string) (result *string) {
	if transactionalID != "" {
		result = &transactionalID
	}
	return result
}

// NewConnWith returns a new kafka connection configured with config.
// The offset is initialized to FirstOffset.
func NewConnWith(conn net.Conn, config ConnConfig) *Conn {
	if len(config.ClientID) == 0 {
		config.ClientID = DefaultClientID
	}

	if config.Partition < 0 || config.Partition > math.MaxInt32 {
		panic(fmt.Sprintf("invalid partition number: %d", config.Partition))
	}

	c := &Conn{
		conn:            conn,
		rbuf:            *bufio.NewReader(conn),
		wbuf:            *bufio.NewWriter(conn),
		clientID:        config.ClientID,
		topic:           config.Topic,
		partition:       int32(config.Partition),
		broker:          int32(config.Broker),
		rack:            config.Rack,
		offset:          FirstOffset,
		requiredAcks:    -1,
		transactionalID: emptyToNullable(config.TransactionalID),
	}

	c.wb.w = &c.wbuf

	// The fetch request needs to ask for a MaxBytes value that is at least
	// enough to load the control data of the response. To avoid having to
	// recompute it on every read, it is cached here in the Conn value.
	c.fetchMinSize = (fetchResponseV2{
		Topics: []fetchResponseTopicV2{{
			TopicName: config.Topic,
			Partitions: []fetchResponsePartitionV2{{
				Partition:  int32(config.Partition),
				MessageSet: messageSet{{}},
			}},
		}},
	}).size()
	c.fetchMaxBytes = math.MaxInt32 - c.fetchMinSize
	return c
}

func (c *Conn) negotiateVersion(key apiKey, sortedSupportedVersions ...apiVersion) (apiVersion, error) {
	v, err := c.loadVersions()
	if err != nil {
		return -1, err
	}
	a := v.negotiate(key, sortedSupportedVersions...)
	if a < 0 {
		return -1, fmt.Errorf("no matching versions were found between the client and the broker for API key %d", key)
	}
	return a, nil
}

func (c *Conn) loadVersions() (apiVersionMap, error) {
	v, _ := c.apiVersions.Load().(apiVersionMap)
	if v != nil {
		return v, nil
	}

	brokerVersions, err := c.ApiVersions()
	if err != nil {
		return nil, err
	}

	v = make(apiVersionMap, len(brokerVersions))

	for _, a := range brokerVersions {
		v[apiKey(a.ApiKey)] = a
	}

	c.apiVersions.Store(v)
	return v, nil
}

// Broker returns a Broker value representing the kafka broker that this
// connection was established to.
func (c *Conn) Broker() Broker {
	addr := c.conn.RemoteAddr()
	host, port, _ := splitHostPortNumber(addr.String())
	return Broker{
		Host: host,
		Port: port,
		ID:   int(c.broker),
		Rack: c.rack,
	}
}

// Controller requests kafka for the current controller and returns its URL.
func (c *Conn) Controller() (broker Broker, err error) {
	err = c.readOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(metadata, v1, id, topicMetadataRequestV1([]string{}))
		},
		func(deadline time.Time, size int) error {
			var res metadataResponseV1

			if err := c.readResponse(size, &res); err != nil {
				return err
			}
			for _, brokerMeta := range res.Brokers {
				if brokerMeta.NodeID == res.ControllerID {
					broker = Broker{ID: int(brokerMeta.NodeID),
						Port: int(brokerMeta.Port),
						Host: brokerMeta.Host,
						Rack: brokerMeta.Rack}
					break
				}
			}
			return nil
		},
	)
	return broker, err
}

// Brokers retrieve the broker list from the Kafka metadata.
func (c *Conn) Brokers() ([]Broker, error) {
	var brokers []Broker
	err := c.readOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(metadata, v1, id, topicMetadataRequestV1([]string{}))
		},
		func(deadline time.Time, size int) error {
			var res metadataResponseV1

			if err := c.readResponse(size, &res); err != nil {
				return err
			}

			brokers = make([]Broker, len(res.Brokers))
			for i, brokerMeta := range res.Brokers {
				brokers[i] = Broker{
					ID:   int(brokerMeta.NodeID),
					Port: int(brokerMeta.Port),
					Host: brokerMeta.Host,
					Rack: brokerMeta.Rack,
				}
			}
			return nil
		},
	)
	return brokers, err
}

// DeleteTopics deletes the specified topics.
func (c *Conn) DeleteTopics(topics ...string) error {
	_, err := c.deleteTopics(deleteTopicsRequestV0{
		Topics: topics,
	})
	return err
}

// findCoordinator finds the coordinator for the specified group or transaction
//
// See http://kafka.apache.org/protocol.html#The_Messages_FindCoordinator
func (c *Conn) findCoordinator(request findCoordinatorRequestV0) (findCoordinatorResponseV0, error) {
	var response findCoordinatorResponseV0

	err := c.readOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(findCoordinator, v0, id, request)

		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return findCoordinatorResponseV0{}, err
	}
	if response.ErrorCode != 0 {
		return findCoordinatorResponseV0{}, Error(response.ErrorCode)
	}

	return response, nil
}

// heartbeat sends a heartbeat message required by consumer groups
//
// See http://kafka.apache.org/protocol.html#The_Messages_Heartbeat
func (c *Conn) heartbeat(request heartbeatRequestV0) (heartbeatResponseV0, error) {
	var response heartbeatResponseV0

	err := c.writeOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(heartbeat, v0, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return heartbeatResponseV0{}, err
	}
	if response.ErrorCode != 0 {
		return heartbeatResponseV0{}, Error(response.ErrorCode)
	}

	return response, nil
}

// joinGroup attempts to join a consumer group
//
// See http://kafka.apache.org/protocol.html#The_Messages_JoinGroup
func (c *Conn) joinGroup(request joinGroupRequestV1) (joinGroupResponseV1, error) {
	var response joinGroupResponseV1

	err := c.writeOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(joinGroup, v1, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return joinGroupResponseV1{}, err
	}
	if response.ErrorCode != 0 {
		return joinGroupResponseV1{}, Error(response.ErrorCode)
	}

	return response, nil
}

// leaveGroup leaves the consumer from the consumer group
//
// See http://kafka.apache.org/protocol.html#The_Messages_LeaveGroup
func (c *Conn) leaveGroup(request leaveGroupRequestV0) (leaveGroupResponseV0, error) {
	var response leaveGroupResponseV0

	err := c.writeOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(leaveGroup, v0, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return leaveGroupResponseV0{}, err
	}
	if response.ErrorCode != 0 {
		return leaveGroupResponseV0{}, Error(response.ErrorCode)
	}

	return response, nil
}

// listGroups lists all the consumer groups
//
// See http://kafka.apache.org/protocol.html#The_Messages_ListGroups
func (c *Conn) listGroups(request listGroupsRequestV1) (listGroupsResponseV1, error) {
	var response listGroupsResponseV1

	err := c.readOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(listGroups, v1, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return listGroupsResponseV1{}, err
	}
	if response.ErrorCode != 0 {
		return listGroupsResponseV1{}, Error(response.ErrorCode)
	}

	return response, nil
}

// offsetCommit commits the specified topic partition offsets
//
// See http://kafka.apache.org/protocol.html#The_Messages_OffsetCommit
func (c *Conn) offsetCommit(request offsetCommitRequestV2) (offsetCommitResponseV2, error) {
	var response offsetCommitResponseV2

	err := c.writeOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(offsetCommit, v2, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return offsetCommitResponseV2{}, err
	}
	for _, r := range response.Responses {
		for _, pr := range r.PartitionResponses {
			if pr.ErrorCode != 0 {
				return offsetCommitResponseV2{}, Error(pr.ErrorCode)
			}
		}
	}

	return response, nil
}

// offsetFetch fetches the offsets for the specified topic partitions.
// -1 indicates that there is no offset saved for the partition.
//
// See http://kafka.apache.org/protocol.html#The_Messages_OffsetFetch
func (c *Conn) offsetFetch(request offsetFetchRequestV1) (offsetFetchResponseV1, error) {
	var response offsetFetchResponseV1

	err := c.readOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(offsetFetch, v1, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return offsetFetchResponseV1{}, err
	}
	for _, r := range response.Responses {
		for _, pr := range r.PartitionResponses {
			if pr.ErrorCode != 0 {
				return offsetFetchResponseV1{}, Error(pr.ErrorCode)
			}
		}
	}

	return response, nil
}

// syncGroup completes the handshake to join a consumer group
//
// See http://kafka.apache.org/protocol.html#The_Messages_SyncGroup
func (c *Conn) syncGroup(request syncGroupRequestV0) (syncGroupResponseV0, error) {
	var response syncGroupResponseV0

	err := c.readOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(syncGroup, v0, id, request)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (remain int, err error) {
				return (&response).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err != nil {
		return syncGroupResponseV0{}, err
	}
	if response.ErrorCode != 0 {
		return syncGroupResponseV0{}, Error(response.ErrorCode)
	}

	return response, nil
}

// Close closes the kafka connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations fail with a timeout
// (see type Error) instead of blocking. The deadline applies to all future and
// pending I/O, not just the immediately following call to Read or Write. After
// a deadline has been exceeded, the connection may be closed if it was found to
// be in an unrecoverable state.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	c.rdeadline.setDeadline(t)
	c.wdeadline.setDeadline(t)
	return nil
}

// SetReadDeadline sets the deadline for future Read calls and any
// currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.rdeadline.setDeadline(t)
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls and any
// currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that some of the
// data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.wdeadline.setDeadline(t)
	return nil
}

// Offset returns the current offset of the connection as pair of integers,
// where the first one is an offset value and the second one indicates how
// to interpret it.
//
// See Seek for more details about the offset and whence values.
func (c *Conn) Offset() (offset int64, whence int) {
	c.mutex.Lock()
	offset = c.offset
	c.mutex.Unlock()

	switch offset {
	case FirstOffset:
		offset = 0
		whence = SeekStart
	case LastOffset:
		offset = 0
		whence = SeekEnd
	default:
		whence = SeekAbsolute
	}
	return
}

const (
	SeekStart    = 0 // Seek relative to the first offset available in the partition.
	SeekAbsolute = 1 // Seek to an absolute offset.
	SeekEnd      = 2 // Seek relative to the last offset available in the partition.
	SeekCurrent  = 3 // Seek relative to the current offset.

	// This flag may be combined to any of the SeekAbsolute and SeekCurrent
	// constants to skip the bound check that the connection would do otherwise.
	// Programs can use this flag to avoid making a metadata request to the kafka
	// broker to read the current first and last offsets of the partition.
	SeekDontCheck = 1 << 30
)

// Seek sets the offset for the next read or write operation according to whence, which
// should be one of SeekStart, SeekAbsolute, SeekEnd, or SeekCurrent.
// When seeking relative to the end, the offset is subtracted from the current offset.
// Note that for historical reasons, these do not align with the usual whence constants
// as in lseek(2) or os.Seek.
// The method returns the new absolute offset of the connection.
func (c *Conn) Seek(offset int64, whence int) (int64, error) {
	seekDontCheck := (whence & SeekDontCheck) != 0
	whence &= ^SeekDontCheck

	switch whence {
	case SeekStart, SeekAbsolute, SeekEnd, SeekCurrent:
	default:
		return 0, fmt.Errorf("whence must be one of 0, 1, 2, or 3. (whence = %d)", whence)
	}

	if seekDontCheck {
		if whence == SeekAbsolute {
			c.mutex.Lock()
			c.offset = offset
			c.mutex.Unlock()
			return offset, nil
		}

		if whence == SeekCurrent {
			c.mutex.Lock()
			c.offset += offset
			offset = c.offset
			c.mutex.Unlock()
			return offset, nil
		}
	}

	if whence == SeekAbsolute {
		c.mutex.Lock()
		unchanged := offset == c.offset
		c.mutex.Unlock()
		if unchanged {
			return offset, nil
		}
	}

	if whence == SeekCurrent {
		c.mutex.Lock()
		offset = c.offset + offset
		c.mutex.Unlock()
	}

	first, last, err := c.ReadOffsets()
	if err != nil {
		return 0, err
	}

	switch whence {
	case SeekStart:
		offset = first + offset
	case SeekEnd:
		offset = last - offset
	}

	if offset < first || offset > last {
		return 0, OffsetOutOfRange
	}

	c.mutex.Lock()
	c.offset = offset
	c.mutex.Unlock()
	return offset, nil
}

// Read reads the message at the current offset from the connection, advancing
// the offset on success so the next call to a read method will produce the next
// message.
// The method returns the number of bytes read, or an error if something went
// wrong.
//
// While it is safe to call Read concurrently from multiple goroutines it may
// be hard for the program to predict the results as the connection offset will
// be read and written by multiple goroutines, they could read duplicates, or
// messages may be seen by only some of the goroutines.
//
// The method fails with io.ErrShortBuffer if the buffer passed as argument is
// too small to hold the message value.
//
// This method is provided to satisfy the net.Conn interface but is much less
// efficient than using the more general purpose ReadBatch method.
func (c *Conn) Read(b []byte) (int, error) {
	batch := c.ReadBatch(1, len(b))
	n, err := batch.Read(b)
	return n, coalesceErrors(silentEOF(err), batch.Close())
}

// ReadMessage reads the message at the current offset from the connection,
// advancing the offset on success so the next call to a read method will
// produce the next message.
//
// Because this method allocate memory buffers for the message key and value
// it is less memory-efficient than Read, but has the advantage of never
// failing with io.ErrShortBuffer.
//
// While it is safe to call Read concurrently from multiple goroutines it may
// be hard for the program to predict the results as the connection offset will
// be read and written by multiple goroutines, they could read duplicates, or
// messages may be seen by only some of the goroutines.
//
// This method is provided for convenience purposes but is much less efficient
// than using the more general purpose ReadBatch method.
func (c *Conn) ReadMessage(maxBytes int) (Message, error) {
	batch := c.ReadBatch(1, maxBytes)
	msg, err := batch.ReadMessage()
	return msg, coalesceErrors(silentEOF(err), batch.Close())
}

// ReadBatch reads a batch of messages from the kafka server. The method always
// returns a non-nil Batch value. If an error occurred, either sending the fetch
// request or reading the response, the error will be made available by the
// returned value of  the batch's Close method.
//
// While it is safe to call ReadBatch concurrently from multiple goroutines it
// may be hard for the program to predict the results as the connection offset
// will be read and written by multiple goroutines, they could read duplicates,
// or messages may be seen by only some of the goroutines.
//
// A program doesn't specify the number of messages in wants from a batch, but
// gives the minimum and maximum number of bytes that it wants to receive from
// the kafka server.
func (c *Conn) ReadBatch(minBytes, maxBytes int) *Batch {
	return c.ReadBatchWith(ReadBatchConfig{
		MinBytes: minBytes,
		MaxBytes: maxBytes,
	})
}

// ReadBatchWith in every way is similar to ReadBatch. ReadBatch is configured
// with the default values in ReadBatchConfig except for minBytes and maxBytes.
func (c *Conn) ReadBatchWith(cfg ReadBatchConfig) *Batch {

	var adjustedDeadline time.Time
	var maxFetch = int(c.fetchMaxBytes)

	if cfg.MinBytes < 0 || cfg.MinBytes > maxFetch {
		return &Batch{err: fmt.Errorf("kafka.(*Conn).ReadBatch: minBytes of %d out of [1,%d] bounds", cfg.MinBytes, maxFetch)}
	}
	if cfg.MaxBytes < 0 || cfg.MaxBytes > maxFetch {
		return &Batch{err: fmt.Errorf("kafka.(*Conn).ReadBatch: maxBytes of %d out of [1,%d] bounds", cfg.MaxBytes, maxFetch)}
	}
	if cfg.MinBytes > cfg.MaxBytes {
		return &Batch{err: fmt.Errorf("kafka.(*Conn).ReadBatch: minBytes (%d) > maxBytes (%d)", cfg.MinBytes, cfg.MaxBytes)}
	}

	offset, whence := c.Offset()

	offset, err := c.Seek(offset, whence|SeekDontCheck)
	if err != nil {
		return &Batch{err: dontExpectEOF(err)}
	}

	fetchVersion, err := c.negotiateVersion(fetch, v2, v5, v10)
	if err != nil {
		return &Batch{err: dontExpectEOF(err)}
	}

	id, err := c.doRequest(&c.rdeadline, func(deadline time.Time, id int32) error {
		now := time.Now()
		var timeout time.Duration
		if cfg.MaxWait > 0 {
			// explicitly-configured case: no changes are made to the deadline,
			// and the timeout is sent exactly as specified.
			timeout = cfg.MaxWait
		} else {
			// default case: use the original logic to adjust the conn's
			// deadline.T
			deadline = adjustDeadlineForRTT(deadline, now, defaultRTT)
			timeout = deadlineToTimeout(deadline, now)
		}
		// save this variable outside of the closure for later use in detecting
		// truncated messages.
		adjustedDeadline = deadline
		switch fetchVersion {
		case v10:
			return c.wb.writeFetchRequestV10(
				id,
				c.clientID,
				c.topic,
				c.partition,
				offset,
				cfg.MinBytes,
				cfg.MaxBytes+int(c.fetchMinSize),
				timeout,
				int8(cfg.IsolationLevel),
			)
		case v5:
			return c.wb.writeFetchRequestV5(
				id,
				c.clientID,
				c.topic,
				c.partition,
				offset,
				cfg.MinBytes,
				cfg.MaxBytes+int(c.fetchMinSize),
				timeout,
				int8(cfg.IsolationLevel),
			)
		default:
			return c.wb.writeFetchRequestV2(
				id,
				c.clientID,
				c.topic,
				c.partition,
				offset,
				cfg.MinBytes,
				cfg.MaxBytes+int(c.fetchMinSize),
				timeout,
			)
		}
	})
	if err != nil {
		return &Batch{err: dontExpectEOF(err)}
	}

	_, size, lock, err := c.waitResponse(&c.rdeadline, id)
	if err != nil {
		return &Batch{err: dontExpectEOF(err)}
	}

	var throttle int32
	var highWaterMark int64
	var remain int

	switch fetchVersion {
	case v10:
		throttle, highWaterMark, remain, err = readFetchResponseHeaderV10(&c.rbuf, size)
	case v5:
		throttle, highWaterMark, remain, err = readFetchResponseHeaderV5(&c.rbuf, size)
	default:
		throttle, highWaterMark, remain, err = readFetchResponseHeaderV2(&c.rbuf, size)
	}
	if errors.Is(err, errShortRead) {
		err = checkTimeoutErr(adjustedDeadline)
	}

	var msgs *messageSetReader
	if err == nil {
		if highWaterMark == offset {
			msgs = &messageSetReader{empty: true}
		} else {
			msgs, err = newMessageSetReader(&c.rbuf, remain)
		}
	}
	if errors.Is(err, errShortRead) {
		err = checkTimeoutErr(adjustedDeadline)
	}

	return &Batch{
		conn:          c,
		msgs:          msgs,
		deadline:      adjustedDeadline,
		throttle:      makeDuration(throttle),
		lock:          lock,
		topic:         c.topic,          // topic is copied to Batch to prevent race with Batch.close
		partition:     int(c.partition), // partition is copied to Batch to prevent race with Batch.close
		offset:        offset,
		highWaterMark: highWaterMark,
		// there shouldn't be a short read on initially setting up the batch.
		// as such, any io.EOF is re-mapped to an io.ErrUnexpectedEOF so that we
		// don't accidentally signal that we successfully reached the end of the
		// batch.
		err: dontExpectEOF(err),
	}
}

// ReadOffset returns the offset of the first message with a timestamp equal or
// greater to t.
func (c *Conn) ReadOffset(t time.Time) (int64, error) {
	return c.readOffset(timestamp(t))
}

// ReadFirstOffset returns the first offset available on the connection.
func (c *Conn) ReadFirstOffset() (int64, error) {
	return c.readOffset(FirstOffset)
}

// ReadLastOffset returns the last offset available on the connection.
func (c *Conn) ReadLastOffset() (int64, error) {
	return c.readOffset(LastOffset)
}

// ReadOffsets returns the absolute first and last offsets of the topic used by
// the connection.
func (c *Conn) ReadOffsets() (first, last int64, err error) {
	// We have to submit two different requests to fetch the first and last
	// offsets because kafka refuses requests that ask for multiple offsets
	// on the same topic and partition.
	if first, err = c.ReadFirstOffset(); err != nil {
		return
	}
	if last, err = c.ReadLastOffset(); err != nil {
		first = 0 // don't leak the value on error
		return
	}
	return
}

func (c *Conn) readOffset(t int64) (offset int64, err error) {
	err = c.readOperation(
		func(deadline time.Time, id int32) error {
			return c.wb.writeListOffsetRequestV1(id, c.clientID, c.topic, c.partition, t)
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(readArrayWith(&c.rbuf, size, func(r *bufio.Reader, size int) (int, error) {
				// We skip the topic name because we've made a request for
				// a single topic.
				size, err := discardString(r, size)
				if err != nil {
					return size, err
				}

				// Reading the array of partitions, there will be only one
				// partition which gives the offset we're looking for.
				return readArrayWith(r, size, func(r *bufio.Reader, size int) (int, error) {
					var p partitionOffsetV1
					size, err := p.readFrom(r, size)
					if err != nil {
						return size, err
					}
					if p.ErrorCode != 0 {
						return size, Error(p.ErrorCode)
					}
					offset = p.Offset
					return size, nil
				})
			}))
		},
	)
	return
}

// ReadPartitions returns the list of available partitions for the given list of
// topics.
//
// If the method is called with no topic, it uses the topic configured on the
// connection. If there are none, the method fetches all partitions of the kafka
// cluster.
func (c *Conn) ReadPartitions(topics ...string) (partitions []Partition, err error) {

	if len(topics) == 0 {
		if len(c.topic) != 0 {
			defaultTopics := [...]string{c.topic}
			topics = defaultTopics[:]
		} else {
			// topics needs to be explicitly nil-ed out or the broker will
			// interpret it as a request for 0 partitions instead of all.
			topics = nil
		}
	}
	metadataVersion, err := c.negotiateVersion(metadata, v1, v6)
	if err != nil {
		return nil, err
	}

	err = c.readOperation(
		func(deadline time.Time, id int32) error {
			switch metadataVersion {
			case v6:
				return c.writeRequest(metadata, v6, id, topicMetadataRequestV6{Topics: topics, AllowAutoTopicCreation: true})
			default:
				return c.writeRequest(metadata, v1, id, topicMetadataRequestV1(topics))
			}
		},
		func(deadline time.Time, size int) error {
			partitions, err = c.readPartitionsResponse(metadataVersion, size)
			return err
		},
	)
	return
}

func (c *Conn) readPartitionsResponse(metadataVersion apiVersion, size int) ([]Partition, error) {
	switch metadataVersion {
	case v6:
		var res metadataResponseV6
		if err := c.readResponse(size, &res); err != nil {
			return nil, err
		}
		brokers := readBrokerMetadata(res.Brokers)
		return c.readTopicMetadatav6(brokers, res.Topics)
	default:
		var res metadataResponseV1
		if err := c.readResponse(size, &res); err != nil {
			return nil, err
		}
		brokers := readBrokerMetadata(res.Brokers)
		return c.readTopicMetadatav1(brokers, res.Topics)
	}
}

func readBrokerMetadata(brokerMetadata []brokerMetadataV1) map[int32]Broker {
	brokers := make(map[int32]Broker, len(brokerMetadata))
	for _, b := range brokerMetadata {
		brokers[b.NodeID] = Broker{
			Host: b.Host,
			Port: int(b.Port),
			ID:   int(b.NodeID),
			Rack: b.Rack,
		}
	}
	return brokers
}

func (c *Conn) readTopicMetadatav1(brokers map[int32]Broker, topicMetadata []topicMetadataV1) (partitions []Partition, err error) {
	for _, t := range topicMetadata {
		if t.TopicErrorCode != 0 && (c.topic == "" || t.TopicName == c.topic) {
			// We only report errors if they happened for the topic of
			// the connection, otherwise the topic will simply have no
			// partitions in the result set.
			return nil, Error(t.TopicErrorCode)
		}
		for _, p := range t.Partitions {
			partitions = append(partitions, Partition{
				Topic:           t.TopicName,
				Leader:          brokers[p.Leader],
				Replicas:        makeBrokers(brokers, p.Replicas...),
				Isr:             makeBrokers(brokers, p.Isr...),
				ID:              int(p.PartitionID),
				OfflineReplicas: []Broker{},
			})
		}
	}
	return
}

func (c *Conn) readTopicMetadatav6(brokers map[int32]Broker, topicMetadata []topicMetadataV6) (partitions []Partition, err error) {
	for _, t := range topicMetadata {
		if t.TopicErrorCode != 0 && (c.topic == "" || t.TopicName == c.topic) {
			// We only report errors if they happened for the topic of
			// the connection, otherwise the topic will simply have no
			// partitions in the result set.
			return nil, Error(t.TopicErrorCode)
		}
		for _, p := range t.Partitions {
			partitions = append(partitions, Partition{
				Topic:           t.TopicName,
				Leader:          brokers[p.Leader],
				Replicas:        makeBrokers(brokers, p.Replicas...),
				Isr:             makeBrokers(brokers, p.Isr...),
				ID:              int(p.PartitionID),
				OfflineReplicas: makeBrokers(brokers, p.OfflineReplicas...),
			})
		}
	}
	return
}

func makeBrokers(brokers map[int32]Broker, ids ...int32) []Broker {
	b := make([]Broker, len(ids))
	for i, id := range ids {
		br, ok := brokers[id]
		if !ok {
			// When the broker id isn't found in the current list of known
			// brokers, use a placeholder to report that the cluster has
			// logical knowledge of the broker but no information about the
			// physical host where it is running.
			br.ID = int(id)
		}
		b[i] = br
	}
	return b
}

// Write writes a message to the kafka broker that this connection was
// established to. The method returns the number of bytes written, or an error
// if something went wrong.
//
// The operation either succeeds or fail, it never partially writes the message.
//
// This method is exposed to satisfy the net.Conn interface but is less efficient
// than the more general purpose WriteMessages method.
func (c *Conn) Write(b []byte) (int, error) {
	return c.WriteCompressedMessages(nil, Message{Value: b})
}

// WriteMessages writes a batch of messages to the connection's topic and
// partition, returning the number of bytes written. The write is an atomic
// operation, it either fully succeeds or fails.
func (c *Conn) WriteMessages(msgs ...Message) (int, error) {
	return c.WriteCompressedMessages(nil, msgs...)
}

// WriteCompressedMessages writes a batch of messages to the connection's topic
// and partition, returning the number of bytes written. The write is an atomic
// operation, it either fully succeeds or fails.
//
// If the compression codec is not nil, the messages will be compressed.
func (c *Conn) WriteCompressedMessages(codec CompressionCodec, msgs ...Message) (nbytes int, err error) {
	nbytes, _, _, _, err = c.writeCompressedMessages(codec, msgs...)
	return
}

// WriteCompressedMessagesAt writes a batch of messages to the connection's topic
// and partition, returning the number of bytes written, partition and offset numbers
// and timestamp assigned by the kafka broker to the message set. The write is an atomic
// operation, it either fully succeeds or fails.
//
// If the compression codec is not nil, the messages will be compressed.
func (c *Conn) WriteCompressedMessagesAt(codec CompressionCodec, msgs ...Message) (nbytes int, partition int32, offset int64, appendTime time.Time, err error) {
	return c.writeCompressedMessages(codec, msgs...)
}

func (c *Conn) writeCompressedMessages(codec CompressionCodec, msgs ...Message) (nbytes int, partition int32, offset int64, appendTime time.Time, err error) {
	if len(msgs) == 0 {
		return
	}

	writeTime := time.Now()
	for i, msg := range msgs {
		// users may believe they can set the Topic and/or Partition
		// on the kafka message.
		if msg.Topic != "" && msg.Topic != c.topic {
			err = errInvalidWriteTopic
			return
		}
		if msg.Partition != 0 {
			err = errInvalidWritePartition
			return
		}

		if msg.Time.IsZero() {
			msgs[i].Time = writeTime
		}

		nbytes += len(msg.Key) + len(msg.Value)
	}

	var produceVersion apiVersion
	if produceVersion, err = c.negotiateVersion(produce, v2, v3, v7); err != nil {
		return
	}

	err = c.writeOperation(
		func(deadline time.Time, id int32) error {
			now := time.Now()
			deadline = adjustDeadlineForRTT(deadline, now, defaultRTT)
			switch produceVersion {
			case v7:
				recordBatch, err :=
					newRecordBatch(
						codec,
						msgs...,
					)
				if err != nil {
					return err
				}
				return c.wb.writeProduceRequestV7(
					id,
					c.clientID,
					c.topic,
					c.partition,
					deadlineToTimeout(deadline, now),
					int16(atomic.LoadInt32(&c.requiredAcks)),
					c.transactionalID,
					recordBatch,
				)
			case v3:
				recordBatch, err :=
					newRecordBatch(
						codec,
						msgs...,
					)
				if err != nil {
					return err
				}
				return c.wb.writeProduceRequestV3(
					id,
					c.clientID,
					c.topic,
					c.partition,
					deadlineToTimeout(deadline, now),
					int16(atomic.LoadInt32(&c.requiredAcks)),
					c.transactionalID,
					recordBatch,
				)
			default:
				return c.wb.writeProduceRequestV2(
					codec,
					id,
					c.clientID,
					c.topic,
					c.partition,
					deadlineToTimeout(deadline, now),
					int16(atomic.LoadInt32(&c.requiredAcks)),
					msgs...,
				)
			}
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(readArrayWith(&c.rbuf, size, func(r *bufio.Reader, size int) (int, error) {
				// Skip the topic, we've produced the message to only one topic,
				// no need to waste resources loading it in memory.
				size, err := discardString(r, size)
				if err != nil {
					return size, err
				}

				// Read the list of partitions, there should be only one since
				// we've produced a message to a single partition.
				size, err = readArrayWith(r, size, func(r *bufio.Reader, size int) (int, error) {
					switch produceVersion {
					case v7:
						var p produceResponsePartitionV7
						size, err := p.readFrom(r, size)
						if err == nil && p.ErrorCode != 0 {
							err = Error(p.ErrorCode)
						}
						if err == nil {
							partition = p.Partition
							offset = p.Offset
							appendTime = time.Unix(0, p.Timestamp*int64(time.Millisecond))
						}
						return size, err
					default:
						var p produceResponsePartitionV2
						size, err := p.readFrom(r, size)
						if err == nil && p.ErrorCode != 0 {
							err = Error(p.ErrorCode)
						}
						if err == nil {
							partition = p.Partition
							offset = p.Offset
							appendTime = time.Unix(0, p.Timestamp*int64(time.Millisecond))
						}
						return size, err
					}

				})
				if err != nil {
					return size, err
				}

				// The response is trailed by the throttle time, also skipping
				// since it's not interesting here.
				return discardInt32(r, size)
			}))
		},
	)

	if err != nil {
		nbytes = 0
	}

	return
}

// SetRequiredAcks sets the number of acknowledges from replicas that the
// connection requests when producing messages.
func (c *Conn) SetRequiredAcks(n int) error {
	switch n {
	case -1, 1:
		atomic.StoreInt32(&c.requiredAcks, int32(n))
		return nil
	default:
		return InvalidRequiredAcks
	}
}

func (c *Conn) writeRequest(apiKey apiKey, apiVersion apiVersion, correlationID int32, req request) error {
	hdr := c.requestHeader(apiKey, apiVersion, correlationID)
	hdr.Size = (hdr.size() + req.size()) - 4
	hdr.writeTo(&c.wb)
	req.writeTo(&c.wb)
	return c.wbuf.Flush()
}

func (c *Conn) readResponse(size int, res interface{}) error {
	size, err := read(&c.rbuf, size, res)
	if err != nil {
		var kafkaError Error
		if errors.As(err, &kafkaError) {
			size, err = discardN(&c.rbuf, size, size)
		}
	}
	return expectZeroSize(size, err)
}

func (c *Conn) peekResponseSizeAndID() (int32, int32, error) {
	b, err := c.rbuf.Peek(8)
	if err != nil {
		return 0, 0, err
	}
	size, id := makeInt32(b[:4]), makeInt32(b[4:])
	return size, id, nil
}

func (c *Conn) skipResponseSizeAndID() {
	c.rbuf.Discard(8)
}

func (c *Conn) readDeadline() time.Time {
	return c.rdeadline.deadline()
}

func (c *Conn) writeDeadline() time.Time {
	return c.wdeadline.deadline()
}

func (c *Conn) readOperation(write func(time.Time, int32) error, read func(time.Time, int) error) error {
	return c.do(&c.rdeadline, write, read)
}

func (c *Conn) writeOperation(write func(time.Time, int32) error, read func(time.Time, int) error) error {
	return c.do(&c.wdeadline, write, read)
}

func (c *Conn) enter() {
	atomic.AddInt32(&c.inflight, +1)
}

func (c *Conn) leave() {
	atomic.AddInt32(&c.inflight, -1)
}

func (c *Conn) concurrency() int {
	return int(atomic.LoadInt32(&c.inflight))
}

func (c *Conn) do(d *connDeadline, write func(time.Time, int32) error, read func(time.Time, int) error) error {
	id, err := c.doRequest(d, write)
	if err != nil {
		return err
	}

	deadline, size, lock, err := c.waitResponse(d, id)
	if err != nil {
		return err
	}

	if err = read(deadline, size); err != nil {
		var kafkaError Error
		if !errors.As(err, &kafkaError) {
			c.conn.Close()
		}
	}

	d.unsetConnReadDeadline()
	lock.Unlock()
	return err
}

func (c *Conn) doRequest(d *connDeadline, write func(time.Time, int32) error) (id int32, err error) {
	c.enter()
	c.wlock.Lock()
	c.correlationID++
	id = c.correlationID
	err = write(d.setConnWriteDeadline(c.conn), id)
	d.unsetConnWriteDeadline()

	if err != nil {
		// When an error occurs there's no way to know if the connection is in a
		// recoverable state so we're better off just giving up at this point to
		// avoid any risk of corrupting the following operations.
		c.conn.Close()
		c.leave()
	}

	c.wlock.Unlock()
	return
}

func (c *Conn) waitResponse(d *connDeadline, id int32) (deadline time.Time, size int, lock *sync.Mutex, err error) {
	for {
		var rsz int32
		var rid int32

		c.rlock.Lock()
		deadline = d.setConnReadDeadline(c.conn)
		rsz, rid, err = c.peekResponseSizeAndID()

		if err != nil {
			d.unsetConnReadDeadline()
			c.conn.Close()
			c.rlock.Unlock()
			break
		}

		if id == rid {
			c.skipResponseSizeAndID()
			size, lock = int(rsz-4), &c.rlock
			// Don't unlock the read mutex to yield ownership to the caller.
			break
		}

		if c.concurrency() == 1 {
			// If the goroutine is the only one waiting on this connection it
			// should be impossible to read a correlation id different from the
			// one it expects. This is a sign that the data we are reading on
			// the wire is corrupted and the connection needs to be closed.
			err = io.ErrNoProgress
			c.rlock.Unlock()
			break
		}

		// Optimistically release the read lock if a response has already
		// been received but the current operation is not the target for it.
		c.rlock.Unlock()
	}

	c.leave()
	return
}

func (c *Conn) requestHeader(apiKey apiKey, apiVersion apiVersion, correlationID int32) requestHeader {
	return requestHeader{
		ApiKey:        int16(apiKey),
		ApiVersion:    int16(apiVersion),
		CorrelationID: correlationID,
		ClientID:      c.clientID,
	}
}

func (c *Conn) ApiVersions() ([]ApiVersion, error) {
	deadline := &c.rdeadline

	if deadline.deadline().IsZero() {
		// ApiVersions is called automatically when API version negotiation
		// needs to happen, so we are not guaranteed that a read deadline has
		// been set yet. Fallback to use the write deadline in case it was
		// set, for example when version negotiation is initiated during a
		// produce request.
		deadline = &c.wdeadline
	}

	id, err := c.doRequest(deadline, func(_ time.Time, id int32) error {
		h := requestHeader{
			ApiKey:        int16(apiVersions),
			ApiVersion:    int16(v0),
			CorrelationID: id,
			ClientID:      c.clientID,
		}
		h.Size = (h.size() - 4)
		h.writeTo(&c.wb)
		return c.wbuf.Flush()
	})
	if err != nil {
		return nil, err
	}

	_, size, lock, err := c.waitResponse(deadline, id)
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	var errorCode int16
	if size, err = readInt16(&c.rbuf, size, &errorCode); err != nil {
		return nil, err
	}
	var arrSize int32
	if size, err = readInt32(&c.rbuf, size, &arrSize); err != nil {
		return nil, err
	}
	r := make([]ApiVersion, arrSize)
	for i := 0; i < int(arrSize); i++ {
		if size, err = readInt16(&c.rbuf, size, &r[i].ApiKey); err != nil {
			return nil, err
		}
		if size, err = readInt16(&c.rbuf, size, &r[i].MinVersion); err != nil {
			return nil, err
		}
		if size, err = readInt16(&c.rbuf, size, &r[i].MaxVersion); err != nil {
			return nil, err
		}
	}

	if errorCode != 0 {
		return r, Error(errorCode)
	}

	return r, nil
}

// connDeadline is a helper type to implement read/write deadline management on
// the kafka connection.
type connDeadline struct {
	mutex sync.Mutex
	value time.Time
	rconn net.Conn
	wconn net.Conn
}

func (d *connDeadline) deadline() time.Time {
	d.mutex.Lock()
	t := d.value
	d.mutex.Unlock()
	return t
}

func (d *connDeadline) setDeadline(t time.Time) {
	d.mutex.Lock()
	d.value = t

	if d.rconn != nil {
		d.rconn.SetReadDeadline(t)
	}

	if d.wconn != nil {
		d.wconn.SetWriteDeadline(t)
	}

	d.mutex.Unlock()
}

func (d *connDeadline) setConnReadDeadline(conn net.Conn) time.Time {
	d.mutex.Lock()
	deadline := d.value
	d.rconn = conn
	d.rconn.SetReadDeadline(deadline)
	d.mutex.Unlock()
	return deadline
}

func (d *connDeadline) setConnWriteDeadline(conn net.Conn) time.Time {
	d.mutex.Lock()
	deadline := d.value
	d.wconn = conn
	d.wconn.SetWriteDeadline(deadline)
	d.mutex.Unlock()
	return deadline
}

func (d *connDeadline) unsetConnReadDeadline() {
	d.mutex.Lock()
	d.rconn = nil
	d.mutex.Unlock()
}

func (d *connDeadline) unsetConnWriteDeadline() {
	d.mutex.Lock()
	d.wconn = nil
	d.mutex.Unlock()
}

// saslHandshake sends the SASL handshake message.  This will determine whether
// the Mechanism is supported by the cluster.  If it's not, this function will
// error out with UnsupportedSASLMechanism.
//
// If the mechanism is unsupported, the handshake request will reply with the
// list of the cluster's configured mechanisms, which could potentially be used
// to facilitate negotiation.  At the moment, we are not negotiating the
// mechanism as we believe that brokers are usually known to the client, and
// therefore the client should already know which mechanisms are supported.
//
// See http://kafka.apache.org/protocol.html#The_Messages_SaslHandshake
func (c *Conn) saslHandshake(mechanism string) error {
	// The wire format for V0 and V1 is identical, but the version
	// number will affect how the SASL authentication
	// challenge/responses are sent
	var resp saslHandshakeResponseV0

	version, err := c.negotiateVersion(saslHandshake, v0, v1)
	if err != nil {
		return err
	}

	err = c.writeOperation(
		func(deadline time.Time, id int32) error {
			return c.writeRequest(saslHandshake, version, id, &saslHandshakeRequestV0{Mechanism: mechanism})
		},
		func(deadline time.Time, size int) error {
			return expectZeroSize(func() (int, error) {
				return (&resp).readFrom(&c.rbuf, size)
			}())
		},
	)
	if err == nil && resp.ErrorCode != 0 {
		err = Error(resp.ErrorCode)
	}
	return err
}

// saslAuthenticate sends the SASL authenticate message.  This function must
// be immediately preceded by a successful saslHandshake.
//
// See http://kafka.apache.org/protocol.html#The_Messages_SaslAuthenticate
func (c *Conn) saslAuthenticate(data []byte) ([]byte, error) {
	// if we sent a v1 handshake, then we must encapsulate the authentication
	// request in a saslAuthenticateRequest.  otherwise, we read and write raw
	// bytes.
	version, err := c.negotiateVersion(saslHandshake, v0, v1)
	if err != nil {
		return nil, err
	}
	if version == v1 {
		var request = saslAuthenticateRequestV0{Data: data}
		var response saslAuthenticateResponseV0

		err := c.writeOperation(
			func(deadline time.Time, id int32) error {
				return c.writeRequest(saslAuthenticate, v0, id, request)
			},
			func(deadline time.Time, size int) error {
				return expectZeroSize(func() (remain int, err error) {
					return (&response).readFrom(&c.rbuf, size)
				}())
			},
		)
		if err == nil && response.ErrorCode != 0 {
			err = Error(response.ErrorCode)
		}
		return response.Data, err
	}

	// fall back to opaque bytes on the wire.  the broker is expecting these if
	// it just processed a v0 sasl handshake.
	c.wb.writeInt32(int32(len(data)))
	if _, err := c.wb.Write(data); err != nil {
		return nil, err
	}
	if err := c.wb.Flush(); err != nil {
		return nil, err
	}

	var respLen int32
	if _, err := readInt32(&c.rbuf, 4, &respLen); err != nil {
		return nil, err
	}

	resp, _, err := readNewBytes(&c.rbuf, int(respLen), int(respLen))
	return resp, err
}
