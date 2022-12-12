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
	"encoding/binary"
	rio "github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/racing"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
	"net"
	"time"
)

// dechunkMessage takes a buffer to be reused and returns the reusable buffer
// (might have been reallocated to handle growth), the message buffer and
// error.
// Reads will race against the provided context ctx
// If the server provides the connection read timeout hint readTimeout, a new context will be created from that timeout
// and the user-provided context ctx before every read
func dechunkMessage(
	ctx context.Context,
	conn net.Conn,
	msgBuf []byte,
	readTimeout time.Duration,
	logger log.Logger,
	logName string,
	logId string) ([]byte, []byte, error) {

	sizeBuf := []byte{0x00, 0x00}
	off := 0

	reader := rio.NewRacingReader(conn)

	for {
		updatedCtx, cancelFunc := newContext(ctx, readTimeout, logger, logName, logId)
		_, err := reader.ReadFull(updatedCtx, sizeBuf)
		if err != nil {
			return msgBuf, nil, processReadError(err, ctx, readTimeout)
		}
		if cancelFunc != nil { // reading has been completed, time to release the context
			cancelFunc()
		}
		chunkSize := int(binary.BigEndian.Uint16(sizeBuf))
		if chunkSize == 0 {
			if off > 0 {
				return msgBuf, msgBuf[:off], nil
			}
			// Got a nop chunk
			continue
		}

		// Need to expand buffer
		if (off + chunkSize) > cap(msgBuf) {
			newMsgBuf := make([]byte, (off+chunkSize)+4096)
			copy(newMsgBuf, msgBuf)
			msgBuf = newMsgBuf
		}
		// Read the chunk into buffer
		updatedCtx, cancelFunc = newContext(ctx, readTimeout, logger, logName, logId)
		_, err = reader.ReadFull(updatedCtx, msgBuf[off:(off+chunkSize)])
		if err != nil {
			return msgBuf, nil, processReadError(err, ctx, readTimeout)
		}
		if cancelFunc != nil { // reading has been completed, time to release the context
			cancelFunc()
		}
		off += chunkSize
	}
}

// newContext computes a new context and cancel function if a readTimeout is set
func newContext(
	ctx context.Context,
	readTimeout time.Duration,
	logger log.Logger,
	logName string,
	logId string) (context.Context, context.CancelFunc) {

	if readTimeout >= 0 {
		newCtx, cancelFunc := context.WithTimeout(ctx, readTimeout)
		logger.Debugf(logName, logId,
			"read timeout of %s applied, chunk read deadline is now: %s",
			readTimeout.String(),
			deadlineOf(newCtx),
		)
		return newCtx, cancelFunc
	}
	return ctx, nil
}

func processReadError(err error, ctx context.Context, readTimeout time.Duration) error {
	if IsTimeoutError(err) {
		return &ConnectionReadTimeout{
			userContext: ctx,
			readTimeout: readTimeout,
			err:         err,
		}
	}
	if err == context.Canceled {
		return &ConnectionReadCanceled{
			err: err,
		}
	}
	return err
}

func deadlineOf(ctx context.Context) string {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return "N/A (no deadline set)"
	}
	return deadline.String()
}
