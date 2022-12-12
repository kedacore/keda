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
	"io"
)

type chunker struct {
	buf    []byte
	sizes  []int
	offset int
}

func newChunker() chunker {
	return chunker{
		buf:    make([]byte, 0, 1024),
		sizes:  make([]int, 0, 3),
		offset: 0,
	}
}

func (c *chunker) beginMessage() {
	// Space for length of next message
	c.buf = append(c.buf, 0, 0)
	c.offset += 2
}

func (c *chunker) endMessage() {
	// Calculate size and stash it
	size := len(c.buf) - c.offset
	c.offset += size
	c.sizes = append(c.sizes, size)

	// Add zero chunk to mark end of message
	c.buf = append(c.buf, 0, 0)
	c.offset += 2
}

func (c *chunker) send(ctx context.Context, wr io.Writer) error {
	// Try to make as few writes as possible to reduce network overhead
	// Whenever we encounter a message that is bigger than max chunk size we need
	// to write and make a new chunk
	start := 0
	end := 0

	writer := rio.NewRacingWriter(wr)

	for _, size := range c.sizes {
		if size <= 0xffff {
			binary.BigEndian.PutUint16(c.buf[end:], uint16(size))
			// Size + message + end of message marker
			end += 2 + size + 2
		} else {
			// Could be a message that ranges over multiple chunks
			for size > 0xffff {
				c.buf[end] = 0xff
				c.buf[end+1] = 0xff
				// Size + message
				end += 2 + 0xffff

				_, err := writer.Write(ctx, c.buf[start:end])
				if err != nil {
					return processWriteError(err, ctx)
				}
				// Reuse part of buffer that has already been written to specify size
				// of the chunk
				end -= 2
				start = end
				size -= 0xffff
			}
			binary.BigEndian.PutUint16(c.buf[end:], uint16(size))
			// Size + message + end of message marker
			end += 2 + size + 2
		}
	}

	if end > start {
		_, err := writer.Write(ctx, c.buf[start:end])
		if err != nil {
			return processWriteError(err, ctx)
		}
	}

	// Prepare for reuse
	c.offset = 0
	c.buf = c.buf[:0]
	c.sizes = c.sizes[:0]

	return nil
}

func processWriteError(err error, ctx context.Context) error {
	if IsTimeoutError(err) {
		return &ConnectionWriteTimeout{
			userContext: ctx,
			err:         err,
		}
	}
	if err == context.Canceled {
		return &ConnectionWriteCanceled{
			err: err,
		}
	}
	return err
}
