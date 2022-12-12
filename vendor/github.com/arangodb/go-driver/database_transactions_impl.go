//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
// Author Lars Maier
//

package driver

import (
	"context"
	"path"
)

type beginTransactionRequest struct {
	WaitForSync        bool                   `json:"waitForSync,omitempty"`
	AllowImplicit      bool                   `json:"allowImplicit,omitempty"`
	LockTimeout        float64                `json:"lockTimeout,omitempty"`
	MaxTransactionSize uint64                 `json:"maxTransactionSize,omitempty"`
	Collections        TransactionCollections `json:"collections,omitempty"`
}

func (d *database) BeginTransaction(ctx context.Context, cols TransactionCollections, opts *BeginTransactionOptions) (TransactionID, error) {
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/transaction/begin"))
	if err != nil {
		return "", WithStack(err)
	}
	var reqBody beginTransactionRequest
	if opts != nil {
		reqBody.WaitForSync = opts.WaitForSync
		reqBody.AllowImplicit = opts.AllowImplicit
		reqBody.LockTimeout = opts.LockTimeout.Seconds()
	}
	reqBody.Collections = cols
	if _, err := req.SetBody(reqBody); err != nil {
		return "", WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return "", WithStack(err)
	}
	var result struct {
		TransactionID TransactionID `json:"id,omitempty"`
	}
	if err := resp.ParseBody("result", &result); err != nil {
		return "", WithStack(err)
	}
	return result.TransactionID, nil
}

func (d *database) requestForTransaction(ctx context.Context, tid TransactionID, method string) (TransactionStatusRecord, error) {
	req, err := d.conn.NewRequest(method, path.Join(d.relPath(), "_api/transaction/", string(tid)))
	if err != nil {
		return TransactionStatusRecord{}, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return TransactionStatusRecord{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return TransactionStatusRecord{}, WithStack(err)
	}
	var result TransactionStatusRecord
	if err := resp.ParseBody("result", &result); err != nil {
		return TransactionStatusRecord{}, WithStack(err)
	}
	return result, nil
}

func (d *database) CommitTransaction(ctx context.Context, tid TransactionID, opts *CommitTransactionOptions) error {
	_, err := d.requestForTransaction(ctx, tid, "PUT")
	return err
}

func (d *database) AbortTransaction(ctx context.Context, tid TransactionID, opts *AbortTransactionOptions) error {
	_, err := d.requestForTransaction(ctx, tid, "DELETE")
	return err
}

func (d *database) TransactionStatus(ctx context.Context, tid TransactionID) (TransactionStatusRecord, error) {
	return d.requestForTransaction(ctx, tid, "GET")
}
