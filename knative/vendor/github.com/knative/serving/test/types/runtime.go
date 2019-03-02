/*
Copyright 2019 The Knative Authors
 Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
     http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"net/http"
	"time"
)

// RuntimeInfo encapsulates both the host and request information.
type RuntimeInfo struct {
	// Request is information about the request.
	Request *RequestInfo `json:"request"`
	// Host is a set of host information.
	Host *HostInfo `json:"host"`
}

// RequestInfo encapsulates information about the request.
type RequestInfo struct {
	// Ts is the timestamp of when the request came in from the system time.
	Ts time.Time `json:"ts"`
	// URI is the request-target of the Request-Line.
	URI string `json:"uri"`
	// Host is the hostname on which the URL is sought.
	Host string `json:"host"`
	// Method is the method used for the request.
	Method string `json:"method"`
	// Headers is a Map of all headers set.
	Headers http.Header `json:"headers"`
}

// HostInfo contains information about the host environment.
type HostInfo struct {
	// Files is a map of file metadata.
	Files []*FileInfo `json:"files"`
	// EnvVars is a map of all environment variables set.
	EnvVars map[string]string `json:"envs"`
	// Cgroups is a list of cgroup information.
	Cgroups []*Cgroup `json:"cgroups"`
	// Mounts is a list of mounted volume information, or error.
	Mounts []*Mount `json:"mounts"`
}

// FileInfo contains the metadata for a given file.
type FileInfo struct {
	// Name is the full filename.
	Name string `json:"name"`
	// Size is the length in bytes for regular files; system-dependent for others.
	Size *int64 `json:"size,omitempty"`
	// Mode is the file mode bits.
	Mode string `json:"mode,omitempty"`
	// ModTime is the file last modified time
	ModTime time.Time `json:"modTime,omitempty"`
	// IsDir is true if the file is a directory.
	IsDir *bool `json:"isDir,omitempty"`
	// Error is the String representation of the error returned obtaining the information.
	Error string `json:"error,omitempty"`
}

// Cgroup contains the Cgroup value for a given setting.
type Cgroup struct {
	// Name is the full path name of the cgroup.
	Name string `json:"name"`
	// Value is the integer files in the cgroup file.
	Value *int `json:"value,omitempty"`
	// ReadOnly is true if the cgroup was not writable.
	ReadOnly *bool `json:"readOnly,omitempty"`
	// Error is the String representation of the error returned obtaining the information.
	Error string `json:"error,omitempty"`
}

// Mount contains information about a given mount.
type Mount struct {
	// Device is the device that is mounted
	Device string `json:"device,omitempty"`
	// Path is the location where the volume is mounted
	Path string `json:"path,omitempty"`
	// Type is the filesystem type (i.e. sysfs, proc, tmpfs, ext4, overlay, etc.)
	Type string `json:"type,omitempty"`
	// Options is the mount options set (i.e. rw, nosuid, relatime, etc.)
	Options []string `json:"options,omitempty"`
	// Error is the String representation of the error returned obtaining the information.
	Error string `json:"error,omitempty"`
}
