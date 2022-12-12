/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package bolt

import (
	"context"
	"errors"
	"fmt"
	idb "github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"net"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/packstream"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

const (
	bolt5Ready        = iota // Ready for use
	bolt5Streaming           // Receiving result from auto commit query
	bolt5Tx                  // Transaction pending
	bolt5StreamingTx         // Receiving result from a query within a transaction
	bolt5Failed              // Recoverable error, needs reset
	bolt5Dead                // Non recoverable protocol or connection error
	bolt5Unauthorized        // Initial state, not sent hello message with authentication
)

// Default fetch size
const bolt5FetchSize = 1000

type internalTx5 struct {
	mode             idb.AccessMode
	bookmarks        []string
	timeout          time.Duration
	txMeta           map[string]any
	databaseName     string
	impersonatedUser string
}

func (i *internalTx5) toMeta() map[string]any {
	meta := map[string]any{}
	if i.mode == idb.ReadMode {
		meta["mode"] = "r"
	}
	if len(i.bookmarks) > 0 {
		meta["bookmarks"] = i.bookmarks
	}
	ms := int(i.timeout.Nanoseconds() / 1e6)
	if ms > 0 {
		meta["tx_timeout"] = ms
	}
	if len(i.txMeta) > 0 {
		meta["tx_metadata"] = i.txMeta
	}
	if i.databaseName != idb.DefaultDatabase {
		meta["db"] = i.databaseName
	}
	if i.impersonatedUser != "" {
		meta["imp_user"] = i.impersonatedUser
	}
	return meta
}

type bolt5 struct {
	state         int
	txId          idb.TxHandle
	streams       openstreams
	conn          net.Conn
	serverName    string
	out           outgoing
	in            incoming
	connId        string
	logId         string
	serverVersion string
	tfirst        int64  // Time that server started streaming
	bookmark      string // Last bookmark
	birthDate     time.Time
	log           log.Logger
	databaseName  string
	err           error // Last fatal error
	minor         int
	lastQid       int64 // Last seen qid
	idleDate      time.Time
}

func NewBolt5(serverName string, conn net.Conn, logger log.Logger, boltLog log.BoltLogger) *bolt5 {
	now := time.Now()
	b := &bolt5{
		state:      bolt5Unauthorized,
		conn:       conn,
		serverName: serverName,
		birthDate:  now,
		idleDate:   now,
		log:        logger,
		streams:    openstreams{},
		in: incoming{
			buf: make([]byte, 4096),
			hyd: hydrator{
				boltLogger: boltLog,
				boltMajor:  5,
				useUtc:     true,
			},
			connReadTimeout: -1,
			logger:          logger,
			logName:         log.Bolt5,
		},
		lastQid: -1,
	}
	b.out = outgoing{
		chunker:    newChunker(),
		packer:     packstream.Packer{},
		onErr:      func(err error) { b.setError(err, true) },
		boltLogger: boltLog,
		useUtc:     true,
	}

	return b
}

func (b *bolt5) checkStreams() {
	if b.streams.num <= 0 {
		// Perform state transition from streaming, if in that state otherwise keep the current
		// state as we are in some kind of bad shape
		switch b.state {
		case bolt5StreamingTx:
			b.state = bolt5Tx
		case bolt5Streaming:
			b.state = bolt5Ready
		}
	}
}

func (b *bolt5) ServerName() string {
	return b.serverName
}

func (b *bolt5) ServerVersion() string {
	return b.serverVersion
}

// Sets b.err and b.state to bolt5Failed or bolt5Dead when fatal is true.
func (b *bolt5) setError(err error, fatal bool) {
	// Has no effect, can reduce nested ifs
	if err == nil {
		return
	}

	// No previous error
	if b.err == nil {
		b.err = err
		b.state = bolt5Failed
	}

	// Increase severity even if it was a previous error
	if fatal {
		b.state = bolt5Dead
	}

	// Forward error to current stream if there is one
	if b.streams.curr != nil {
		b.streams.detach(nil, err)
		b.checkStreams()
	}

	// Do not log big cypher statements as errors
	neo4jErr, casted := err.(*db.Neo4jError)
	if casted && neo4jErr.Classification() == "ClientError" {
		b.log.Debugf(log.Bolt5, b.logId, "%s", err)
	} else {
		b.log.Error(log.Bolt5, b.logId, err)
	}
}

func (b *bolt5) receiveMsg(ctx context.Context) any {
	// Potentially dangerous to receive when an error has occurred, could hang.
	// Important, a lot of code has been simplified relying on this check.
	if b.err != nil {
		return nil
	}

	msg, err := b.in.next(ctx, b.conn)
	b.setError(err, true)
	if err == nil {
		b.idleDate = time.Now()
	}
	return msg
}

// Receives a message that is assumed to be a success response or a failure
// in response to a sent command. Sets b.err and b.state on failure
func (b *bolt5) receiveSuccess(ctx context.Context) *success {
	msg := b.receiveMsg(ctx)
	if b.err != nil {
		return nil
	}

	switch v := msg.(type) {
	case *success:
		if v.qid > -1 {
			b.lastQid = v.qid
		}
		return v
	case *db.Neo4jError:
		b.setError(v, isFatalError(v))
		return nil
	default:
		// Unexpected message received
		b.setError(errors.New("expected success or database error"), true)
		return nil
	}
}

func (b *bolt5) Connect(ctx context.Context, minor int, auth map[string]any, userAgent string, routingContext map[string]string) error {
	if err := b.assertState(bolt5Unauthorized); err != nil {
		return err
	}

	// Prepare hello message
	hello := map[string]any{
		"user_agent": userAgent,
	}
	if routingContext != nil {
		hello["routing"] = routingContext
	}
	// Merge authentication keys into hello, avoid overwriting existing keys
	for k, v := range auth {
		_, exists := hello[k]
		if !exists {
			hello[k] = v
		}
	}

	// Send hello message and wait for confirmation
	b.out.appendHello(hello)
	b.out.send(ctx, b.conn)
	succ := b.receiveSuccess(ctx)
	if b.err != nil {
		return b.err
	}

	b.connId = succ.connectionId
	b.serverVersion = succ.server

	// Construct log identity
	connectionLogId := fmt.Sprintf("%s@%s", b.connId, b.serverName)
	b.logId = connectionLogId
	b.in.hyd.logId = connectionLogId
	b.in.logId = connectionLogId
	b.out.logId = connectionLogId

	b.initializeReadTimeoutHint(succ.configurationHints)
	// Transition into ready state
	b.state = bolt5Ready
	b.minor = minor
	b.streams.reset()
	b.log.Infof(log.Bolt5, b.logId, "Connected")
	return nil
}

func (b *bolt5) TxBegin(ctx context.Context, txConfig idb.TxConfig) (idb.TxHandle, error) {
	// Ok, to begin transaction while streaming auto-commit, just empty the stream and continue.
	if b.state == bolt5Streaming {
		if b.bufferStream(ctx); b.err != nil {
			return 0, b.err
		}
	}
	// Makes all outstanding streams invalid
	b.streams.reset()

	if err := b.assertState(bolt5Ready); err != nil {
		return 0, err
	}

	tx := internalTx5{
		mode:             txConfig.Mode,
		bookmarks:        txConfig.Bookmarks,
		timeout:          txConfig.Timeout,
		txMeta:           txConfig.Meta,
		databaseName:     b.databaseName,
		impersonatedUser: txConfig.ImpersonatedUser,
	}

	b.out.appendBegin(tx.toMeta())
	b.out.send(ctx, b.conn)
	b.receiveSuccess(ctx)
	if b.err != nil {
		return 0, b.err
	}
	b.state = bolt5Tx
	b.txId = idb.TxHandle(time.Now().Unix())
	return b.txId, nil
}

// Should NOT set b.err or change b.state as this is used to guard against
// misuse from clients that stick to their connections when they shouldn't.
func (b *bolt5) assertTxHandle(h1, h2 idb.TxHandle) error {
	if h1 != h2 {
		err := errors.New(InvalidTransactionError)
		b.log.Error(log.Bolt5, b.logId, err)
		return err
	}
	return nil
}

// Should NOT set b.err or b.state since the connection is still valid
func (b *bolt5) assertState(allowed ...int) error {
	// Forward prior error instead, this former error is probably the
	// root cause of any state error. Like a call to Run with malformed
	// cypher causes an error and another call to Commit would cause the
	// state to be wrong. Do not log this.
	if b.err != nil {
		return b.err
	}
	for _, a := range allowed {
		if b.state == a {
			return nil
		}
	}
	err := fmt.Errorf("invalid state %d, expected: %+v", b.state, allowed)
	b.log.Error(log.Bolt5, b.logId, err)
	return err
}

func (b *bolt5) TxCommit(ctx context.Context, txh idb.TxHandle) error {
	if err := b.assertTxHandle(b.txId, txh); err != nil {
		return err
	}

	// Consume pending stream if any to turn state from streamingtx to tx
	// Access to the streams outside tx boundary is not allowed, therefore we should discard
	// the stream (not buffer).
	if b.discardAllStreams(ctx); b.err != nil {
		return b.err
	}

	// Should be in vanilla tx state now
	if err := b.assertState(bolt5Tx); err != nil {
		return err
	}

	// Send request to server to commit
	b.out.appendCommit()
	b.out.send(ctx, b.conn)
	succ := b.receiveSuccess(ctx)
	if b.err != nil {
		return b.err
	}
	// Keep track of bookmark
	if len(succ.bookmark) > 0 {
		b.bookmark = succ.bookmark
	}

	// Transition into ready state
	b.state = bolt5Ready
	return nil
}

func (b *bolt5) TxRollback(ctx context.Context, txh idb.TxHandle) error {
	if err := b.assertTxHandle(b.txId, txh); err != nil {
		return err
	}

	// Can not send rollback while still streaming, consume to turn state into tx
	// Access to the streams outside tx boundary is not allowed, therefore we should discard
	// the stream (not buffer).
	if b.discardAllStreams(ctx); b.err != nil {
		return b.err
	}

	// Should be in vanilla tx state now
	if err := b.assertState(bolt5Tx); err != nil {
		return err
	}

	// Send rollback request to server
	b.out.appendRollback()
	b.out.send(ctx, b.conn)
	if b.receiveSuccess(ctx); b.err != nil {
		return b.err
	}

	b.state = bolt5Ready
	return nil
}

// Discards all records in current stream if in streaming state and there is a current stream.
func (b *bolt5) discardStream(ctx context.Context) {
	if b.state != bolt5Streaming && b.state != bolt5StreamingTx {
		return
	}

	stream := b.streams.curr
	if stream == nil {
		return
	}

	discarded := false
	for {
		_, batch, sum := b.receiveNext(ctx)
		if batch {
			if discarded {
				// Response to discard, see below
				b.streams.remove(stream)
				b.checkStreams()
				return
			}
			// Discard all! After this the next receive will get another batch
			// as a response to the discard, we need to keep track of that we
			// already sent a discard.
			discarded = true
			stream.fetchSize = -1
			if b.state == bolt5StreamingTx && stream.qid != b.lastQid {
				b.out.appendDiscardNQid(stream.fetchSize, stream.qid)
			} else {
				b.out.appendDiscardN(stream.fetchSize)
			}
			b.out.send(ctx, b.conn)
		} else if sum != nil || b.err != nil {
			// Stream is detached in receiveNext
			return
		}
	}
}

func (b *bolt5) discardAllStreams(ctx context.Context) {
	if b.state != bolt5Streaming && b.state != bolt5StreamingTx {
		return
	}

	// Discard current
	b.discardStream(ctx)
	b.streams.reset()
	b.checkStreams()
}

// Sends a PULL n request to server. State should be streaming and there should be a current stream.
func (b *bolt5) sendPullN(ctx context.Context) {
	_ = b.assertState(bolt5Streaming, bolt5StreamingTx)
	if b.state == bolt5Streaming {
		b.out.appendPullN(b.streams.curr.fetchSize)
		b.out.send(ctx, b.conn)
	} else if b.state == bolt5StreamingTx {
		fetchSize := b.streams.curr.fetchSize
		if b.streams.curr.qid == b.lastQid {
			b.out.appendPullN(fetchSize)
		} else {
			b.out.appendPullNQid(fetchSize, b.streams.curr.qid)
		}
		b.out.send(ctx, b.conn)
	}
}

// Collects all records in current stream if in streaming state and there is a current stream.
func (b *bolt5) bufferStream(ctx context.Context) {
	stream := b.streams.curr
	if stream == nil {
		return
	}

	// Buffer current batch and start infinite batch and/or buffer the infinite batch
	for {
		rec, batch, _ := b.receiveNext(ctx)
		if rec != nil {
			stream.push(rec)
		} else if batch {
			stream.fetchSize = -1
			b.sendPullN(ctx)
		} else {
			// Either summary or an error
			return
		}
	}
}

// Prepares the current stream for being switched out by collecting all records in the current
// stream up until the next batch. Assumes that we are in a streaming state.
func (b *bolt5) pauseStream(ctx context.Context) {
	stream := b.streams.curr
	if stream == nil {
		return
	}

	for {
		rec, batch, _ := b.receiveNext(ctx)
		if rec != nil {
			stream.push(rec)
		} else if batch {
			b.streams.pause()
			return
		} else {
			// Either summary or an error
			return
		}
	}
}

func (b *bolt5) resumeStream(ctx context.Context, s *stream) {
	b.streams.resume(s)
	b.sendPullN(ctx)
	if b.err != nil {
		return
	}
}

func (b *bolt5) run(ctx context.Context, cypher string, params map[string]any, fetchSize int, tx *internalTx5) (*stream, error) {
	// If already streaming, consume the whole thing first
	if b.state == bolt5Streaming {
		if b.bufferStream(ctx); b.err != nil {
			return nil, b.err
		}
	} else if b.state == bolt5StreamingTx {
		if b.pauseStream(ctx); b.err != nil {
			return nil, b.err
		}
	}

	if err := b.assertState(bolt5Tx, bolt5Ready, bolt5StreamingTx); err != nil {
		return nil, err
	}

	// Transaction metadata, used either in lazily started transaction or to run message.
	var meta map[string]any
	if tx != nil {
		meta = tx.toMeta()
	}

	// Append run message
	b.out.appendRun(cypher, params, meta)

	// Ensure that fetchSize is in a valid range
	switch {
	case fetchSize < 0:
		fetchSize = -1
	case fetchSize == 0:
		fetchSize = bolt5FetchSize
	}
	// Append pull message and send it along with other pending messages
	b.out.appendPullN(fetchSize)
	b.out.send(ctx, b.conn)

	// Receive confirmation of run message
	succ := b.receiveSuccess(ctx)
	if b.err != nil {
		// If failed with a database error, there will be an ignored response for the
		// pull message as well, this will be cleaned up by Reset
		return nil, b.err
	}
	// Extract the RUN response from success response
	b.tfirst = succ.tfirst
	// Change state to streaming
	if b.state == bolt5Ready {
		b.state = bolt5Streaming
	} else {
		b.state = bolt5StreamingTx
	}

	// Create a stream representation, set it to current and track it
	stream := &stream{keys: succ.fields, qid: succ.qid, fetchSize: fetchSize}
	b.streams.attach(stream)
	// No need to check streams state, we know we are streaming

	return stream, nil
}

func (b *bolt5) Run(ctx context.Context, cmd idb.Command,
	txConfig idb.TxConfig) (idb.StreamHandle, error) {
	if err := b.assertState(bolt5Streaming, bolt5Ready); err != nil {
		return nil, err
	}

	tx := internalTx5{
		mode:             txConfig.Mode,
		bookmarks:        txConfig.Bookmarks,
		timeout:          txConfig.Timeout,
		txMeta:           txConfig.Meta,
		databaseName:     b.databaseName,
		impersonatedUser: txConfig.ImpersonatedUser,
	}
	stream, err := b.run(ctx, cmd.Cypher, cmd.Params, cmd.FetchSize, &tx)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (b *bolt5) RunTx(ctx context.Context, txh idb.TxHandle, cmd idb.Command) (idb.StreamHandle, error) {
	if err := b.assertTxHandle(b.txId, txh); err != nil {
		return nil, err
	}

	stream, err := b.run(ctx, cmd.Cypher, cmd.Params, cmd.FetchSize, nil)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (b *bolt5) Keys(streamHandle idb.StreamHandle) ([]string, error) {
	// Don't care about if the stream is the current or even if it belongs to this connection.
	// Do NOT set b.err for this error
	stream, err := b.streams.getUnsafe(streamHandle)
	if err != nil {
		return nil, err
	}
	return stream.keys, nil
}

// Next reads one record from the stream.
func (b *bolt5) Next(ctx context.Context, streamHandle idb.StreamHandle) (
	*db.Record, *db.Summary, error) {
	// Do NOT set b.err for this error
	stream, err := b.streams.getUnsafe(streamHandle)
	if err != nil {
		return nil, nil, err
	}

	// Buffered stream or someone else's stream, doesn't matter...
	// Summary and error are considered buffered as well.
	buf, rec, sum, err := stream.bufferedNext()
	if buf {
		return rec, sum, err
	}

	// Make sure that the stream belongs to this bolt instance otherwise we might mess
	// up the internal state machine. If clients stick to the streams out of
	// transaction scope or after the connection been sent back to the pool we might end
	// up here.
	if err = b.streams.isSafe(stream); err != nil {
		return nil, nil, err
	}

	// If the stream isn't the current we must finish what we're doing with the current stream
	// and make it the current one.
	if stream != b.streams.curr {
		b.pauseStream(ctx)
		if b.err != nil {
			return nil, nil, b.err
		}
		b.resumeStream(ctx, stream)
	}

	rec, batchCompleted, sum := b.receiveNext(ctx)
	if batchCompleted {
		b.sendPullN(ctx)
		if b.err != nil {
			return nil, nil, b.err
		}
		rec, _, sum = b.receiveNext(ctx)
	}
	return rec, sum, b.err
}

func (b *bolt5) Consume(ctx context.Context, streamHandle idb.StreamHandle) (
	*db.Summary, error) {
	// Do NOT set b.err for this error
	stream, err := b.streams.getUnsafe(streamHandle)
	if err != nil {
		return nil, err
	}

	// If the stream already is complete we don't care about whom it belongs to
	if stream.sum != nil || stream.err != nil {
		return stream.sum, stream.err
	}

	// Make sure the stream is safe (tied to this bolt instance and scope)
	if err = b.streams.isSafe(stream); err != nil {
		return nil, err
	}

	// We should be streaming otherwise it is an internal error, shouldn't be
	// a safe stream while not streaming.
	if err = b.assertState(bolt5Streaming, bolt5StreamingTx); err != nil {
		return nil, err
	}

	// If the stream isn't current, we need to pause the current one.
	if stream != b.streams.curr {
		b.pauseStream(ctx)
		if b.err != nil {
			return nil, b.err
		}
		b.resumeStream(ctx, stream)
	}

	// If the stream is current, discard everything up to next batch and discard the
	// stream on the server.
	b.discardStream(ctx)
	return stream.sum, stream.err
}

func (b *bolt5) Buffer(ctx context.Context,
	streamHandle idb.StreamHandle) error {
	// Do NOT set b.err for this error
	stream, err := b.streams.getUnsafe(streamHandle)
	if err != nil {
		return err
	}

	// If the stream already is complete we don't care about whom it belongs to
	if stream.sum != nil || stream.err != nil {
		return stream.Err()
	}

	// Make sure the stream is safe
	// Do NOT set b.err for this error
	if err = b.streams.isSafe(stream); err != nil {
		return err
	}

	// We should be streaming otherwise it is an internal error, shouldn't be
	// a safe stream while not streaming.
	if err = b.assertState(bolt5Streaming, bolt5StreamingTx); err != nil {
		return err
	}

	// If the stream isn't current, we need to pause the current one.
	if stream != b.streams.curr {
		b.pauseStream(ctx)
		if b.err != nil {
			return b.err
		}
		b.resumeStream(ctx, stream)
	}

	b.bufferStream(ctx)
	return stream.Err()
}

// Reads one record from the network and returns either a record, a flag that indicates that
// a PULL N batch completed, a summary indicating end of stream or an error.
// Assumes that there is a current stream and that streaming is active.
func (b *bolt5) receiveNext(ctx context.Context) (*db.Record, bool, *db.Summary) {
	res := b.receiveMsg(ctx)
	if b.err != nil {
		return nil, false, nil
	}

	switch x := res.(type) {
	case *db.Record:
		// A new record
		x.Keys = b.streams.curr.keys
		return x, false, nil
	case *success:
		// End of batch or end of stream?
		if x.hasMore {
			// End of batch
			return nil, true, nil
		}
		// End of stream, parse summary. Current implementation never fails.
		sum := x.summary()
		// Add some extras to the summary
		sum.Agent = b.serverVersion
		sum.Major = 5
		sum.Minor = b.minor
		sum.ServerName = b.serverName
		sum.TFirst = b.tfirst
		if len(sum.Bookmark) > 0 {
			b.bookmark = sum.Bookmark
		}
		// Done with this stream
		b.streams.detach(sum, nil)
		b.checkStreams()
		return nil, false, sum
	case *db.Neo4jError:
		b.setError(x, isFatalError(x)) // Will detach the stream
		return nil, false, nil
	default:
		// Unknown territory
		b.setError(errors.New("unknown response"), true)
		return nil, false, nil
	}
}

func (b *bolt5) Bookmark() string {
	return b.bookmark
}

func (b *bolt5) IsAlive() bool {
	return b.state != bolt5Dead
}

func (b *bolt5) HasFailed() bool {
	return b.state == bolt5Failed
}

func (b *bolt5) Birthdate() time.Time {
	return b.birthDate
}

func (b *bolt5) IdleDate() time.Time {
	return b.idleDate
}

func (b *bolt5) Reset(ctx context.Context) {
	defer func() {
		b.log.Debugf(log.Bolt5, b.logId, "Resetting connection internal state")
		b.txId = 0
		b.bookmark = ""
		b.databaseName = idb.DefaultDatabase
		b.err = nil
		b.lastQid = -1
		b.streams.reset()
	}()

	if b.state == bolt5Ready {
		// No need for reset
		return
	}

	b.ForceReset(ctx)
}

func (b *bolt5) ForceReset(ctx context.Context) {
	if b.state == bolt5Dead {
		return
	}

	// Reset any pending error, should be matching bolt5_failed, so
	// it should be recoverable.
	b.err = nil

	// Send the reset message to the server
	b.out.appendReset()
	b.out.send(ctx, b.conn)
	if b.err != nil {
		return
	}

	for {
		msg := b.receiveMsg(ctx)
		if b.err != nil {
			return
		}
		switch x := msg.(type) {
		case *ignored, *db.Record:
			// Command ignored
		case *success:
			if x.isResetResponse() {
				// Reset confirmed
				b.state = bolt5Ready
				return
			}
		default:
			b.state = bolt5Dead
			return
		}
	}
}

func (b *bolt5) GetRoutingTable(ctx context.Context,
	routingContext map[string]string, bookmarks []string, database, impersonatedUser string) (*idb.RoutingTable, error) {
	if err := b.assertState(bolt5Ready); err != nil {
		return nil, err
	}

	b.log.Infof(log.Bolt5, b.logId, "Retrieving routing table")
	extras := map[string]any{}
	if database != idb.DefaultDatabase {
		extras["db"] = database
	}
	if impersonatedUser != "" {
		extras["imp_user"] = impersonatedUser
	}
	b.out.appendRoute(routingContext, bookmarks, extras)
	b.out.send(ctx, b.conn)
	succ := b.receiveSuccess(ctx)
	if b.err != nil {
		return nil, b.err
	}
	return succ.routingTable, nil
}

// Close closes the underlying connection.
// Beware: could be called on another thread when driver is closed.
func (b *bolt5) Close(ctx context.Context) {
	b.log.Infof(log.Bolt5, b.logId, "Close")
	if b.state != bolt5Dead {
		b.out.appendGoodbye()
		b.out.send(ctx, b.conn)
	}
	_ = b.conn.Close()
	b.state = bolt5Dead
}

func (b *bolt5) SelectDatabase(database string) {
	b.databaseName = database
}

func (b *bolt5) SetBoltLogger(boltLogger log.BoltLogger) {
	b.in.hyd.boltLogger = boltLogger
	b.out.boltLogger = boltLogger
}

func (b *bolt5) Version() db.ProtocolVersion {
	return db.ProtocolVersion{
		Major: 5,
		Minor: b.minor,
	}
}

func (b *bolt5) initializeReadTimeoutHint(hints map[string]any) {
	readTimeoutHint, ok := hints[readTimeoutHintName]
	if !ok {
		return
	}
	readTimeout, ok := readTimeoutHint.(int64)
	if !ok {
		b.log.Infof(log.Bolt5, b.logId, `invalid %q value: %v, ignoring hint. Only strictly positive integer values are accepted`, readTimeoutHintName, readTimeoutHint)
		return
	}
	if readTimeout <= 0 {
		b.log.Infof(log.Bolt5, b.logId, `invalid %q integer value: %d. Only strictly positive values are accepted"`, readTimeoutHintName, readTimeout)
		return
	}
	b.in.connReadTimeout = time.Duration(readTimeout) * time.Second
}
