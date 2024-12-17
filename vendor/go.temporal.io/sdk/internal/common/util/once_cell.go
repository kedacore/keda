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

package util

import "sync"

// A OnceCell attempts to match the semantics of Rust's `OnceCell`, but only stores strings, since that's what's needed
// at the moment. Could be changed to use interface{} to be generic.
type OnceCell struct {
	// Ensures we only call the fetcher one time
	once sync.Once
	// Stores the result of calling fetcher
	value   string
	fetcher func() string
}

// Get fetches the value in the cell, calling the fetcher function if it has not yet been called
func (oc *OnceCell) Get() string {
	oc.once.Do(func() {
		res := oc.fetcher()
		oc.value = res
	})
	return oc.value
}

// PopulatedOnceCell creates an already-initialized cell
func PopulatedOnceCell(value string) OnceCell {
	return OnceCell{
		once:  sync.Once{},
		value: value,
		fetcher: func() string {
			return value
		},
	}
}

type fetcher func() string

// LazyOnceCell creates a cell with no initial value, the provided function will be called once and only once the first
// time OnceCell.Get is called
func LazyOnceCell(fetcher fetcher) OnceCell {
	return OnceCell{
		once:    sync.Once{},
		value:   "",
		fetcher: fetcher,
	}
}
