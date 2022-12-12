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

package neo4j

import (
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	idb "github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
)

type ResultWithContext interface {
	// Keys returns the keys available on the result set.
	Keys() ([]string, error)
	// NextRecord returns true if there is a record to be processed, record parameter is set
	// to point to current record.
	NextRecord(ctx context.Context, record **Record) bool
	// Next returns true only if there is a record to be processed.
	Next(ctx context.Context) bool
	// PeekRecord returns true if there is a record after the current one to be processed without advancing the record
	// stream, record parameter is set to point to that record if present.
	PeekRecord(ctx context.Context, record **Record) bool
	// Peek returns true only if there is a record after the current one to be processed without advancing the record
	// stream
	Peek(ctx context.Context) bool
	// Err returns the latest error that caused this Next to return false.
	Err() error
	// Record returns the current record.
	Record() *Record
	// Collect fetches all remaining records and returns them.
	Collect(ctx context.Context) ([]*Record, error)
	// Single returns one and only one record from the stream.
	// If the result stream contains zero or more than one records, error is returned.
	Single(ctx context.Context) (*Record, error)
	// Consume discards all remaining records and returns the summary information
	// about the statement execution.
	Consume(ctx context.Context) (ResultSummary, error)
	// IsOpen determines whether this result cursor is available
	IsOpen() bool
	buffer(ctx context.Context)
	legacy() Result
}

const consumedResultError = "result cursor is not available anymore"

type resultWithContext struct {
	conn                 idb.Connection
	streamHandle         idb.StreamHandle
	cypher               string
	params               map[string]any
	record               *Record
	summary              *db.Summary
	err                  error
	peekedRecord         *Record
	peekedSummary        *db.Summary
	peeked               bool
	afterConsumptionHook func()
}

func newResultWithContext(connection idb.Connection, stream idb.StreamHandle, cypher string, params map[string]any, afterConsumptionHook func()) ResultWithContext {
	return &resultWithContext{
		conn:                 connection,
		streamHandle:         stream,
		cypher:               cypher,
		params:               params,
		afterConsumptionHook: afterConsumptionHook,
	}
}

func (r *resultWithContext) Keys() ([]string, error) {
	return r.conn.Keys(r.streamHandle)
}

func (r *resultWithContext) NextRecord(ctx context.Context, out **Record) bool {
	hasNext := r.Next(ctx)
	if out != nil {
		*out = r.record
	}
	return hasNext
}

func (r *resultWithContext) Next(ctx context.Context) bool {
	r.checkOpen()
	if r.err != nil {
		return false
	}
	r.advance(ctx)
	if r.summary != nil {
		r.callAfterConsumptionHook()
	}
	return r.record != nil
}

func (r *resultWithContext) PeekRecord(ctx context.Context, out **Record) bool {
	hasNext := r.Peek(ctx)
	if out != nil {
		*out = r.peekedRecord
	}
	return hasNext
}

func (r *resultWithContext) Peek(ctx context.Context) bool {
	r.checkOpen()
	if r.err != nil {
		return false
	}
	r.peek(ctx)
	return r.peekedRecord != nil
}

func (r *resultWithContext) Err() error {
	return wrapError(r.err)
}

func (r *resultWithContext) Record() *Record {
	if r.peekedRecord != nil {
		return r.peekedRecord
	}
	return r.record
}

func (r *resultWithContext) Collect(ctx context.Context) ([]*Record, error) {
	recs := make([]*Record, 0, 1024)
	for r.summary == nil && r.err == nil {
		r.advance(ctx)
		if r.record != nil {
			recs = append(recs, r.record)
		}
	}
	if r.err != nil {
		return nil, wrapError(r.err)
	}
	r.callAfterConsumptionHook()
	return recs, nil
}

func (r *resultWithContext) Single(ctx context.Context) (*Record, error) {
	// Try retrieving the single record
	r.advance(ctx)
	if r.err != nil {
		return nil, wrapError(r.err)
	}
	if r.summary != nil {
		r.err = &UsageError{Message: "Result contains no more records"}
		return nil, r.err
	}

	// This is the potential single record
	single := r.record

	// Probe connection for more records
	r.advance(ctx)
	if r.record != nil {
		// There were more records, consume the stream since the user didn't
		// expect more records and should therefore not use them.
		r.summary, _ = r.conn.Consume(ctx, r.streamHandle)
		r.err = &UsageError{Message: "Result contains more than one record"}
		r.record = nil
		return nil, r.err
	}
	if r.err != nil {
		// Might be more records or not, anyway something is bad.
		// Both r.record and r.summary are nil at this point which is good.
		return nil, wrapError(r.err)
	}
	// We got the expected summary
	// r.record contains the single record and r.summary the summary.
	r.record = single
	r.callAfterConsumptionHook()
	return single, nil
}

func (r *resultWithContext) Consume(ctx context.Context) (ResultSummary, error) {
	// Already failed, reuse the internal error, might have been
	// set by Single to indicate some kind of usage error that "destroyed"
	// the result.
	if r.err != nil {
		return nil, wrapError(r.err)
	}

	r.record = nil
	r.summary, r.err = r.conn.Consume(ctx, r.streamHandle)
	if r.err != nil {
		return nil, wrapError(r.err)
	}
	r.callAfterConsumptionHook()
	return r.toResultSummary(), nil
}

func (r *resultWithContext) IsOpen() bool {
	return r.isOpen()
}

func (r *resultWithContext) legacy() Result {
	return &result{delegate: r}
}

func (r *resultWithContext) buffer(ctx context.Context) {
	if r.err = r.conn.Buffer(ctx, r.streamHandle); r.err == nil {
		r.callAfterConsumptionHook()
	}
}

func (r *resultWithContext) toResultSummary() ResultSummary {
	return &resultSummary{
		sum:    r.summary,
		cypher: r.cypher,
		params: r.params,
	}
}

func (r *resultWithContext) advance(ctx context.Context) {
	if r.peeked {
		r.record, r.peekedRecord = r.peekedRecord, nil
		r.summary, r.peekedSummary = r.peekedSummary, nil
		r.peeked = false
	} else {
		r.record, r.summary, r.err = r.conn.Next(ctx, r.streamHandle)
	}
}

func (r *resultWithContext) peek(ctx context.Context) {
	if !r.peeked {
		r.peekedRecord, r.peekedSummary, r.err = r.conn.Next(ctx, r.streamHandle)
		r.peeked = true
	}
}

func (r *resultWithContext) checkOpen() {
	alreadyChecked := r.err != nil && r.err.Error() == consumedResultError
	if !alreadyChecked && !r.isOpen() {
		r.err = &UsageError{Message: consumedResultError}
	}
}

func (r *resultWithContext) isOpen() bool {
	return r.summary == nil
}

func (r *resultWithContext) callAfterConsumptionHook() {
	if r.afterConsumptionHook == nil {
		return
	}
	r.afterConsumptionHook()
	r.afterConsumptionHook = nil
}
