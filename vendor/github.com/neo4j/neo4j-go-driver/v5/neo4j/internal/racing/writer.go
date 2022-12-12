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
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package racing

import (
	"context"
	"io"
	"time"
)

type RacingWriter interface {
	Write(ctx context.Context, bytes []byte) (int, error)
}

func NewRacingWriter(writer io.Writer) RacingWriter {
	return &racingWriter{writer: writer}
}

type racingWriter struct {
	writer io.Writer
}

type ioResult struct {
	n   int
	err error
}

func (rw *racingWriter) Write(ctx context.Context, bytes []byte) (int, error) {
	deadline, hasDeadline := ctx.Deadline()
	err := ctx.Err()
	switch {
	case !hasDeadline && err == nil:
		return rw.writer.Write(bytes)
	case deadline.Before(time.Now()) || err != nil:
		return 0, err
	}
	resultChan := make(chan *ioResult, 1)
	go func() {
		defer close(resultChan)
		n, err := rw.writer.Write(bytes)
		resultChan <- &ioResult{
			n:   n,
			err: err,
		}
	}()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case result := <-resultChan:
		return result.n, result.err
	}
}
