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
	"fmt"
)

// SingleTWithContext is like SingleT. It accepts a context.Context parameter
func SingleTWithContext[T any](ctx context.Context, result ResultWithContext, mapper func(*Record) (T, error)) (T, error) {
	single, err := result.Single(ctx)
	if err != nil {
		return *new(T), err
	}
	return mapper(single)
}

// SingleT is like Single but maps the single Record with the provided mapper.
// Deprecated: use SingleTWithContext instead (the entry point of context-aware
// APIs is NewDriverWithContext)
func SingleT[T any](result Result, mapper func(*Record) (T, error)) (T, error) {
	single, err := result.Single()
	if err != nil {
		return *new(T), err
	}
	return mapper(single)
}

// CollectTWithContext is like CollectT. It accepts a context.Context parameter
func CollectTWithContext[T any](ctx context.Context, result ResultWithContext, mapper func(*Record) (T, error)) ([]T, error) {
	records, err := result.Collect(ctx)
	if err != nil {
		return nil, err
	}
	return mapAll(records, mapper)
}

// CollectT is like Collect but maps each record with the provided mapper.
// Deprecated: use CollectTWithContext instead (the entry point of context-aware
// APIs is NewDriverWithContext)
func CollectT[T any](result Result, mapper func(*Record) (T, error)) ([]T, error) {
	records, err := result.Collect()
	if err != nil {
		return nil, err
	}
	return mapAll(records, mapper)
}

// Single returns one and only one record from the result stream. Any error passed in
// or reported while navigating the result stream is returned without any conversion.
// If the result stream contains zero or more than one records error is returned.
//
//	record, err := neo4j.Single(session.Run(...))
func Single(result Result, err error) (*Record, error) {
	if err != nil {
		return nil, err
	}
	return result.Single()
}

// Collect behaves similarly to CollectWithContext
//
//	records, err := neo4j.Collect(session.Run(...))
//
// Deprecated: use CollectWithContext instead (the entry point of context-aware
// APIs is NewDriverWithContext)
func Collect(result Result, err error) ([]*Record, error) {
	if err != nil {
		return nil, err
	}
	return result.Collect()
}

// CollectWithContext loops through the result stream, collects records into a slice and returns the
// resulting slice. Any error passed in or reported while navigating the result stream is
// returned without any conversion.
//
//	result, err := session.Run(...)
//	records, err := neo4j.CollectWithContext(ctx, result, err)
//
// Note, you cannot write neo4j.CollectWithContext(ctx, session.Run(...)) due to Go limitations
func CollectWithContext(ctx context.Context, result ResultWithContext, err error) ([]*Record, error) {
	if err != nil {
		return nil, err
	}
	return result.Collect(ctx)
}

// AsRecords passes any existing error or casts from to a slice of records.
// Use in combination with Collect and transactional functions:
//
//	records, err := neo4j.AsRecords(session.ExecuteRead(func (tx neo4j.Transaction) {
//	    return neo4j.Collect(tx.Run(...))
//	}))
func AsRecords(from any, err error) ([]*Record, error) {
	if err != nil {
		return nil, err
	}
	recs, ok := from.([]*Record)
	if !ok {
		return nil, &UsageError{
			Message: fmt.Sprintf("Expected type []*Record, not %T", from),
		}
	}
	return recs, nil
}

// AsRecord passes any existing error or casts from to a record.
// Use in combination with Single and transactional functions:
//
//	record, err := neo4j.AsRecord(session.ExecuteRead(func (tx neo4j.Transaction) {
//	    return neo4j.Single(tx.Run(...))
//	}))
func AsRecord(from any, err error) (*Record, error) {
	if err != nil {
		return nil, err
	}
	rec, ok := from.(*Record)
	if !ok {
		return nil, &UsageError{
			Message: fmt.Sprintf("Expected type *Record, not %T", from),
		}
	}
	return rec, nil
}

func mapAll[T any](records []*Record, mapper func(*Record) (T, error)) ([]T, error) {
	results := make([]T, len(records))
	for i, record := range records {
		mappedRecord, err := mapper(record)
		if err != nil {
			return nil, err
		}
		results[i] = mappedRecord
	}
	return results, nil
}
