// Copyright (c) 2021 VMware, Inc. or its affiliates. All Rights Reserved.
// Copyright (c) 2012-2021, Sean Treadway, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build gofuzz
// +build gofuzz

package amqp091

import "bytes"

func Fuzz(data []byte) int {
	r := reader{bytes.NewReader(data)}
	frame, err := r.ReadFrame()
	if err != nil {
		if frame != nil {
			panic("frame is not nil")
		}
		return 0
	}
	return 1
}
