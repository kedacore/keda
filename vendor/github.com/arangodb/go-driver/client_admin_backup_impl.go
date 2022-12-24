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
	"time"
)

type clientBackup struct {
	conn Connection
}

func (c *client) Backup() ClientBackup {
	return &clientBackup{
		conn: c.conn,
	}
}

// Create creates a new backup and returns its id
func (c *clientBackup) Create(ctx context.Context, opt *BackupCreateOptions) (BackupID, BackupCreateResponse, error) {
	req, err := c.conn.NewRequest("POST", "_admin/backup/create")
	if err != nil {
		return "", BackupCreateResponse{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	if opt != nil {
		body := struct {
			Label             string  `json:"label,omitempty"`
			AllowInconsistent bool    `json:"allowInconsistent,omitempty"`
			Timeout           float64 `json:"timeout,omitempty"`
		}{
			Label:             opt.Label,
			AllowInconsistent: opt.AllowInconsistent,
			Timeout:           opt.Timeout.Seconds(),
		}
		req, err = req.SetBody(body)
		if err != nil {
			return "", BackupCreateResponse{}, WithStack(err)
		}
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", BackupCreateResponse{}, WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return "", BackupCreateResponse{}, WithStack(err)
	}
	var result struct {
		ID                      BackupID  `json:"id,omitempty"`
		PotentiallyInconsistent bool      `json:"potentiallyInconsistent,omitempty"`
		NumberOfFiles           uint      `json:"nrFiles,omitempty"`
		NumberOfDBServers       uint      `json:"nrDBServers,omitempty"`
		SizeInBytes             uint64    `json:"sizeInBytes,omitempty"`
		CreationTime            time.Time `json:"datetime,omitempty"`
	}
	if err := resp.ParseBody("result", &result); err != nil {
		return "", BackupCreateResponse{}, WithStack(err)
	}
	return result.ID, BackupCreateResponse{
		PotentiallyInconsistent: result.PotentiallyInconsistent,
		NumberOfFiles:           result.NumberOfFiles,
		NumberOfDBServers:       result.NumberOfDBServers,
		SizeInBytes:             result.SizeInBytes,
		CreationTime:            result.CreationTime,
	}, nil
}

// Delete deletes the backup with given id
func (c *clientBackup) Delete(ctx context.Context, id BackupID) error {
	req, err := c.conn.NewRequest("POST", "_admin/backup/delete")
	if err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	body := struct {
		ID BackupID `json:"id,omitempty"`
	}{
		ID: id,
	}
	req, err = req.SetBody(body)
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// Restore restores the backup with given id
func (c *clientBackup) Restore(ctx context.Context, id BackupID, opt *BackupRestoreOptions) error {
	req, err := c.conn.NewRequest("POST", "_admin/backup/restore")
	if err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	body := struct {
		ID            BackupID `json:"id,omitempty"`
		IgnoreVersion bool     `json:"ignoreVersion,omitempty"`
	}{
		ID: id,
	}
	if opt != nil {
		body.IgnoreVersion = opt.IgnoreVersion
	}
	req, err = req.SetBody(body)
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	// THIS SHOULD BE 202 ACCEPTED and not OK, because it is not completed when returns (at least for single server)
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// List returns meta data about some/all backups available
func (c *clientBackup) List(ctx context.Context, opt *BackupListOptions) (map[BackupID]BackupMeta, error) {
	req, err := c.conn.NewRequest("POST", "_admin/backup/list")
	if err != nil {
		return nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	if opt != nil {
		req, err = req.SetBody(opt)
		if err != nil {
			return nil, WithStack(err)
		}
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var result struct {
		List map[BackupID]BackupMeta `json:"list,omitempty"`
	}
	if err := resp.ParseBody("result", &result); err != nil {
		return nil, WithStack(err)
	}
	return result.List, nil
}

// Upload triggers an upload to the remote repository of backup with id using the given config
// and returns the job id.
func (c *clientBackup) Upload(ctx context.Context, id BackupID, remoteRepository string, config interface{}) (BackupTransferJobID, error) {
	req, err := c.conn.NewRequest("POST", "_admin/backup/upload")
	if err != nil {
		return "", WithStack(err)
	}
	applyContextSettings(ctx, req)
	body := struct {
		ID         BackupID    `json:"id,omitempty"`
		RemoteRepo string      `json:"remoteRepository,omitempty"`
		Config     interface{} `json:"config,omitempty"`
	}{
		ID:         id,
		RemoteRepo: remoteRepository,
		Config:     config,
	}
	req, err = req.SetBody(body)
	if err != nil {
		return "", WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(202); err != nil {
		return "", WithStack(err)
	}
	var result struct {
		UploadID BackupTransferJobID `json:"uploadId,omitempty"`
	}
	if err := resp.ParseBody("result", &result); err != nil {
		return "", WithStack(err)
	}
	return result.UploadID, nil
}

// Download triggers an download to the remote repository of backup with id using the given config
// and returns the job id.
func (c *clientBackup) Download(ctx context.Context, id BackupID, remoteRepository string, config interface{}) (BackupTransferJobID, error) {
	req, err := c.conn.NewRequest("POST", "_admin/backup/download")
	if err != nil {
		return "", WithStack(err)
	}
	applyContextSettings(ctx, req)
	body := struct {
		ID         BackupID    `json:"id,omitempty"`
		RemoteRepo string      `json:"remoteRepository,omitempty"`
		Config     interface{} `json:"config,omitempty"`
	}{
		ID:         id,
		RemoteRepo: remoteRepository,
		Config:     config,
	}

	req, err = req.SetBody(body)
	if err != nil {
		return "", WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(202); err != nil {
		return "", WithStack(err)
	}
	var result struct {
		DownloadID BackupTransferJobID `json:"downloadId,omitempty"`
	}
	if err := resp.ParseBody("result", &result); err != nil {
		return "", WithStack(err)
	}
	return result.DownloadID, nil
}

// Progress returns the progress state of the given Transfer job
func (c *clientBackup) Progress(ctx context.Context, job BackupTransferJobID) (result BackupTransferProgressReport, error error) {
	req, err := c.conn.NewRequest("POST", "_admin/backup/upload")
	if err != nil {
		return BackupTransferProgressReport{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	body := struct {
		ID BackupTransferJobID `json:"uploadId,omitempty"`
	}{
		ID: job,
	}
	req, err = req.SetBody(body)
	if err != nil {
		return BackupTransferProgressReport{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return BackupTransferProgressReport{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return BackupTransferProgressReport{}, WithStack(err)
	}
	if err := resp.ParseBody("result", &result); err != nil {
		return BackupTransferProgressReport{}, WithStack(err)
	}
	return result, nil
}

// Abort aborts the Transfer job if possible
func (c *clientBackup) Abort(ctx context.Context, job BackupTransferJobID) error {
	req, err := c.conn.NewRequest("POST", "_admin/backup/upload")
	if err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	body := struct {
		ID    BackupTransferJobID `json:"uploadId,omitempty"`
		Abort bool                `json:"abort,omitempty"`
	}{
		ID:    job,
		Abort: true,
	}
	req, err = req.SetBody(body)
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(202); err != nil {
		return WithStack(err)
	}
	return nil
}
