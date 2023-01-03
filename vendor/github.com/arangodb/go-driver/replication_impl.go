//
// DISCLAIMER
//
// Copyright 2018-2021 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
// Author Tomasz Mielech
//

package driver

import (
	"context"
	"errors"
	"path"
	"strconv"
	"sync/atomic"
	"time"
)

// Content of the create batch resp
type batchMetadata struct {
	// ID of the batch
	ID string `json:"id"`
	// Last Tick reported by the server
	LastTickInt Tick `json:"lastTick,omitempty"`

	cl       *client
	serverID int64
	database string
	closed   int32
}

// ErrBatchClosed occurs when there is an attempt closing or prolonging closed batch
var ErrBatchClosed = errors.New("Batch already closed")

// CreateBatch creates a "batch" to prevent WAL file removal and to take a snapshot
func (c *client) CreateBatch(ctx context.Context, db Database, serverID int64, ttl time.Duration) (Batch, error) {
	req, err := c.conn.NewRequest("POST", path.Join("_db", db.Name(), "_api/replication/batch"))
	if err != nil {
		return nil, WithStack(err)
	}
	req = req.SetQuery("serverId", strconv.FormatInt(serverID, 10))
	params := struct {
		TTL float64 `json:"ttl"`
	}{TTL: ttl.Seconds()} // just use a default ttl value
	req, err = req.SetBody(params)
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var batch batchMetadata
	if err := resp.ParseBody("", &batch); err != nil {
		return nil, WithStack(err)
	}
	batch.cl = c
	batch.serverID = serverID
	batch.database = db.Name()
	return &batch, nil
}

// Get the inventory of a server containing all collections (with entire details) of a database.
func (c *client) DatabaseInventory(ctx context.Context, db Database) (DatabaseInventory, error) {
	req, err := c.conn.NewRequest("GET", path.Join("_db", db.Name(), "_api/replication/inventory"))
	if err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	var result DatabaseInventory
	if err := resp.ParseBody("", &result); err != nil {
		return DatabaseInventory{}, WithStack(err)
	}
	return result, nil
}

// BatchID reported by the server
// The receiver is pointer because this struct contains the field `closed` and it can not be copied
// because race detector will complain.
func (b *batchMetadata) BatchID() string {
	return b.ID
}

// LastTick reported by the server for this batch
// The receiver is pointer because this struct contains the field `closed` and it can not be copied
// because race detector will complain.
func (b *batchMetadata) LastTick() Tick {
	return b.LastTickInt
}

// Extend the lifetime of an existing batch on the server
func (b *batchMetadata) Extend(ctx context.Context, ttl time.Duration) error {
	if !atomic.CompareAndSwapInt32(&b.closed, 0, 0) {
		return WithStack(ErrBatchClosed)
	}

	req, err := b.cl.conn.NewRequest("PUT", path.Join("_db", b.database, "_api/replication/batch", b.ID))
	if err != nil {
		return WithStack(err)
	}
	req = req.SetQuery("serverId", strconv.FormatInt(b.serverID, 10))
	input := struct {
		TTL int64 `json:"ttl"`
	}{
		TTL: int64(ttl.Seconds()),
	}
	req, err = req.SetBody(input)
	if err != nil {
		return WithStack(err)
	}
	resp, err := b.cl.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(204); err != nil {
		return WithStack(err)
	}
	return nil
}

// Delete an existing dump batch
func (b *batchMetadata) Delete(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&b.closed, 0, 1) {
		return WithStack(ErrBatchClosed)
	}

	req, err := b.cl.conn.NewRequest("DELETE", path.Join("_db", b.database, "_api/replication/batch", b.ID))
	if err != nil {
		return WithStack(err)
	}
	resp, err := b.cl.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(204); err != nil {
		return WithStack(err)
	}
	return nil
}
