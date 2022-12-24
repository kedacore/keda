// Copyright (c) 2022 VMware, Inc. or its affiliates. All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package amqp091

type Logging interface {
	Printf(format string, v ...interface{})
}

var Logger Logging = NullLogger{}

// Enables logging using a custom Logging instance. Note that this is
// not thread safe and should be called at application start
func SetLogger(logger Logging) {
	Logger = logger
}

type NullLogger struct {
}

func (l NullLogger) Printf(format string, v ...interface{}) {
}
