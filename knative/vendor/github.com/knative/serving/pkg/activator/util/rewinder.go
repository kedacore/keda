/*
Copyright 2018 The Knative Authors
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

package util

import (
	"bytes"
	"io"
	"sync"
)

type rewinder struct {
	sync.Mutex
	original io.ReadCloser
	current  io.Reader
	next     io.ReadWriter
	eof      bool
}

// NewRewinder wraps a single-use `ReadCloser` into a `ReadCloser`
// that can be read multiple times
func NewRewinder(rc io.ReadCloser) io.ReadCloser {
	r := &rewinder{original: rc}

	r.next = new(bytes.Buffer)
	r.current = r.original

	return r
}

func (r *rewinder) Read(b []byte) (int, error) {
	r.Lock()
	defer r.Unlock()

	// Buffer everything we read
	n, err := io.TeeReader(r.current, r.next).Read(b)

	// Keep track of when we reach the end
	if err == io.EOF {
		r.eof = true
	}

	return n, err
}

func (r *rewinder) Close() error {
	r.Lock()
	defer r.Unlock()

	// Start = what we've read already + what we haven't read yet
	start := io.MultiReader(r.next, r.current)

	// Close the original ReadCloser if we have read it to the end
	if r.eof && r.original != nil {

		if r.current == r.original {
			// Start = only what we've read already
			start = r.next
		}

		r.original.Close()
		r.original = nil
	}

	// Rewind back to the start
	r.next = new(bytes.Buffer)
	r.current = start

	return nil
}
