// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build plan9

package filelock

import (
	"os"
)

type lockType int8

const (
	readLock = iota + 1
	writeLock
)

func lock(f File, lt lockType) error {
	return &os.PathError{
		Op:   lt.String(),
		Path: f.Name(),
		Err:  ErrNotSupported,
	}
}

func unlock(f File) error {
	return &os.PathError{
		Op:   "Unlock",
		Path: f.Name(),
		Err:  ErrNotSupported,
	}
}

func isNotSupported(err error) bool {
	return err == ErrNotSupported
}
