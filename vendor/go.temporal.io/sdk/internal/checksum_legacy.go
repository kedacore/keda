// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

//go:build !go1.24

package internal

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// callers MUST hold binaryChecksumLock before calling
func initBinaryChecksumLocked() error {
	if len(binaryChecksum) > 0 {
		return nil
	}

	exec, err := os.Executable()
	if err != nil {
		return err
	}

	f, err := os.Open(exec)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() // error is unimportant as it is read-only
	}()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	checksum := h.Sum(nil)
	binaryChecksum = hex.EncodeToString(checksum[:])

	return nil
}
