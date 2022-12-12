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

package neo4j

import (
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

// A router implementation that never routes
type directRouter struct {
	address string
}

func (r *directRouter) InvalidateWriter(context.Context, string, string) error {
	return nil
}

func (r *directRouter) InvalidateReader(context.Context, string, string) error {
	return nil
}

func (r *directRouter) Readers(context.Context, func(context.Context) ([]string, error), string, log.BoltLogger) ([]string, error) {
	return []string{r.address}, nil
}

func (r *directRouter) Writers(context.Context, func(context.Context) ([]string, error), string, log.BoltLogger) ([]string, error) {
	return []string{r.address}, nil
}

func (r *directRouter) GetNameOfDefaultDatabase(context.Context, []string, string, log.BoltLogger) (string, error) {
	return db.DefaultDatabase, nil
}

func (r *directRouter) Invalidate(context.Context, string) error {
	return nil
}

func (r *directRouter) CleanUp(context.Context) error {
	return nil
}
