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
)

// Deprecated: use ResultWithContext instead.
// ResultWithContext is created via the context-aware SessionWithContext type.
type Result interface {
	// Keys returns the keys available on the result set.
	Keys() ([]string, error)
	// Next returns true only if there is a record to be processed.
	Next() bool
	// NextRecord returns true if there is a record to be processed, record parameter is set
	// to point to current record.
	NextRecord(record **Record) bool
	// PeekRecord returns true if there is a record after the current one to be processed without advancing the record
	// stream, record parameter is set to point to that record if present.
	PeekRecord(record **Record) bool
	// Err returns the latest error that caused this Next to return false.
	Err() error
	// Record returns the current record.
	Record() *Record
	// Collect fetches all remaining records and returns them.
	Collect() ([]*Record, error)
	// Single returns one and only one record from the stream.
	// If the result stream contains zero or more than one records, error is returned.
	Single() (*Record, error)
	// Consume discards all remaining records and returns the summary information
	// about the statement execution.
	Consume() (ResultSummary, error)
}

// deprecated: use resultWithContext instead
type result struct {
	delegate ResultWithContext
}

func (r *result) Keys() ([]string, error) {
	return r.delegate.Keys()
}

func (r *result) Next() bool {
	return r.delegate.Next(context.Background())
}

func (r *result) NextRecord(out **Record) bool {
	return r.delegate.NextRecord(context.Background(), out)
}

func (r *result) PeekRecord(out **Record) bool {
	return r.delegate.PeekRecord(context.Background(), out)
}

func (r *result) Record() *Record {
	return r.delegate.Record()
}

func (r *result) Err() error {
	return r.delegate.Err()
}

func (r *result) Collect() ([]*Record, error) {
	return r.delegate.Collect(context.Background())
}

func (r *result) Single() (*Record, error) {
	return r.delegate.Single(context.Background())
}

func (r *result) Consume() (ResultSummary, error) {
	return r.delegate.Consume(context.Background())
}
