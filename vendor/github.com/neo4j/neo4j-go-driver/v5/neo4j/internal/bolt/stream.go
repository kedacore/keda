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
	"container/list"
	"errors"
	idb "github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
)

type stream struct {
	keys      []string
	fifo      list.List
	sum       *db.Summary
	err       error
	qid       int64
	fetchSize int
	key       int64
}

// Acts on buffered data, first return value indicates if buffering
// is active or not.
func (s *stream) bufferedNext() (bool, *db.Record, *db.Summary, error) {
	e := s.fifo.Front()
	if e != nil {
		s.fifo.Remove(e)
		return true, e.Value.(*db.Record), nil, nil
	}
	if s.err != nil {
		return true, nil, nil, s.err
	}
	if s.sum != nil {
		return true, nil, s.sum, nil
	}

	return false, nil, nil, nil
}

// Delayed error until fifo emptied
func (s *stream) Err() error {
	if s.fifo.Len() > 0 {
		return nil
	}
	return s.err
}

func (s *stream) push(rec *db.Record) {
	s.fifo.PushBack(rec)
}

// Only need to keep track of current stream. Client keeps track of other
// open streams and a key in each stream is used to validate if it belongs to
// current bolt connection or not.
type openstreams struct {
	curr *stream
	num  int
	key  int64
}

var (
	errInvalidStream = errors.New("invalid stream handle")
)

// Adds a new open stream and sets it as current.
// There should NOT be a current stream .
func (o *openstreams) attach(s *stream) {
	if o.curr != nil {
		return
	}
	// Track number of open streams and set the stream as current
	o.num++
	o.curr = s
	s.key = o.key
}

// Detaches the current stream from being current and
// removes it from set of open streams it is no longer open.
// The stream should be either in failed state or completed.
func (o *openstreams) detach(sum *db.Summary, err error) {
	if o.curr == nil {
		return
	}

	o.curr.sum = sum
	o.curr.err = err
	o.remove(o.curr)
}

// Streams can be paused when they have received a "has_more" response from server
// Pauses the current stream
func (o *openstreams) pause() {
	o.curr = nil
}

// When resuming a stream a new PULL message needs to be sent.
func (o *openstreams) resume(s *stream) {
	if o.curr != nil {
		return
	}
	o.curr = s
}

// Removes the stream by disabling its key and removing it from the count of streams.
// If the stream is current the current is set to nil.
func (o *openstreams) remove(s *stream) {
	o.num--
	s.key = 0
	if o.curr == s {
		o.curr = nil
	}
}

func (o *openstreams) reset() {
	o.num = 0
	o.curr = nil
	o.key = time.Now().UnixNano()
}

// Checks that the handle represents a stream but not necessarily a stream belonging
// to this set of open streams.
func (o openstreams) getUnsafe(h idb.StreamHandle) (*stream, error) {
	stream, ok := h.(*stream)
	if !ok || stream == nil {
		return nil, errInvalidStream
	}
	return stream, nil
}

func (o openstreams) isSafe(s *stream) error {
	if s.key == o.key {
		return nil
	}
	return errInvalidStream
}
