//
// DISCLAIMER
//
// Copyright 2017-2025 ArangoDB GmbH, Cologne, Germany
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

package driver

import (
	"context"
	"time"
)

// BackupMeta provides meta data of a backup
type BackupMeta struct {
	ID                      BackupID           `json:"id,omitempty"`
	Version                 string             `json:"version,omitempty"`
	DateTime                time.Time          `json:"datetime,omitempty"`
	NumberOfFiles           uint               `json:"nrFiles,omitempty"`
	NumberOfDBServers       uint               `json:"nrDBServers,omitempty"`
	SizeInBytes             uint64             `json:"sizeInBytes,omitempty"`
	PotentiallyInconsistent bool               `json:"potentiallyInconsistent,omitempty"`
	Available               bool               `json:"available,omitempty"`
	NumberOfPiecesPresent   uint               `json:"nrPiecesPresent,omitempty"`
	Keys                    []BackupMetaSha256 `json:"keys,omitempty"`
}

// BackupMetaSha256 backup sha details
type BackupMetaSha256 struct {
	SHA256 string `json:"sha256"`
}

// BackupRestoreOptions provides options for Restore
type BackupRestoreOptions struct {
	// do not version check when doing a restore (expert only)
	IgnoreVersion bool `json:"ignoreVersion,omitempty"`
}

// BackupListOptions provides options for List
type BackupListOptions struct {
	// Only receive meta data about a specific id
	ID BackupID `json:"id,omitempty"`
}

// BackupCreateOptions provides options for Create
type BackupCreateOptions struct {
	Label string `json:"label,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	// Deprecated: - since 3.10.10 it exists only for backwards compatibility
	AllowInconsistent bool `json:"allowInconsistent,omitempty"`
}

// BackupTransferStatus represents all possible states a transfer job can be in
type BackupTransferStatus string

const (
	TransferAcknowledged BackupTransferStatus = "ACK"
	TransferStarted      BackupTransferStatus = "STARTED"
	TransferCompleted    BackupTransferStatus = "COMPLETED"
	TransferFailed       BackupTransferStatus = "FAILED"
	TransferCancelled    BackupTransferStatus = "CANCELLED"
)

// BackupTransferReport provides progress information of a backup transfer job for a single dbserver
type BackupTransferReport struct {
	Status       BackupTransferStatus `json:"Status,omitempty"`
	Error        int                  `json:"Error,omitempty"`
	ErrorMessage string               `json:"ErrorMessage,omitempty"`
	Progress     struct {
		Total     int    `json:"Total,omitempty"`
		Done      int    `json:"Done,omitempty"`
		Timestamp string `json:"Timestamp,omitempty"`
	} `json:"Progress,omitempty"`
}

// BackupTransferProgressReport provides progress information for a backup transfer job
type BackupTransferProgressReport struct {
	BackupID  BackupID                        `json:"BackupID,omitempty"`
	Cancelled bool                            `json:"Cancelled,omitempty"`
	Timestamp string                          `json:"Timestamp,omitempty"`
	DBServers map[string]BackupTransferReport `json:"DBServers,omitempty"`
}

// BackupTransferJobID represents a Transfer (upload/download) job
type BackupTransferJobID string

// BackupID identifies a backup
type BackupID string

// ClientAdminBackup provides access to the Backup API via the Client interface
type ClientAdminBackup interface {
	Backup() ClientBackup
}

// BackupCreateResponse contains information about a newly created backup
type BackupCreateResponse struct {
	NumberOfFiles           uint
	NumberOfDBServers       uint
	SizeInBytes             uint64
	PotentiallyInconsistent bool
	CreationTime            time.Time
}

// ClientBackup provides access to server/cluster backup functions of an arangodb database server
// or an entire cluster of arangodb servers.
type ClientBackup interface {
	// Create creates a new backup and returns its id
	Create(ctx context.Context, opt *BackupCreateOptions) (BackupID, BackupCreateResponse, error)

	// Delete deletes the backup with given id
	Delete(ctx context.Context, id BackupID) error

	// Restore restores the backup with given id
	Restore(ctx context.Context, id BackupID, opt *BackupRestoreOptions) error

	// List returns meta data about some/all backups available
	List(ctx context.Context, opt *BackupListOptions) (map[BackupID]BackupMeta, error)

	// only enterprise version

	// Upload triggers an upload to the remote repository of backup with id using the given config
	// and returns the job id.
	Upload(ctx context.Context, id BackupID, remoteRepository string, config interface{}) (BackupTransferJobID, error)

	// Download triggers an download to the remote repository of backup with id using the given config
	// and returns the job id.
	Download(ctx context.Context, id BackupID, remoteRepository string, config interface{}) (BackupTransferJobID, error)

	// Progress returns the progress state of the given Transfer job
	Progress(ctx context.Context, job BackupTransferJobID) (BackupTransferProgressReport, error)

	// Abort aborts the Transfer job if possible
	Abort(ctx context.Context, job BackupTransferJobID) error
}
